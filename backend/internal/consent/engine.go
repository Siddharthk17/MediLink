package consent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Valid FHIR resource types that can appear in consent scope.
var validScopeTypes = map[string]bool{
	"*": true,
	"Patient": true, "Observation": true, "DiagnosticReport": true,
	"MedicationRequest": true, "Condition": true, "Encounter": true,
	"AllergyIntolerance": true, "Immunization": true,
}

// Resource types that don't require patient-scoped consent.
var unrestrictedResources = map[string]bool{
	"Practitioner": true, "Organization": true,
}

// RequiresPatientScope returns true if the resource type needs a patient filter.
func RequiresPatientScope(resourceType string) bool {
	return !unrestrictedResources[resourceType]
}

type GrantConsentRequest struct {
	ProviderUserID string     `json:"providerId" binding:"required"`
	Scope          []string   `json:"scope"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	Purpose        string     `json:"purpose"`
	Notes          string     `json:"notes"`
}

type BreakGlassRequest struct {
	PatientFHIRID   string `json:"patientId" binding:"required"`
	Reason          string `json:"reason" binding:"required,min=20"`
	ClinicalContext string `json:"clinicalContext"`
}

type BreakGlassNotification struct {
	PatientName   string
	PhysicianName string
	OrgName       string
	AccessTime    time.Time
	Reason        string
	SupportEmail  string
}

type ConsentEngine interface {
	CheckConsent(ctx context.Context, providerID, patientFHIRID, resourceType string) (bool, error)
	GetConsentScope(ctx context.Context, providerID, patientFHIRID string) ([]string, error)
	GrantConsent(ctx context.Context, req GrantConsentRequest, patientUserID string) (*Consent, error)
	RevokeConsent(ctx context.Context, consentID string, actorUserID string, actorRole string) error
	GetPatientConsents(ctx context.Context, patientUserID string) ([]*Consent, error)
	GetPhysicianPatients(ctx context.Context, physicianUserID string) ([]*ConsentedPatient, error)
	RecordBreakGlass(ctx context.Context, req BreakGlassRequest, physicianUserID string) (*Consent, error)
	InvalidateCache(ctx context.Context, providerID, patientFHIRID string)
	GetPatientFHIRID(ctx context.Context, userID string) (string, error)
}

type consentEngine struct {
	repo   ConsentRepository
	cache  *ConsentCache
	audit  audit.AuditLogger
	logger zerolog.Logger
}

func NewConsentEngine(repo ConsentRepository, cache *ConsentCache, auditLogger audit.AuditLogger, logger zerolog.Logger) ConsentEngine {
	return &consentEngine{
		repo:   repo,
		cache:  cache,
		audit:  auditLogger,
		logger: logger,
	}
}

func (e *consentEngine) CheckConsent(ctx context.Context, providerID, patientFHIRID, resourceType string) (bool, error) {
	// Unrestricted resources (Practitioner, Organization) don't need consent
	if !RequiresPatientScope(resourceType) {
		return true, nil
	}

	// Patient resource is always readable with any consent
	// (physicians need demographics for consultation)
	isPatientResource := resourceType == "Patient"

	// Check cache first
	if e.cache != nil {
		granted, scope, found := e.cache.Get(ctx, providerID, patientFHIRID)
		if found {
			if !granted {
				return false, nil
			}
			return e.scopeAllows(scope, resourceType, isPatientResource), nil
		}
	}

	// Cache miss — check DB
	providerUUID, err := uuid.Parse(providerID)
	if err != nil {
		return false, fmt.Errorf("invalid provider ID: %w", err)
	}

	granted, scope, err := e.repo.CheckConsent(ctx, providerUUID, patientFHIRID)
	if err != nil {
		return false, err
	}

	// Cache the result
	if e.cache != nil {
		e.cache.Set(ctx, providerID, patientFHIRID, granted, scope)
	}

	if !granted {
		return false, nil
	}

	return e.scopeAllows(scope, resourceType, isPatientResource), nil
}

func (e *consentEngine) scopeAllows(scope []string, resourceType string, isPatientResource bool) bool {
	// Patient is always allowed with any consent
	if isPatientResource {
		return true
	}

	for _, s := range scope {
		if s == "*" || s == resourceType {
			return true
		}
	}
	return false
}

func (e *consentEngine) GetConsentScope(ctx context.Context, providerID, patientFHIRID string) ([]string, error) {
	if e.cache != nil {
		granted, scope, found := e.cache.Get(ctx, providerID, patientFHIRID)
		if found && granted {
			return scope, nil
		}
	}

	providerUUID, err := uuid.Parse(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %w", err)
	}

	_, scope, err := e.repo.CheckConsent(ctx, providerUUID, patientFHIRID)
	return scope, err
}

func (e *consentEngine) GrantConsent(ctx context.Context, req GrantConsentRequest, patientUserID string) (*Consent, error) {
	patientUUID, err := uuid.Parse(patientUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient user ID: %w", err)
	}

	providerUUID, err := uuid.Parse(req.ProviderUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %w", err)
	}

	// Cannot grant consent to yourself
	if patientUUID == providerUUID {
		return nil, fmt.Errorf("cannot grant consent to yourself")
	}

	// Validate scope
	scope := req.Scope
	if len(scope) == 0 {
		scope = []string{"*"}
	}
	for _, s := range scope {
		if !validScopeTypes[s] {
			return nil, fmt.Errorf("invalid scope type: %s", s)
		}
	}

	// Validate expiry
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("expiresAt must be in the future")
	}

	purpose := req.Purpose
	if purpose == "" {
		purpose = "treatment"
	}

	// Check if active consent already exists — update instead of duplicate
	existing, err := e.repo.GetActiveConsent(ctx, patientUUID, providerUUID)
	if err != nil {
		return nil, err
	}

	scopeJSON, _ := json.Marshal(scope)

	if existing != nil {
		existing.Scope = scopeJSON
		existing.ExpiresAt = req.ExpiresAt
		existing.Purpose = purpose
		existing.Notes = req.Notes
		if err := e.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		// Invalidate cache
		fhirID, _ := e.repo.GetPatientFHIRID(ctx, patientUUID)
		if e.cache != nil && fhirID != "" {
			e.cache.Invalidate(ctx, req.ProviderUserID, fhirID)
		}
		return existing, nil
	}

	// Create new consent
	consent := &Consent{
		PatientID:  patientUUID,
		ProviderID: providerUUID,
		Scope:      scopeJSON,
		ExpiresAt:  req.ExpiresAt,
		Purpose:    purpose,
		Notes:      req.Notes,
		CreatedBy:  patientUUID,
	}

	if err := e.repo.Create(ctx, consent); err != nil {
		return nil, err
	}

	// Audit log
	e.audit.LogAsync(audit.AuditEntry{
		UserID:       &patientUUID,
		UserRole:     "patient",
		ResourceType: "Consent",
		ResourceID:   consent.ID.String(),
		Action:       "grant",
		Purpose:      purpose,
		Success:      true,
		StatusCode:   201,
	})

	return consent, nil
}

func (e *consentEngine) RevokeConsent(ctx context.Context, consentID string, actorUserID string, actorRole string) error {
	consentUUID, err := uuid.Parse(consentID)
	if err != nil {
		return fmt.Errorf("invalid consent ID: %w", err)
	}

	actorUUID, err := uuid.Parse(actorUserID)
	if err != nil {
		return fmt.Errorf("invalid actor ID: %w", err)
	}

	consent, err := e.repo.GetByID(ctx, consentUUID)
	if err != nil {
		return err
	}
	if consent == nil {
		return fmt.Errorf("consent not found")
	}

	// Already revoked — idempotent
	if consent.RevokedAt != nil {
		return nil
	}

	// Only the patient or admin can revoke
	if actorRole != "admin" && consent.PatientID != actorUUID {
		return fmt.Errorf("only the patient who granted consent can revoke it")
	}

	if err := e.repo.RevokeConsent(ctx, consentUUID, actorUUID); err != nil {
		return err
	}

	// Immediately invalidate cache
	fhirID, _ := e.repo.GetPatientFHIRID(ctx, consent.PatientID)
	if e.cache != nil && fhirID != "" {
		e.cache.Invalidate(ctx, consent.ProviderID.String(), fhirID)
	}

	// Audit log
	e.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorUUID,
		UserRole:     actorRole,
		ResourceType: "Consent",
		ResourceID:   consentID,
		Action:       "revoke",
		Success:      true,
		StatusCode:   200,
	})

	return nil
}

func (e *consentEngine) GetPatientConsents(ctx context.Context, patientUserID string) ([]*Consent, error) {
	uid, err := uuid.Parse(patientUserID)
	if err != nil {
		return nil, err
	}
	return e.repo.GetPatientConsents(ctx, uid)
}

func (e *consentEngine) GetPhysicianPatients(ctx context.Context, physicianUserID string) ([]*ConsentedPatient, error) {
	uid, err := uuid.Parse(physicianUserID)
	if err != nil {
		return nil, err
	}
	return e.repo.GetPhysicianPatients(ctx, uid)
}

func (e *consentEngine) RecordBreakGlass(ctx context.Context, req BreakGlassRequest, physicianUserID string) (*Consent, error) {
	physicianUUID, err := uuid.Parse(physicianUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid physician ID: %w", err)
	}

	if len(req.Reason) < 20 {
		return nil, fmt.Errorf("break-glass reason must be at least 20 characters")
	}

	// Find patient user ID from FHIR ID
	patientUserID, err := e.repo.GetUserIDByFHIRPatientID(ctx, req.PatientFHIRID)
	if err != nil {
		return nil, fmt.Errorf("patient not found: %w", err)
	}

	// Create temporary 24-hour consent
	expiresAt := time.Now().Add(24 * time.Hour)
	scopeJSON, _ := json.Marshal([]string{"*"})

	consent := &Consent{
		PatientID:  patientUserID,
		ProviderID: physicianUUID,
		Scope:      scopeJSON,
		ExpiresAt:  &expiresAt,
		Purpose:    "break-glass",
		Notes:      req.Reason,
		CreatedBy:  physicianUUID,
	}

	if err := e.repo.Create(ctx, consent); err != nil {
		return nil, err
	}

	// Audit log with break-glass flag
	e.audit.LogAsync(audit.AuditEntry{
		UserID:       &physicianUUID,
		UserRole:     "physician",
		ResourceType: "Consent",
		ResourceID:   consent.ID.String(),
		Action:       "break-glass",
		PatientRef:   "Patient/" + req.PatientFHIRID,
		Purpose:      "break-glass",
		Success:      true,
		StatusCode:   201,
	})

	return consent, nil
}

func (e *consentEngine) InvalidateCache(ctx context.Context, providerID, patientFHIRID string) {
	if e.cache != nil {
		e.cache.Invalidate(ctx, providerID, patientFHIRID)
	}
}

func (e *consentEngine) GetPatientFHIRID(ctx context.Context, userID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	return e.repo.GetPatientFHIRID(ctx, uid)
}
