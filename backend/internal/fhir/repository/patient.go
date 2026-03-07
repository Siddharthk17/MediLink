// Package repository provides FHIR resource data access.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// FHIRResource represents a FHIR resource stored in the database.
type FHIRResource struct {
	ID           uuid.UUID       `db:"id" json:"-"`
	ResourceType string          `db:"resource_type" json:"-"`
	ResourceID   string          `db:"resource_id" json:"-"`
	VersionID    int             `db:"version_id" json:"-"`
	Data         json.RawMessage `db:"data" json:"-"`
	PatientRef   *string         `db:"patient_ref" json:"-"`
	Status       string          `db:"status" json:"-"`
	CreatedAt    time.Time       `db:"created_at" json:"-"`
	UpdatedAt    time.Time       `db:"updated_at" json:"-"`
	DeletedAt    *time.Time      `db:"deleted_at" json:"-"`
	CreatedBy    *uuid.UUID      `db:"created_by" json:"-"`
	UpdatedBy    *uuid.UUID      `db:"updated_by" json:"-"`
}

// PatientSearchParams holds search query parameters for Patient resources.
type PatientSearchParams struct {
	Family     string
	Given      string
	Name       string
	BirthDate  string
	BirthDateOp string // lt, gt, eq (default eq)
	Gender     string
	Identifier string
	ID         string
	Active     *bool
	Count      int
	Offset     int
}

// PatientRepository defines the interface for Patient data access.
type PatientRepository interface {
	// Create inserts a new Patient resource and its first history entry.
	Create(ctx context.Context, resource *FHIRResource) (*FHIRResource, error)

	// GetByID returns the active (non-deleted) patient with the given resource_id.
	GetByID(ctx context.Context, resourceID string) (*FHIRResource, error)

	// GetVersion returns a specific historical version of a patient.
	GetVersion(ctx context.Context, resourceID string, versionID int) (*FHIRResource, error)

	// GetHistory returns all versions of a patient, newest first.
	GetHistory(ctx context.Context, resourceID string, count, offset int) ([]*FHIRResource, int, error)

	// Update creates a new version of the patient resource.
	Update(ctx context.Context, resourceID string, resource *FHIRResource) (*FHIRResource, error)

	// Delete soft-deletes the patient by setting deleted_at = NOW().
	Delete(ctx context.Context, resourceID string, deletedBy uuid.UUID) error

	// Search executes a FHIR search query and returns a paginated result.
	Search(ctx context.Context, params PatientSearchParams) ([]*FHIRResource, int, error)

	// Exists returns true if a patient with the given resource_id exists and is active.
	Exists(ctx context.Context, resourceID string) (bool, error)
}

// PostgresPatientRepository implements PatientRepository using PostgreSQL.
type PostgresPatientRepository struct {
	db *sqlx.DB
}

// NewPostgresPatientRepository creates a new PostgreSQL-backed patient repository.
func NewPostgresPatientRepository(db *sqlx.DB) *PostgresPatientRepository {
	return &PostgresPatientRepository{db: db}
}

// Create inserts a new Patient resource and its first history entry.
func (r *PostgresPatientRepository) Create(ctx context.Context, resource *FHIRResource) (*FHIRResource, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to begin transaction")
	}
	defer tx.Rollback()

	resourceID := uuid.New().String()
	now := time.Now().UTC()

	resource.ResourceID = resourceID
	resource.VersionID = 1
	resource.CreatedAt = now
	resource.UpdatedAt = now
	resource.Status = "active"

	// Update the FHIR data with server-assigned fields
	resource.Data, err = setServerFields(resource.Data, resourceID, 1, now)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to set server fields")
	}

	// Insert into fhir_resources
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resources (resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		resource.ResourceType, resource.ResourceID, resource.VersionID,
		resource.Data, resource.PatientRef, resource.Status,
		resource.CreatedAt, resource.UpdatedAt, resource.CreatedBy, resource.UpdatedBy,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to create patient resource")
	}

	// Insert into fhir_resource_history
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resource_history (resource_type, resource_id, version_id, data, operation, operated_at, operated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		resource.ResourceType, resource.ResourceID, resource.VersionID,
		resource.Data, "create", now, resource.CreatedBy,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to create history entry")
	}

	if err := tx.Commit(); err != nil {
		return nil, fhirerrors.NewProcessingError("failed to commit transaction")
	}

	return resource, nil
}

// GetByID returns the active (non-deleted) patient with the given resource_id.
func (r *PostgresPatientRepository) GetByID(ctx context.Context, resourceID string) (*FHIRResource, error) {
	var resource FHIRResource
	err := r.db.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by
		FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL`,
		resourceID,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError("Patient", resourceID)
	}
	return &resource, nil
}

// GetVersion returns a specific historical version of a patient.
func (r *PostgresPatientRepository) GetVersion(ctx context.Context, resourceID string, versionID int) (*FHIRResource, error) {
	var resource FHIRResource
	err := r.db.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, '' AS patient_ref, '' AS status, operated_at AS created_at, operated_at AS updated_at, NULL AS deleted_at, operated_by AS created_by, operated_by AS updated_by
		FROM fhir_resource_history
		WHERE resource_id = $1 AND version_id = $2 AND resource_type = 'Patient'`,
		resourceID, versionID,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError("Patient", fmt.Sprintf("%s/_history/%d", resourceID, versionID))
	}
	return &resource, nil
}

// GetHistory returns all versions of a patient, newest first.
func (r *PostgresPatientRepository) GetHistory(ctx context.Context, resourceID string, count, offset int) ([]*FHIRResource, int, error) {
	// Check resource exists
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = 'Patient')`,
		resourceID,
	)
	if err != nil || !exists {
		return nil, 0, fhirerrors.NewNotFoundError("Patient", resourceID)
	}

	// Get total count
	var total int
	err = r.db.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM fhir_resource_history WHERE resource_id = $1 AND resource_type = 'Patient'`,
		resourceID,
	)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to count history entries")
	}

	// Get history entries
	var resources []*FHIRResource
	err = r.db.SelectContext(ctx, &resources, `
		SELECT id, resource_type, resource_id, version_id, data, '' AS patient_ref, '' AS status, operated_at AS created_at, operated_at AS updated_at, NULL AS deleted_at, operated_by AS created_by, operated_by AS updated_by
		FROM fhir_resource_history
		WHERE resource_id = $1 AND resource_type = 'Patient'
		ORDER BY version_id DESC
		LIMIT $2 OFFSET $3`,
		resourceID, count, offset,
	)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to get history entries")
	}

	return resources, total, nil
}

// Update creates a new version of the patient resource.
func (r *PostgresPatientRepository) Update(ctx context.Context, resourceID string, resource *FHIRResource) (*FHIRResource, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to begin transaction")
	}
	defer tx.Rollback()

	// Get current version
	var currentVersion int
	err = tx.GetContext(ctx, &currentVersion, `
		SELECT version_id FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL
		FOR UPDATE`,
		resourceID,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError("Patient", resourceID)
	}

	newVersion := currentVersion + 1
	now := time.Now().UTC()

	// Update the FHIR data with server-assigned fields
	resource.Data, err = setServerFields(resource.Data, resourceID, newVersion, now)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to set server fields")
	}

	resource.ResourceID = resourceID
	resource.VersionID = newVersion
	resource.UpdatedAt = now

	// Update fhir_resources
	_, err = tx.ExecContext(ctx, `
		UPDATE fhir_resources
		SET version_id = $1, data = $2, patient_ref = $3, updated_at = $4, updated_by = $5
		WHERE resource_id = $6 AND resource_type = 'Patient' AND deleted_at IS NULL`,
		newVersion, resource.Data, resource.PatientRef, now, resource.UpdatedBy, resourceID,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to update patient resource")
	}

	// Insert into fhir_resource_history
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resource_history (resource_type, resource_id, version_id, data, operation, operated_at, operated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		resource.ResourceType, resourceID, newVersion, resource.Data, "update", now, resource.UpdatedBy,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to create history entry")
	}

	if err := tx.Commit(); err != nil {
		return nil, fhirerrors.NewProcessingError("failed to commit transaction")
	}

	return resource, nil
}

// Delete soft-deletes the patient by setting deleted_at = NOW().
func (r *PostgresPatientRepository) Delete(ctx context.Context, resourceID string, deletedBy uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to begin transaction")
	}
	defer tx.Rollback()

	// Get current resource for history
	var resource FHIRResource
	err = tx.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by
		FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL
		FOR UPDATE`,
		resourceID,
	)
	if err != nil {
		return fhirerrors.NewNotFoundError("Patient", resourceID)
	}

	now := time.Now().UTC()

	// Soft delete
	result, err := tx.ExecContext(ctx, `
		UPDATE fhir_resources SET deleted_at = $1, updated_at = $2, updated_by = $3
		WHERE resource_id = $4 AND resource_type = 'Patient' AND deleted_at IS NULL`,
		now, now, deletedBy, resourceID,
	)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to delete patient resource")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fhirerrors.NewNotFoundError("Patient", resourceID)
	}

	// Write delete history entry
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resource_history (resource_type, resource_id, version_id, data, operation, operated_at, operated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		"Patient", resourceID, resource.VersionID+1, resource.Data, "delete", now, deletedBy,
	)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to create delete history entry")
	}

	if err := tx.Commit(); err != nil {
		return fhirerrors.NewProcessingError("failed to commit transaction")
	}

	return nil
}

// Search executes a FHIR search query and returns a paginated result.
func (r *PostgresPatientRepository) Search(ctx context.Context, params PatientSearchParams) ([]*FHIRResource, int, error) {
	if params.Count <= 0 {
		params.Count = 20
	}
	if params.Count > 100 {
		params.Count = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "resource_type = 'Patient'")
	conditions = append(conditions, "deleted_at IS NULL")

	if params.ID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argIdx))
		args = append(args, params.ID)
		argIdx++
	}

	if params.Family != "" {
		conditions = append(conditions, fmt.Sprintf("data->'name' @> $%d::jsonb", argIdx))
		familyJSON, _ := json.Marshal([]map[string]string{{"family": params.Family}})
		args = append(args, string(familyJSON))
		argIdx++
	}

	if params.Given != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE EXISTS (SELECT 1 FROM jsonb_array_elements_text(n->'given') AS g WHERE LOWER(g) = LOWER($%d)))", argIdx))
		args = append(args, params.Given)
		argIdx++
	}

	if params.Name != "" {
		conditions = append(conditions, fmt.Sprintf(`(
			EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE LOWER(n->>'family') LIKE LOWER($%d))
			OR EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE EXISTS (SELECT 1 FROM jsonb_array_elements_text(n->'given') AS g WHERE LOWER(g) LIKE LOWER($%d)))
			OR EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE LOWER(n->>'text') LIKE LOWER($%d))
		)`, argIdx, argIdx+1, argIdx+2))
		nameLike := "%" + params.Name + "%"
		args = append(args, nameLike, nameLike, nameLike)
		argIdx += 3
	}

	if params.Gender != "" {
		conditions = append(conditions, fmt.Sprintf("data->>'gender' = $%d", argIdx))
		args = append(args, params.Gender)
		argIdx++
	}

	if params.BirthDate != "" {
		op := "="
		switch params.BirthDateOp {
		case "lt":
			op = "<"
		case "gt":
			op = ">"
		case "le":
			op = "<="
		case "ge":
			op = ">="
		}
		conditions = append(conditions, fmt.Sprintf("data->>'birthDate' %s $%d", op, argIdx))
		args = append(args, params.BirthDate)
		argIdx++
	}

	if params.Identifier != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_array_elements(data->'identifier') AS ident WHERE ident->>'value' = $%d)", argIdx))
		args = append(args, params.Identifier)
		argIdx++
	}

	if params.Active != nil {
		conditions = append(conditions, fmt.Sprintf("(data->>'active')::boolean = $%d", argIdx))
		args = append(args, *params.Active)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fhir_resources WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to count search results")
	}

	// Get paginated results
	dataQuery := fmt.Sprintf(`
		SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by
		FROM fhir_resources
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, params.Count, params.Offset)

	var resources []*FHIRResource
	err = r.db.SelectContext(ctx, &resources, dataQuery, args...)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to execute search query")
	}

	return resources, total, nil
}

// Exists returns true if a patient with the given resource_id exists and is active.
func (r *PostgresPatientRepository) Exists(ctx context.Context, resourceID string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL)`,
		resourceID,
	)
	if err != nil {
		return false, fhirerrors.NewProcessingError("failed to check resource existence")
	}
	return exists, nil
}

// setServerFields updates the FHIR JSON data with server-controlled fields.
func setServerFields(data json.RawMessage, resourceID string, versionID int, lastUpdated time.Time) (json.RawMessage, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	m["id"] = resourceID
	m["meta"] = map[string]interface{}{
		"versionId":   fmt.Sprintf("%d", versionID),
		"lastUpdated": lastUpdated.Format(time.RFC3339),
		"profile":     []string{"http://hl7.org/fhir/StructureDefinition/Patient"},
	}

	return json.Marshal(m)
}
