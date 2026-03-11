package consent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Consent struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	PatientID     uuid.UUID       `db:"patient_id" json:"patientId"`
	ProviderID    uuid.UUID       `db:"provider_id" json:"providerId"`
	Scope         json.RawMessage `db:"scope" json:"scope"`
	Status        string          `db:"status" json:"status"`
	GrantedAt     time.Time       `db:"granted_at" json:"grantedAt"`
	ExpiresAt     *time.Time      `db:"expires_at" json:"expiresAt,omitempty"`
	RevokedAt     *time.Time      `db:"revoked_at" json:"revokedAt,omitempty"`
	RevokedBy     *uuid.UUID      `db:"revoked_by" json:"revokedBy,omitempty"`
	RevokedReason *string         `db:"revoked_reason" json:"revokedReason,omitempty"`
	Purpose       string          `db:"purpose" json:"purpose"`
	Notes         string          `db:"notes" json:"notes,omitempty"`
	CreatedBy     uuid.UUID       `db:"created_by" json:"createdBy"`
}

type ConsentedPatient struct {
	PatientID     uuid.UUID       `db:"patient_id" json:"patientId"`
	FHIRPatientID string          `db:"fhir_patient_id" json:"fhirPatientId"`
	Scope         json.RawMessage `db:"scope" json:"scope"`
	GrantedAt     time.Time       `db:"granted_at" json:"grantedAt"`
	ExpiresAt     *time.Time      `db:"expires_at" json:"expiresAt,omitempty"`
	Status        string          `db:"status" json:"status"`
	ConsentID     uuid.UUID       `db:"id" json:"consentId"`
	PatientData   json.RawMessage `db:"patient_data" json:"-"`
}

type ConsentRepository interface {
	Create(ctx context.Context, consent *Consent) error
	GetByID(ctx context.Context, id uuid.UUID) (*Consent, error)
	Update(ctx context.Context, consent *Consent) error
	UpdateStatus(ctx context.Context, consentID uuid.UUID, status string) error
	GetActiveConsent(ctx context.Context, patientID, providerID uuid.UUID) (*Consent, error)
	RevokeConsent(ctx context.Context, consentID uuid.UUID, revokedBy uuid.UUID) error
	GetPatientConsents(ctx context.Context, patientID uuid.UUID) ([]*Consent, error)
	GetPhysicianPatients(ctx context.Context, providerID uuid.UUID) ([]*ConsentedPatient, error)
	GetPendingRequests(ctx context.Context, providerID uuid.UUID) ([]*ConsentedPatient, error)
	CheckConsent(ctx context.Context, providerID uuid.UUID, patientFHIRID string) (bool, []string, error)
	GetPatientFHIRID(ctx context.Context, userID uuid.UUID) (string, error)
	GetUserIDByFHIRPatientID(ctx context.Context, fhirPatientID string) (uuid.UUID, error)
}

type postgresConsentRepository struct {
	db *sqlx.DB
}

func NewConsentRepository(db *sqlx.DB) ConsentRepository {
	return &postgresConsentRepository{db: db}
}

func (r *postgresConsentRepository) Create(ctx context.Context, consent *Consent) error {
	status := consent.Status
	if status == "" {
		status = "active"
	}
	query := `INSERT INTO consents (patient_id, provider_id, scope, status, expires_at, purpose, notes, created_by)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, granted_at`
	return r.db.QueryRowContext(ctx, query,
		consent.PatientID, consent.ProviderID, consent.Scope, status,
		consent.ExpiresAt, consent.Purpose, consent.Notes, consent.CreatedBy,
	).Scan(&consent.ID, &consent.GrantedAt)
}

func (r *postgresConsentRepository) GetByID(ctx context.Context, id uuid.UUID) (*Consent, error) {
	var c Consent
	err := r.db.GetContext(ctx, &c, "SELECT * FROM consents WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get consent: %w", err)
	}
	return &c, nil
}

func (r *postgresConsentRepository) Update(ctx context.Context, consent *Consent) error {
	query := `UPDATE consents SET scope = $1, expires_at = $2, purpose = $3, notes = $4
              WHERE id = $5 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, consent.Scope, consent.ExpiresAt, consent.Purpose, consent.Notes, consent.ID)
	return err
}

func (r *postgresConsentRepository) GetActiveConsent(ctx context.Context, patientID, providerID uuid.UUID) (*Consent, error) {
	var c Consent
	query := `SELECT * FROM consents WHERE patient_id = $1 AND provider_id = $2 
              AND revoked_at IS NULL AND status IN ('active', 'pending')
              AND (expires_at IS NULL OR expires_at > NOW())`
	err := r.db.GetContext(ctx, &c, query, patientID, providerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active consent: %w", err)
	}
	return &c, nil
}

func (r *postgresConsentRepository) RevokeConsent(ctx context.Context, consentID uuid.UUID, revokedBy uuid.UUID) error {
	query := `UPDATE consents SET revoked_at = NOW(), revoked_by = $1, status = 'revoked' WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, revokedBy, consentID)
	return err
}

func (r *postgresConsentRepository) UpdateStatus(ctx context.Context, consentID uuid.UUID, status string) error {
	query := `UPDATE consents SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, consentID)
	return err
}

func (r *postgresConsentRepository) GetPatientConsents(ctx context.Context, patientID uuid.UUID) ([]*Consent, error) {
	var consents []*Consent
	query := `SELECT * FROM consents WHERE patient_id = $1 ORDER BY granted_at DESC`
	err := r.db.SelectContext(ctx, &consents, query, patientID)
	return consents, err
}

func (r *postgresConsentRepository) GetPhysicianPatients(ctx context.Context, providerID uuid.UUID) ([]*ConsentedPatient, error) {
	var patients []*ConsentedPatient
	query := `SELECT c.id, c.patient_id, c.scope, c.granted_at, c.expires_at, c.status,
                     u.fhir_patient_id,
                     COALESCE(f.data, '{}'::jsonb) as patient_data
              FROM consents c JOIN users u ON c.patient_id = u.id
              LEFT JOIN fhir_resources f ON f.resource_id = u.fhir_patient_id
                AND f.resource_type = 'Patient' AND f.deleted_at IS NULL
              WHERE c.provider_id = $1 AND c.status = 'active' AND c.revoked_at IS NULL
              AND (c.expires_at IS NULL OR c.expires_at > NOW())
              ORDER BY c.granted_at DESC`
	err := r.db.SelectContext(ctx, &patients, query, providerID)
	return patients, err
}

func (r *postgresConsentRepository) GetPendingRequests(ctx context.Context, providerID uuid.UUID) ([]*ConsentedPatient, error) {
	var patients []*ConsentedPatient
	query := `SELECT c.id, c.patient_id, c.scope, c.granted_at, c.expires_at, c.status,
                     u.fhir_patient_id,
                     COALESCE(f.data, '{}'::jsonb) as patient_data
              FROM consents c JOIN users u ON c.patient_id = u.id
              LEFT JOIN fhir_resources f ON f.resource_id = u.fhir_patient_id
                AND f.resource_type = 'Patient' AND f.deleted_at IS NULL
              WHERE c.provider_id = $1 AND c.status = 'pending'
              ORDER BY c.granted_at DESC`
	err := r.db.SelectContext(ctx, &patients, query, providerID)
	return patients, err
}

// CheckConsent checks if a provider has active consent for a patient (by FHIR patient ID).
func (r *postgresConsentRepository) CheckConsent(ctx context.Context, providerID uuid.UUID, patientFHIRID string) (bool, []string, error) {
	var consent Consent
	query := `SELECT c.* FROM consents c
              JOIN users u ON c.patient_id = u.id
              WHERE c.provider_id = $1 AND u.fhir_patient_id = $2
              AND c.status = 'active' AND c.revoked_at IS NULL AND (c.expires_at IS NULL OR c.expires_at > NOW())`
	err := r.db.GetContext(ctx, &consent, query, providerID, patientFHIRID)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check consent: %w", err)
	}

	var scope []string
	if err := json.Unmarshal(consent.Scope, &scope); err != nil {
		return false, nil, fmt.Errorf("failed to parse consent scope: %w", err)
	}

	return true, scope, nil
}

func (r *postgresConsentRepository) GetPatientFHIRID(ctx context.Context, userID uuid.UUID) (string, error) {
	var fhirID string
	err := r.db.GetContext(ctx, &fhirID, "SELECT COALESCE(fhir_patient_id, '') FROM users WHERE id = $1", userID)
	return fhirID, err
}

func (r *postgresConsentRepository) GetUserIDByFHIRPatientID(ctx context.Context, fhirPatientID string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.GetContext(ctx, &userID, "SELECT id FROM users WHERE fhir_patient_id = $1", fhirPatientID)
	return userID, err
}
