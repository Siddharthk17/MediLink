// Package services provides FHIR resource business logic.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/fhir/repository"
	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// ResourceService handles CRUD operations for any FHIR resource type.
type ResourceService interface {
	Create(ctx context.Context, data json.RawMessage, actorID uuid.UUID) (*PatientResponse, error)
	Get(ctx context.Context, resourceID string, actorID uuid.UUID) (*PatientResponse, error)
	GetVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*PatientResponse, error)
	GetHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*BundleResponse, error)
	Update(ctx context.Context, resourceID string, data json.RawMessage, actorID uuid.UUID) (*PatientResponse, error)
	Delete(ctx context.Context, resourceID string, actorID uuid.UUID) error
	Search(ctx context.Context, conditions []string, args []interface{}, count, offset int, actorID uuid.UUID, baseURL string) (*BundleResponse, error)
}

// ResourceServiceOption configures optional behaviour on ResourceServiceImpl.
type ResourceServiceOption func(*ResourceServiceImpl)

// WithPatientRefExtractor sets a custom function to extract the patient reference from FHIR data.
func WithPatientRefExtractor(fn func(json.RawMessage) *string) ResourceServiceOption {
	return func(s *ResourceServiceImpl) { s.patientRefExtractor = fn }
}

// WithStatusExtractor sets a custom function to extract the status field from FHIR data.
func WithStatusExtractor(fn func(json.RawMessage) string) ResourceServiceOption {
	return func(s *ResourceServiceImpl) { s.statusExtractor = fn }
}

// WithPreCreateHook sets a hook invoked before resource creation (e.g. reference validation).
func WithPreCreateHook(fn func(ctx context.Context, data json.RawMessage) error) ResourceServiceOption {
	return func(s *ResourceServiceImpl) { s.preCreateHook = fn }
}

// WithPreUpdateHook sets a hook invoked before resource update (e.g. status transition validation).
func WithPreUpdateHook(fn func(ctx context.Context, resourceID string, data json.RawMessage) error) ResourceServiceOption {
	return func(s *ResourceServiceImpl) { s.preUpdateHook = fn }
}

// WithPostCreateHook sets a hook invoked after resource creation (e.g. push notifications).
// Receives the created resource data and returned resource ID.
func WithPostCreateHook(fn func(ctx context.Context, resourceID string, data json.RawMessage)) ResourceServiceOption {
	return func(s *ResourceServiceImpl) { s.postCreateHook = fn }
}

// ResourceServiceImpl implements ResourceService for a configurable resource type.
type ResourceServiceImpl struct {
	repo                *repository.BaseRepository
	validator           func(json.RawMessage, string) *fhirerrors.FHIRError
	refValidator        *ReferenceValidator
	audit               audit.AuditLogger
	resourceType        string
	patientRefExtractor func(json.RawMessage) *string
	statusExtractor     func(json.RawMessage) string
	preCreateHook       func(ctx context.Context, data json.RawMessage) error
	preUpdateHook       func(ctx context.Context, resourceID string, data json.RawMessage) error
	postCreateHook      func(ctx context.Context, resourceID string, data json.RawMessage)
}

// NewResourceService creates a new ResourceServiceImpl for the given resource type.
func NewResourceService(
	repo *repository.BaseRepository,
	validator func(json.RawMessage, string) *fhirerrors.FHIRError,
	refValidator *ReferenceValidator,
	al audit.AuditLogger,
	resourceType string,
	opts ...ResourceServiceOption,
) *ResourceServiceImpl {
	s := &ResourceServiceImpl{
		repo:         repo,
		validator:    validator,
		refValidator: refValidator,
		audit:        al,
		resourceType: resourceType,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create validates and creates a new FHIR resource.
func (s *ResourceServiceImpl) Create(ctx context.Context, data json.RawMessage, actorID uuid.UUID) (*PatientResponse, error) {
	if fhirErr := s.validator(data, ""); fhirErr != nil {
		s.audit.LogAsync(audit.AuditEntry{
			UserID:       &actorID,
			ResourceType: s.resourceType,
			Action:       "create",
			Success:      false,
			StatusCode:   fhirErr.StatusCode,
			ErrorMessage: fhirErr.Error(),
		})
		return nil, fhirErr
	}

	if s.preCreateHook != nil {
		hookCtx := context.WithValue(ctx, "actor_id", actorID)
		if err := s.preCreateHook(hookCtx, data); err != nil {
			return nil, err
		}
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
		Action:       "create",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusCreated,
	})

	patientRef := s.extractPatientRef(data)
	status := s.extractStatus(data)

	resource := &repository.FHIRResource{
		ResourceType: s.resourceType,
		Data:         data,
		PatientRef:   patientRef,
		Status:       status,
		CreatedBy:    &actorID,
		UpdatedBy:    &actorID,
	}

	created, err := s.repo.Create(ctx, resource)
	if err != nil {
		return nil, err
	}

	if s.postCreateHook != nil {
		// Extract resource ID from created data for the hook
		var idHolder struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(created.Data, &idHolder)
		go s.postCreateHook(ctx, idHolder.ID, data)
	}

	return &PatientResponse{Data: created.Data}, nil
}

// Get retrieves a resource by its resource ID.
func (s *ResourceServiceImpl) Get(ctx context.Context, resourceID string, actorID uuid.UUID) (*PatientResponse, error) {
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
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

// GetVersion retrieves a specific version of a resource.
func (s *ResourceServiceImpl) GetVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*PatientResponse, error) {
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
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

// GetHistory retrieves all versions of a resource as a FHIR Bundle.
func (s *ResourceServiceImpl) GetHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*BundleResponse, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
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

// Update validates and updates an existing FHIR resource.
func (s *ResourceServiceImpl) Update(ctx context.Context, resourceID string, data json.RawMessage, actorID uuid.UUID) (*PatientResponse, error) {
	if fhirErr := s.validator(data, resourceID); fhirErr != nil {
		s.audit.LogAsync(audit.AuditEntry{
			UserID:       &actorID,
			ResourceType: s.resourceType,
			ResourceID:   resourceID,
			Action:       "update",
			Success:      false,
			StatusCode:   fhirErr.StatusCode,
			ErrorMessage: fhirErr.Error(),
		})
		return nil, fhirErr
	}

	if s.preUpdateHook != nil {
		if err := s.preUpdateHook(ctx, resourceID, data); err != nil {
			return nil, err
		}
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
		ResourceID:   resourceID,
		Action:       "update",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	patientRef := s.extractPatientRef(data)
	status := s.extractStatus(data)

	resource := &repository.FHIRResource{
		ResourceType: s.resourceType,
		Data:         data,
		PatientRef:   patientRef,
		Status:       status,
		UpdatedBy:    &actorID,
	}

	updated, err := s.repo.Update(ctx, resourceID, resource)
	if err != nil {
		return nil, err
	}

	return &PatientResponse{Data: updated.Data}, nil
}

// Delete soft-deletes a FHIR resource.
func (s *ResourceServiceImpl) Delete(ctx context.Context, resourceID string, actorID uuid.UUID) error {
	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
		ResourceID:   resourceID,
		Action:       "delete",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusNoContent,
	})

	return s.repo.SoftDelete(ctx, resourceID, actorID)
}

// Search executes a search with pre-built SQL conditions and returns a FHIR Bundle.
func (s *ResourceServiceImpl) Search(ctx context.Context, conditions []string, args []interface{}, count, offset int, actorID uuid.UUID, baseURL string) (*BundleResponse, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: s.resourceType,
		Action:       "search",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	// Add base conditions for resource type and soft-delete filter.
	nextParam := len(args) + 1
	conditions = append(conditions, fmt.Sprintf("resource_type = $%d", nextParam))
	args = append(args, s.resourceType)
	conditions = append(conditions, "deleted_at IS NULL")

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching resources.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fhir_resources WHERE %s", whereClause)
	if err := s.repo.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		log.Error().Err(err).Str("resource_type", s.resourceType).Msg("search count query failed")
		return nil, fhirerrors.NewProcessingError("failed to count search results")
	}

	// Fetch the page of results.
	dataQuery := fmt.Sprintf(
		"SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by FROM fhir_resources WHERE %s ORDER BY created_at DESC LIMIT %d OFFSET %d",
		whereClause, count, offset,
	)

	var resources []*repository.FHIRResource
	if err := s.repo.DB.SelectContext(ctx, &resources, dataQuery, args...); err != nil {
		log.Error().Err(err).Str("resource_type", s.resourceType).Msg("search data query failed")
		return nil, fhirerrors.NewProcessingError("failed to execute search query")
	}

	bundle := GenericSearchBundle(resources, total, s.resourceType, count, offset, baseURL)
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal search bundle")
		return nil, fhirerrors.NewProcessingError("failed to build search response")
	}

	return &BundleResponse{Data: bundleJSON}, nil
}

// Helper: patient-ref / status extraction

func (s *ResourceServiceImpl) extractPatientRef(data json.RawMessage) *string {
	if s.patientRefExtractor != nil {
		return s.patientRefExtractor(data)
	}
	// Default: try subject.reference then patient.reference.
	if ref := ExtractSubjectRef(data); ref != nil {
		return ref
	}
	return ExtractPatientFieldRef(data)
}

func (s *ResourceServiceImpl) extractStatus(data json.RawMessage) string {
	if s.statusExtractor != nil {
		return s.statusExtractor(data)
	}
	return ExtractStatusField(data)
}

// Exported helpers for reference / status extraction and bundle building

// ExtractSubjectRef extracts subject.reference for resources that use "subject".
func ExtractSubjectRef(data json.RawMessage) *string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	if subject, ok := m["subject"].(map[string]interface{}); ok {
		if ref, ok := subject["reference"].(string); ok && ref != "" {
			return &ref
		}
	}
	return nil
}

// ExtractPatientFieldRef extracts patient.reference for resources that use "patient".
func ExtractPatientFieldRef(data json.RawMessage) *string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	if patient, ok := m["patient"].(map[string]interface{}); ok {
		if ref, ok := patient["reference"].(string); ok && ref != "" {
			return &ref
		}
	}
	return nil
}

// ExtractStatusField extracts the status field from FHIR data.
func ExtractStatusField(data json.RawMessage) string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if status, ok := m["status"].(string); ok {
		return status
	}
	return ""
}

// GenericSearchBundle builds a FHIR Bundle for search results with the given resourceType.
func GenericSearchBundle(resources []*repository.FHIRResource, total int, resourceType string, count, offset int, baseURL string) map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(resources))
	for _, r := range resources {
		var data interface{}
		json.Unmarshal(r.Data, &data)
		entries = append(entries, map[string]interface{}{
			"fullUrl":  fmt.Sprintf("%s/fhir/R4/%s/%s", baseURL, resourceType, r.ResourceID),
			"resource": data,
			"search": map[string]interface{}{
				"mode": "match",
			},
		})
	}

	links := []map[string]string{
		{
			"relation": "self",
			"url":      fmt.Sprintf("%s/fhir/R4/%s?_count=%d&_offset=%d", baseURL, resourceType, count, offset),
		},
	}

	if offset+count < total {
		links = append(links, map[string]string{
			"relation": "next",
			"url":      fmt.Sprintf("%s/fhir/R4/%s?_count=%d&_offset=%d", baseURL, resourceType, count, offset+count),
		})
	}

	if offset > 0 {
		prevOffset := offset - count
		if prevOffset < 0 {
			prevOffset = 0
		}
		links = append(links, map[string]string{
			"relation": "previous",
			"url":      fmt.Sprintf("%s/fhir/R4/%s?_count=%d&_offset=%d", baseURL, resourceType, count, prevOffset),
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
