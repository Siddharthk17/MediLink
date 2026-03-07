// Package handlers provides HTTP handlers for FHIR resources.
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Siddharthk17/MediLink/internal/auth"
	"github.com/Siddharthk17/MediLink/internal/fhir/repository"
	"github.com/Siddharthk17/MediLink/internal/fhir/services"
	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// PatientHandler handles HTTP requests for Patient resources.
type PatientHandler struct {
	service services.PatientService
}

// NewPatientHandler creates a new PatientHandler.
func NewPatientHandler(service services.PatientService) *PatientHandler {
	return &PatientHandler{service: service}
}

// CreatePatient handles POST /fhir/R4/Patient.
func (h *PatientHandler) CreatePatient(c *gin.Context) {
	actorID := auth.GetActorID(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("failed to read request body"))
		return
	}

	if !json.Valid(body) {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid JSON in request body"))
		return
	}

	req := services.CreatePatientRequest{Data: body}
	resp, svcErr := h.service.CreatePatient(c.Request.Context(), req, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	// Extract resource ID for Location header
	var m map[string]interface{}
	json.Unmarshal(resp.Data, &m)
	resourceID, _ := m["id"].(string)

	c.Header("Location", "/fhir/R4/Patient/"+resourceID)
	c.Data(http.StatusCreated, "application/fhir+json; charset=utf-8", resp.Data)
}

// GetPatient handles GET /fhir/R4/Patient/:id.
func (h *PatientHandler) GetPatient(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	resp, svcErr := h.service.GetPatient(c.Request.Context(), resourceID, actorID)
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

// UpdatePatient handles PUT /fhir/R4/Patient/:id.
func (h *PatientHandler) UpdatePatient(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("failed to read request body"))
		return
	}

	if !json.Valid(body) {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid JSON in request body"))
		return
	}

	req := services.UpdatePatientRequest{Data: body}
	resp, svcErr := h.service.UpdatePatient(c.Request.Context(), resourceID, req, actorID)
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

// DeletePatient handles DELETE /fhir/R4/Patient/:id.
func (h *PatientHandler) DeletePatient(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	svcErr := h.service.DeletePatient(c.Request.Context(), resourceID, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchPatients handles GET /fhir/R4/Patient.
func (h *PatientHandler) SearchPatients(c *gin.Context) {
	actorID := auth.GetActorID(c)

	params := repository.PatientSearchParams{
		Family:     c.Query("family"),
		Given:      c.Query("given"),
		Name:       c.Query("name"),
		Gender:     c.Query("gender"),
		Identifier: c.Query("identifier"),
		ID:         c.Query("_id"),
	}

	// Parse birthdate with optional prefix
	birthdate := c.Query("birthdate")
	if birthdate != "" {
		if len(birthdate) > 2 {
			prefix := birthdate[:2]
			switch prefix {
			case "lt", "gt", "le", "ge":
				params.BirthDateOp = prefix
				params.BirthDate = birthdate[2:]
			default:
				params.BirthDateOp = "eq"
				params.BirthDate = birthdate
			}
		} else {
			params.BirthDateOp = "eq"
			params.BirthDate = birthdate
		}
	}

	// Parse active
	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		params.Active = &active
	}

	// Parse pagination
	if countStr := c.Query("_count"); countStr != "" {
		if count, err := strconv.Atoi(countStr); err == nil {
			params.Count = count
		}
	}
	if params.Count <= 0 {
		params.Count = 20
	}

	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = offset
		}
	}

	baseURL := getBaseURL(c)
	resp, svcErr := h.service.SearchPatients(c.Request.Context(), params, actorID, baseURL)
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

// GetPatientHistory handles GET /fhir/R4/Patient/:id/_history.
func (h *PatientHandler) GetPatientHistory(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	count := 20
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	resp, svcErr := h.service.GetPatientHistory(c.Request.Context(), resourceID, actorID, count, offset)
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

// GetPatientVersion handles GET /fhir/R4/Patient/:id/_history/:vid.
func (h *PatientHandler) GetPatientVersion(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")
	vidStr := c.Param("vid")

	vid, err := strconv.Atoi(vidStr)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid version ID"))
		return
	}

	resp, svcErr := h.service.GetPatientVersion(c.Request.Context(), resourceID, vid, actorID)
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

// writeFHIRError writes a FHIR OperationOutcome error response.
func writeFHIRError(c *gin.Context, fhirErr *fhirerrors.FHIRError) {
	c.Data(fhirErr.StatusCode, "application/fhir+json; charset=utf-8", fhirErr.JSON())
}

// getBaseURL derives the base URL from the request.
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + c.Request.Host
}
