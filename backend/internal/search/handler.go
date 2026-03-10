package search

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// SearchHandler exposes the GET /search endpoint.
type SearchHandler struct {
	svc    *SearchService
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewSearchHandler creates a SearchHandler.
func NewSearchHandler(svc *SearchService, db *sqlx.DB, logger zerolog.Logger) *SearchHandler {
	return &SearchHandler{
		svc:    svc,
		db:     db,
		logger: logger,
	}
}

// UnifiedSearch handles GET /search with role-aware, consent-checked full-text search.
func (h *SearchHandler) UnifiedSearch(c *gin.Context) {
	start := time.Now()

	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity":    "error",
				"code":        "required",
				"diagnostics": "query parameter 'q' is required",
			}},
		})
		return
	}

	var types []string
	if t := c.Query("type"); t != "" {
		types = strings.Split(t, ",")
	}

	patientRef := c.Query("patient")

	var dateFrom, dateTo *time.Time
	for _, d := range c.QueryArray("date") {
		if strings.HasPrefix(d, "ge") {
			if parsed, err := time.Parse("2006-01-02", d[2:]); err == nil {
				dateFrom = &parsed
			}
		} else if strings.HasPrefix(d, "le") {
			if parsed, err := time.Parse("2006-01-02", d[2:]); err == nil {
				dateTo = &parsed
			}
		}
	}

	count := 20
	if v, err := strconv.Atoi(c.Query("_count")); err == nil && v > 0 {
		count = v
	}
	if count > 50 {
		count = 50
	}

	offset := 0
	if v, err := strconv.Atoi(c.Query("_offset")); err == nil && v >= 0 {
		offset = v
	}

	actorID := c.GetString("actor_id")
	actorRole := c.GetString("actor_role")

	// If consent middleware already resolved a forced patient ref, use it
	forcedRef := c.GetString("forced_patient_ref")

	// Resolve FHIR patient ID for patient-role actors.
	var patientFHIRID string
	if actorRole == "patient" {
		if forcedRef != "" {
			patientFHIRID = strings.TrimPrefix(forcedRef, "Patient/")
		} else {
			uid, err := uuid.Parse(actorID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid actor ID"})
				return
			}
			err = h.db.GetContext(c.Request.Context(), &patientFHIRID,
				"SELECT COALESCE(fhir_patient_id, '') FROM users WHERE id = $1", uid)
			if err != nil || patientFHIRID == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"resourceType": "OperationOutcome",
					"issue": []gin.H{{
						"severity":    "error",
						"code":        "forbidden",
						"diagnostics": "patient FHIR ID not found",
					}},
				})
				return
			}
		}
	}

	params := SearchParams{
		Query:         q,
		Types:         types,
		PatientRef:    patientRef,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		Count:         count,
		Offset:        offset,
		ActorID:       actorID,
		ActorRole:     actorRole,
		PatientFHIRID: patientFHIRID,
	}

	resp, err := h.svc.Search(c.Request.Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "forbidden",
					"diagnostics": err.Error(),
				}},
			})
		case errors.Is(err, ErrPatientRequired):
			c.JSON(http.StatusBadRequest, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "required",
					"diagnostics": err.Error(),
				}},
			})
		case errors.Is(err, ErrConsentRequired):
			c.JSON(http.StatusForbidden, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "forbidden",
					"diagnostics": err.Error(),
				}},
			})
		default:
			h.logger.Error().Err(err).Str("query", q).Msg("search failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "exception",
					"diagnostics": "search failed",
				}},
			})
		}
		return
	}

	durationMS := time.Since(start).Milliseconds()
	totalResults := resp.Total

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.db.ExecContext(ctx,
			`INSERT INTO search_queries (id, actor_id, actor_role, query_text, resource_types, results_count, duration_ms)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New(), actorID, actorRole, q, pq.Array(types), totalResults, durationMS,
		)
	}()

	c.JSON(http.StatusOK, resp.Bundle)
}
