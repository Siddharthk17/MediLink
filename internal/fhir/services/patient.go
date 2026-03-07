// Package services provides FHIR resource business logic.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/medilink/internal/audit"
	"github.com/Siddharthk17/medilink/internal/fhir/repository"
	"github.com/Siddharthk17/medilink/internal/fhir/validator"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// PatientResponse represents the response for a single Patient resource.
type PatientResponse struct {
	Data json.RawMessage
}

// BundleResponse represents a FHIR Bundle search response.
type BundleResponse struct {
	Data json.RawMessage
}

// CreatePatientRequest is the request payload for creating a Patient.
type CreatePatientRequest struct {
	Data json.RawMessage
}

// UpdatePatientRequest is the request payload for updating a Patient.
type UpdatePatientRequest struct {
	Data json.RawMessage
}

// PatientService defines the interface for Patient business logic.
type PatientService interface {
	// CreatePatient validates and creates a new Patient resource.
	CreatePatient(ctx context.Context, req CreatePatientRequest, actorID uuid.UUID) (*PatientResponse, error)
	// GetPatient retrieves a Patient by resource ID.
	GetPatient(ctx context.Context, resourceID string, actorID uuid.UUID) (*PatientResponse, error)
	// GetPatientVersion retrieves a specific version of a Patient.
	GetPatientVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*PatientResponse, error)
	// GetPatientHistory retrieves all versions of a Patient as a FHIR Bundle.
	GetPatientHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*BundleResponse, error)
	// UpdatePatient validates and updates an existing Patient resource.
	UpdatePatient(ctx context.Context, resourceID string, req UpdatePatientRequest, actorID uuid.UUID) (*PatientResponse, error)
	// DeletePatient soft-deletes a Patient resource.
	DeletePatient(ctx context.Context, resourceID string, actorID uuid.UUID) error
	// SearchPatients searches for Patients and returns a FHIR Bundle.
	SearchPatients(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*BundleResponse, error)
}

// PatientServiceImpl implements PatientService.
type PatientServiceImpl struct {
	repo      repository.PatientRepository
	validator *validator.FHIRValidator
	audit     audit.AuditLogger
}

// NewPatientService creates a new PatientService.
func NewPatientService(repo repository.PatientRepository, v *validator.FHIRValidator, al audit.AuditLogger) *PatientServiceImpl {
	return &PatientServiceImpl{
		repo:      repo,
		validator: v,
		audit:     al,
	}
}

// CreatePatient validates and creates a new Patient resource.
func (s *PatientServiceImpl) CreatePatient(ctx context.Context, req CreatePatientRequest, actorID uuid.UUID) (*PatientResponse, error) {
	// Validate the Patient resource
	if fhirErr := s.validator.ValidatePatient(req.Data, ""); fhirErr != nil {
		s.audit.LogAsync(audit.AuditEntry{
			UserID:       &actorID,
			ResourceType: "Patient",
			Action:       "create",
			Success:      false,
			StatusCode:   fhirErr.StatusCode,
			ErrorMessage: fhirErr.Error(),
		})
		return nil, fhirErr
	}

	// Write audit log BEFORE the operation
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		Action:       "create",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusCreated,
	})

	// Extract patient_ref for denormalization
	patientRef := extractPatientRef(req.Data)

	resource := &repository.FHIRResource{
		ResourceType: "Patient",
		Data:         req.Data,
		PatientRef:   patientRef,
		CreatedBy:    &actorID,
		UpdatedBy:    &actorID,
	}

	created, err := s.repo.Create(ctx, resource)
	if err != nil {
		return nil, err
	}

	return &PatientResponse{Data: created.Data}, nil
}

// GetPatient retrieves a Patient by resource ID.
func (s *PatientServiceImpl) GetPatient(ctx context.Context, resourceID string, actorID uuid.UUID) (*PatientResponse, error) {
	// Write audit log BEFORE the operation
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Action:       "read",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	resource, err := s.repo.GetByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	return &PatientResponse{Data: resource.Data}, nil
}

// GetPatientVersion retrieves a specific version of a Patient.
func (s *PatientServiceImpl) GetPatientVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*PatientResponse, error) {
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Action:       "read",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	resource, err := s.repo.GetVersion(ctx, resourceID, versionID)
	if err != nil {
		return nil, err
	}

	return &PatientResponse{Data: resource.Data}, nil
}

// GetPatientHistory retrieves all versions of a Patient as a FHIR Bundle.
func (s *PatientServiceImpl) GetPatientHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*BundleResponse, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Action:       "read",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	resources, total, err := s.repo.GetHistory(ctx, resourceID, count, offset)
	if err != nil {
		return nil, err
	}

	bundle := buildHistoryBundle(resources, total, resourceID, count, offset)
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal history bundle")
		return nil, fhirerrors.NewProcessingError("failed to build history response")
	}

	return &BundleResponse{Data: bundleJSON}, nil
}

// UpdatePatient validates and updates an existing Patient resource.
func (s *PatientServiceImpl) UpdatePatient(ctx context.Context, resourceID string, req UpdatePatientRequest, actorID uuid.UUID) (*PatientResponse, error) {
	// Validate the Patient resource
	if fhirErr := s.validator.ValidatePatient(req.Data, resourceID); fhirErr != nil {
		s.audit.LogAsync(audit.AuditEntry{
			UserID:       &actorID,
			ResourceType: "Patient",
			ResourceID:   resourceID,
			Action:       "update",
			Success:      false,
			StatusCode:   fhirErr.StatusCode,
			ErrorMessage: fhirErr.Error(),
		})
		return nil, fhirErr
	}

	// Write audit log BEFORE the operation
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Action:       "update",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	patientRef := extractPatientRef(req.Data)

	resource := &repository.FHIRResource{
		ResourceType: "Patient",
		Data:         req.Data,
		PatientRef:   patientRef,
		UpdatedBy:    &actorID,
	}

	updated, err := s.repo.Update(ctx, resourceID, resource)
	if err != nil {
		return nil, err
	}

	return &PatientResponse{Data: updated.Data}, nil
}

// DeletePatient soft-deletes a Patient resource.
func (s *PatientServiceImpl) DeletePatient(ctx context.Context, resourceID string, actorID uuid.UUID) error {
	// Write audit log BEFORE the operation
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   resourceID,
		Action:       "delete",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusNoContent,
	})

	return s.repo.Delete(ctx, resourceID, actorID)
}

// SearchPatients searches for Patients and returns a FHIR Bundle.
func (s *PatientServiceImpl) SearchPatients(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*BundleResponse, error) {
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		Action:       "search",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	resources, total, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	bundle := buildSearchBundle(resources, total, params, baseURL)
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal search bundle")
		return nil, fhirerrors.NewProcessingError("failed to build search response")
	}

	return &BundleResponse{Data: bundleJSON}, nil
}

// extractPatientRef extracts the patient reference from FHIR data for denormalization.
func extractPatientRef(data json.RawMessage) *string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	// For a Patient resource, the patient_ref is "Patient/<id>"
	if id, ok := m["id"].(string); ok && id != "" {
		ref := "Patient/" + id
		return &ref
	}
	return nil
}

// buildSearchBundle constructs a FHIR Bundle for search results.
func buildSearchBundle(resources []*repository.FHIRResource, total int, params repository.PatientSearchParams, baseURL string) map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(resources))
	for _, r := range resources {
		var data interface{}
		json.Unmarshal(r.Data, &data)
		entries = append(entries, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("%s/fhir/R4/Patient/%s", baseURL, r.ResourceID),
			"resource": data,
			"search": map[string]interface{}{
				"mode": "match",
			},
		})
	}

	count := params.Count
	if count <= 0 {
		count = 20
	}

	links := []map[string]string{
		{
			"relation": "self",
			"url":      fmt.Sprintf("%s/fhir/R4/Patient?_count=%d&_offset=%d", baseURL, count, params.Offset),
		},
	}

	if params.Offset+count < total {
		links = append(links, map[string]string{
			"relation": "next",
			"url":      fmt.Sprintf("%s/fhir/R4/Patient?_count=%d&_offset=%d", baseURL, count, params.Offset+count),
		})
	}

	if params.Offset > 0 {
		prevOffset := params.Offset - count
		if prevOffset < 0 {
			prevOffset = 0
		}
		links = append(links, map[string]string{
			"relation": "previous",
			"url":      fmt.Sprintf("%s/fhir/R4/Patient?_count=%d&_offset=%d", baseURL, count, prevOffset),
		})
	}

	return map[string]interface{}{
		"resourceType": "Bundle",
		"id":           uuid.New().String(),
		"meta": map[string]interface{}{
			"lastUpdated": time.Now().UTC().Format(time.RFC3339),
		},
		"type":  "searchset",
		"total": total,
		"link":  links,
		"entry": entries,
	}
}

// buildHistoryBundle constructs a FHIR Bundle for history results.
func buildHistoryBundle(resources []*repository.FHIRResource, total int, resourceID string, count, offset int) map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(resources))
	for _, r := range resources {
		var data interface{}
		json.Unmarshal(r.Data, &data)
		entries = append(entries, map[string]interface{}{
			"resource": data,
		})
	}

	return map[string]interface{}{
		"resourceType": "Bundle",
		"id":           uuid.New().String(),
		"meta": map[string]interface{}{
			"lastUpdated": time.Now().UTC().Format(time.RFC3339),
		},
		"type":  "history",
		"total": total,
		"entry": entries,
	}
}
