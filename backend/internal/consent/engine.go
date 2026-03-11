package consent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Sentinel errors for consent operations.
var (
	ErrConsentNotFound     = errors.New("consent not found")
	ErrSelfConsent         = errors.New("cannot grant consent to yourself")
	ErrProviderNotFound    = errors.New("provider not found")
	ErrProviderNotPhysician = errors.New("provider must be a physician")
	ErrInvalidScope        = errors.New("invalid scope type")
	ErrExpiredDate         = errors.New("expiresAt must be in the future")
	ErrNotPending          = errors.New("consent is not in pending status")
	ErrUnauthorizedAction  = errors.New("unauthorized to perform this action")
	ErrPatientNotFound     = errors.New("patient not found")
	ErrBreakGlassReason    = errors.New("break-glass reason must be at least 20 characters")
	ErrBreakGlassRateLimit = errors.New("break-glass rate limit exceeded: maximum 3 emergency accesses per 24 hours")
)

// UserLookup retrieves basic contact info for notification purposes.
type UserLookup interface {
	GetUserContactInfo(ctx context.Context, userID uuid.UUID) (email, displayName string, err error)
}

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

type AccessLogEntry struct {
	ID             string    `db:"id" json:"id"`
	AccessedBy     string    `db:"user_id" json:"accessedBy"`
	AccessedByName string    `db:"accessor_name" json:"accessedByName"`
	Action         string    `db:"action" json:"action"`
	ResourceType   string    `db:"resource_type" json:"resourceType"`
	ResourceID     string    `db:"resource_id" json:"resourceId"`
	AccessedAt     time.Time `db:"created_at" json:"accessedAt"`
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
	AcceptConsent(ctx context.Context, consentID string, physicianUserID string) error
	DeclineConsent(ctx context.Context, consentID string, physicianUserID string, reason string) error
	GetPatientConsents(ctx context.Context, patientUserID string) ([]*Consent, error)
	GetPhysicianPatients(ctx context.Context, physicianUserID string) ([]*ConsentedPatient, error)
	GetPendingRequests(ctx context.Context, physicianUserID string) ([]*ConsentedPatient, error)
	RecordBreakGlass(ctx context.Context, req BreakGlassRequest, physicianUserID string) (*Consent, error)
	InvalidateCache(ctx context.Context, providerID, patientFHIRID string)
	GetPatientFHIRID(ctx context.Context, userID string) (string, error)
	GetConsentByID(ctx context.Context, consentID uuid.UUID) (*Consent, error)
	GetAccessLog(ctx context.Context, patientFHIRID string) ([]AccessLogEntry, error)
	GetUserDisplayName(ctx context.Context, userID uuid.UUID) string
}

type consentEngine struct {
	repo   ConsentRepository
	cache  *ConsentCache
	audit  audit.AuditLogger
	logger zerolog.Logger
	db     *sqlx.DB
	email  notifications.EmailService
	users  UserLookup
	rdb    *redis.Client
}

func NewConsentEngine(repo ConsentRepository, cache *ConsentCache, auditLogger audit.AuditLogger, logger zerolog.Logger, emailSvc notifications.EmailService, users UserLookup, rdb *redis.Client, db ...*sqlx.DB) ConsentEngine {
	e := &consentEngine{
		repo:   repo,
		cache:  cache,
		audit:  auditLogger,
		logger: logger,
		email:  emailSvc,
		users:  users,
		rdb:    rdb,
	}
	if len(db) > 0 {
		e.db = db[0]
	}
	return e
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
		return nil, ErrSelfConsent
	}

	// Validate provider is an active physician
	if e.db != nil {
		var role string
		err := e.db.GetContext(ctx, &role, "SELECT role FROM users WHERE id = $1 AND status = 'active' AND deleted_at IS NULL", req.ProviderUserID)
		if err != nil {
			return nil, ErrProviderNotFound
		}
		if role != "physician" {
			return nil, ErrProviderNotPhysician
		}
	}

	// Validate scope
	scope := req.Scope
	if len(scope) == 0 {
		scope = []string{"*"}
	}
	for _, s := range scope {
		if !validScopeTypes[s] {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, s)
		}
	}

	// Validate expiry
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredDate
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

	// Create new consent — patients create 'pending', admins create 'active'
	status := "pending"
	consent := &Consent{
		PatientID:  patientUUID,
		ProviderID: providerUUID,
		Scope:      scopeJSON,
		Status:     status,
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
		Action:       "request",
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
		return ErrConsentNotFound
	}

	// Already revoked — idempotent
	if consent.RevokedAt != nil {
		return nil
	}

	// Only the patient, the provider, or admin can revoke
	if actorRole != "admin" && consent.PatientID != actorUUID && consent.ProviderID != actorUUID {
		return fmt.Errorf("%w: only the patient who granted consent or the provider can revoke it", ErrUnauthorizedAction)
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
		return nil, ErrBreakGlassReason
	}

	// Rate limit: max 3 break-glass accesses per physician per 24 hours
	if e.rdb != nil {
		key := fmt.Sprintf("breakglass:%s:count", physicianUserID)
		count, err := e.rdb.Incr(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("rate limit check failed: %w", err)
		}
		if count == 1 {
			e.rdb.Expire(ctx, key, 24*time.Hour)
		}
		if count > 3 {
			e.rdb.Decr(ctx, key)
			return nil, ErrBreakGlassRateLimit
		}
	}

	// Find patient user ID from FHIR ID
	patientUserID, err := e.repo.GetUserIDByFHIRPatientID(ctx, req.PatientFHIRID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPatientNotFound, err)
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

	// Fire-and-forget patient email notification
	if e.email != nil && e.users != nil {
		go func() {
			bgCtx := context.Background()
			patientEmail, patientName, lookupErr := e.users.GetUserContactInfo(bgCtx, patientUserID)
			if lookupErr != nil {
				e.logger.Warn().Err(lookupErr).Str("patientID", patientUserID.String()).
					Msg("failed to get patient info for break-glass notification")
				return
			}
			_, physicianName, lookupErr := e.users.GetUserContactInfo(bgCtx, physicianUUID)
			if lookupErr != nil {
				e.logger.Warn().Err(lookupErr).Str("physicianID", physicianUserID).
					Msg("failed to get physician info for break-glass notification")
				return
			}
			if sendErr := e.email.SendBreakGlassNotification(bgCtx, notifications.BreakGlassNotification{
				PatientEmail:  patientEmail,
				PatientName:   patientName,
				PhysicianName: physicianName,
				AccessTime:    time.Now(),
				Reason:        req.Reason,
			}); sendErr != nil {
				e.logger.Warn().Err(sendErr).Msg("failed to send break-glass notification email")
			}
		}()
	}

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

// AcceptConsent allows a physician to accept a pending consent request.
func (e *consentEngine) AcceptConsent(ctx context.Context, consentID string, physicianUserID string) error {
	consentUUID, err := uuid.Parse(consentID)
	if err != nil {
		return fmt.Errorf("invalid consent ID: %w", err)
	}
	physicianUUID, err := uuid.Parse(physicianUserID)
	if err != nil {
		return fmt.Errorf("invalid physician ID: %w", err)
	}

	consent, err := e.repo.GetByID(ctx, consentUUID)
	if err != nil {
		return err
	}
	if consent == nil {
		return ErrConsentNotFound
	}
	if consent.ProviderID != physicianUUID {
		return fmt.Errorf("%w: only the requested physician can accept this consent", ErrUnauthorizedAction)
	}
	if consent.Status != "pending" {
		return ErrNotPending
	}

	if err := e.repo.UpdateStatus(ctx, consentUUID, "active"); err != nil {
		return err
	}

	// Invalidate cache to allow immediate access
	fhirID, _ := e.repo.GetPatientFHIRID(ctx, consent.PatientID)
	if e.cache != nil && fhirID != "" {
		e.cache.Invalidate(ctx, consent.ProviderID.String(), fhirID)
	}

	e.audit.LogAsync(audit.AuditEntry{
		UserID:       &physicianUUID,
		UserRole:     "physician",
		ResourceType: "Consent",
		ResourceID:   consentID,
		Action:       "accept",
		Success:      true,
		StatusCode:   200,
	})

	return nil
}

// DeclineConsent allows a physician to decline a pending consent request.
func (e *consentEngine) DeclineConsent(ctx context.Context, consentID string, physicianUserID string, reason string) error {
	consentUUID, err := uuid.Parse(consentID)
	if err != nil {
		return fmt.Errorf("invalid consent ID: %w", err)
	}
	physicianUUID, err := uuid.Parse(physicianUserID)
	if err != nil {
		return fmt.Errorf("invalid physician ID: %w", err)
	}

	consent, err := e.repo.GetByID(ctx, consentUUID)
	if err != nil {
		return err
	}
	if consent == nil {
		return ErrConsentNotFound
	}
	if consent.ProviderID != physicianUUID {
		return fmt.Errorf("%w: only the requested physician can decline this consent", ErrUnauthorizedAction)
	}
	if consent.Status != "pending" {
		return ErrNotPending
	}

	if err := e.repo.UpdateStatus(ctx, consentUUID, "rejected"); err != nil {
		return err
	}

	e.audit.LogAsync(audit.AuditEntry{
		UserID:       &physicianUUID,
		UserRole:     "physician",
		ResourceType: "Consent",
		ResourceID:   consentID,
		Action:       "decline",
		Success:      true,
		StatusCode:   200,
	})

	return nil
}

// GetPendingRequests returns pending consent requests for a physician.
func (e *consentEngine) GetPendingRequests(ctx context.Context, physicianUserID string) ([]*ConsentedPatient, error) {
	uid, err := uuid.Parse(physicianUserID)
	if err != nil {
		return nil, err
	}
	return e.repo.GetPendingRequests(ctx, uid)
}

// GetConsentByID retrieves a consent record by its UUID.
func (e *consentEngine) GetConsentByID(ctx context.Context, consentID uuid.UUID) (*Consent, error) {
	return e.repo.GetByID(ctx, consentID)
}

// GetAccessLog returns audit log entries for a patient's resources.
func (e *consentEngine) GetAccessLog(ctx context.Context, patientFHIRID string) ([]AccessLogEntry, error) {
	if e.db == nil {
		return []AccessLogEntry{}, nil
	}

	var entries []AccessLogEntry
	query := `SELECT a.id, COALESCE(a.user_id::text, '') as user_id,
	                 COALESCE(u.full_name, a.user_role) as accessor_name,
	                 a.action, a.resource_type, COALESCE(a.resource_id, '') as resource_id,
	                 a.created_at
	          FROM audit_logs a
	          LEFT JOIN users u ON a.user_id = u.id
	          WHERE a.patient_ref = $1 AND a.success = true
	          ORDER BY a.created_at DESC
	          LIMIT 100`
	err := e.db.SelectContext(ctx, &entries, query, "Patient/"+patientFHIRID)
	if err != nil {
		return nil, fmt.Errorf("failed to get access log: %w", err)
	}
	if entries == nil {
		entries = []AccessLogEntry{}
	}
	return entries, nil
}

// GetUserDisplayName returns the display name for a user, or empty string on failure.
func (e *consentEngine) GetUserDisplayName(ctx context.Context, userID uuid.UUID) string {
	if e.users != nil {
		_, name, err := e.users.GetUserContactInfo(ctx, userID)
		if err == nil && name != "" {
			return name
		}
	}
	if e.db != nil {
		var name string
		if err := e.db.GetContext(ctx, &name, "SELECT COALESCE(full_name, '') FROM users WHERE id = $1", userID); err == nil {
			return name
		}
	}
	return ""
}
