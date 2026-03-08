package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	pkgsearch "github.com/Siddharthk17/MediLink/pkg/search"
)

var (
	ErrForbidden       = errors.New("search access denied")
	ErrPatientRequired = errors.New("patient parameter required for physician role")
	ErrConsentRequired = errors.New("no active consent for specified patient")
)

// SearchParams holds all parameters for a unified search request.
type SearchParams struct {
	Query         string
	Types         []string
	PatientRef    string
	DateFrom      *time.Time
	DateTo        *time.Time
	Count         int
	Offset        int
	ActorID       string
	ActorRole     string
	PatientFHIRID string // pre-resolved for patient role
}

// SearchResponse wraps the FHIR Bundle and result metadata.
type SearchResponse struct {
	Bundle map[string]interface{}
	Total  int
}

// SearchService implements consent-aware, role-scoped search.
type SearchService struct {
	es     pkgsearch.SearchClient
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewSearchService creates a SearchService.
func NewSearchService(es pkgsearch.SearchClient, db *sqlx.DB, logger zerolog.Logger) *SearchService {
	return &SearchService{
		es:     es,
		db:     db,
		logger: logger,
	}
}

// Search validates access rules, queries ES, and returns a FHIR Bundle.
func (s *SearchService) Search(ctx context.Context, params SearchParams) (*SearchResponse, error) {
	filters := SearchFilters{
		ActorRole: params.ActorRole,
		DateFrom:  params.DateFrom,
		DateTo:    params.DateTo,
	}

	switch params.ActorRole {
	case "researcher":
		return nil, ErrForbidden
	case "patient":
		if params.PatientFHIRID == "" {
			return nil, fmt.Errorf("patient FHIR ID not found: %w", ErrForbidden)
		}
		filters.PatientFHIRID = params.PatientFHIRID
	case "physician":
		if params.PatientRef == "" {
			return nil, ErrPatientRequired
		}
		patientFHIRID := strings.TrimPrefix(params.PatientRef, "Patient/")
		providerUUID, err := uuid.Parse(params.ActorID)
		if err != nil {
			return nil, fmt.Errorf("invalid actor ID: %w", err)
		}
		var hasConsent bool
		err = s.db.GetContext(ctx, &hasConsent,
			`SELECT EXISTS(
				SELECT 1 FROM consents c
				JOIN users u ON c.patient_id = u.id
				WHERE c.provider_id = $1 AND u.fhir_patient_id = $2
				AND c.revoked_at IS NULL AND (c.expires_at IS NULL OR c.expires_at > NOW())
			)`, providerUUID, patientFHIRID)
		if err != nil {
			return nil, fmt.Errorf("consent check failed: %w", err)
		}
		if !hasConsent {
			return nil, ErrConsentRequired
		}
		filters.PatientRef = params.PatientRef
	case "admin":
		if params.PatientRef != "" {
			filters.PatientRef = params.PatientRef
		}
	}

	query := BuildQuery(params.Query, params.Count, params.Offset, filters)
	indices := IndicesForTypes(params.Types)
	if len(indices) == 0 {
		return &SearchResponse{
			Bundle: BuildSearchBundle(nil, 0, params.Query),
			Total:  0,
		}, nil
	}

	esResp, err := s.es.SearchMultiIndex(ctx, indices, query)
	if err != nil {
		return nil, fmt.Errorf("search execution failed: %w", err)
	}

	hits := make([]SearchHit, 0, len(esResp.Hits))
	for _, h := range esResp.Hits {
		rt := ""
		if v, ok := h.Source["resourceType"].(string); ok {
			rt = v
		}
		hits = append(hits, SearchHit{
			ResourceType: rt,
			ResourceID:   h.ID,
			Score:        h.Score,
			Source:       h.Source,
			Highlights:   h.Highlight,
		})
	}

	bundle := BuildSearchBundle(hits, esResp.Total, params.Query)

	return &SearchResponse{
		Bundle: bundle,
		Total:  esResp.Total,
	}, nil
}
