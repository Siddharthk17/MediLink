package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Siddharthk17/medilink/internal/auth"
	"github.com/Siddharthk17/medilink/internal/fhir/services"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// TimelineHandler handles the patient timeline endpoint.
type TimelineHandler struct {
	service *services.TimelineService
}

// NewTimelineHandler creates a new TimelineHandler.
func NewTimelineHandler(service *services.TimelineService) *TimelineHandler {
	return &TimelineHandler{service: service}
}

// GetTimeline handles GET /fhir/R4/Patient/:id/$timeline.
func (h *TimelineHandler) GetTimeline(c *gin.Context) {
	actorID := auth.GetActorID(c)
	patientID := c.Param("id")

	params := services.TimelineParams{
		PatientID: patientID,
		Since:     c.Query("_since"),
	}

	// Parse resource types filter
	if typeStr := c.Query("_type"); typeStr != "" {
		params.ResourceTypes = strings.Split(typeStr, ",")
	}

	// Parse pagination
	params.Count = 50
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			params.Count = parsed
		}
	}

	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			params.Offset = parsed
		}
	}

	resp, svcErr := h.service.GetTimeline(c.Request.Context(), params, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// LabTrendsHandler handles the lab trends endpoint.
type LabTrendsHandler struct {
	service *services.LabTrendsService
}

// NewLabTrendsHandler creates a new LabTrendsHandler.
func NewLabTrendsHandler(service *services.LabTrendsService) *LabTrendsHandler {
	return &LabTrendsHandler{service: service}
}

// GetLabTrends handles GET /fhir/R4/Observation/$lab-trends.
func (h *LabTrendsHandler) GetLabTrends(c *gin.Context) {
	actorID := auth.GetActorID(c)

	params := services.LabTrendsParams{
		PatientRef: c.Query("patient"),
		Code:       c.Query("code"),
		Since:      c.Query("_since"),
	}

	params.Count = 50
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			params.Count = parsed
		}
	}

	resp, svcErr := h.service.GetLabTrends(c.Request.Context(), params, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}
