// Package repository provides FHIR resource data access.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
	"github.com/Siddharthk17/MediLink/pkg/search"
)

// BaseRepository provides shared data access logic for all FHIR resources.
type BaseRepository struct {
	DB           *sqlx.DB
	Search       search.SearchClient
	Logger       zerolog.Logger
	ResourceType string
}

// NewBaseRepository creates a new BaseRepository.
func NewBaseRepository(db *sqlx.DB, sc search.SearchClient, logger zerolog.Logger, resourceType string) BaseRepository {
	return BaseRepository{
		DB:           db,
		Search:       sc,
		Logger:       logger,
		ResourceType: resourceType,
	}
}

// GetByID returns the active (non-deleted) resource with the given resource_id.
func (b *BaseRepository) GetByID(ctx context.Context, resourceID string) (*FHIRResource, error) {
	var resource FHIRResource
	err := b.DB.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by
		FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}
	return &resource, nil
}

// GetVersion returns a specific historical version of a resource.
func (b *BaseRepository) GetVersion(ctx context.Context, resourceID string, versionID int) (*FHIRResource, error) {
	var resource FHIRResource
	err := b.DB.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, '' AS patient_ref, '' AS status, operated_at AS created_at, operated_at AS updated_at, NULL AS deleted_at, operated_by AS created_by, operated_by AS updated_by
		FROM fhir_resource_history
		WHERE resource_id = $1 AND version_id = $2 AND resource_type = $3`,
		resourceID, versionID, b.ResourceType,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError(b.ResourceType, fmt.Sprintf("%s/_history/%d", resourceID, versionID))
	}
	return &resource, nil
}

// GetHistory returns all versions of a resource, newest first.
func (b *BaseRepository) GetHistory(ctx context.Context, resourceID string, count, offset int) ([]*FHIRResource, int, error) {
	var exists bool
	err := b.DB.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2)`,
		resourceID, b.ResourceType,
	)
	if err != nil || !exists {
		return nil, 0, fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}

	var total int
	err = b.DB.GetContext(ctx, &total, `
		SELECT COUNT(*) FROM fhir_resource_history WHERE resource_id = $1 AND resource_type = $2`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to count history entries")
	}

	var resources []*FHIRResource
	err = b.DB.SelectContext(ctx, &resources, `
		SELECT id, resource_type, resource_id, version_id, data, '' AS patient_ref, '' AS status, operated_at AS created_at, operated_at AS updated_at, NULL AS deleted_at, operated_by AS created_by, operated_by AS updated_by
		FROM fhir_resource_history
		WHERE resource_id = $1 AND resource_type = $2
		ORDER BY version_id DESC
		LIMIT $3 OFFSET $4`,
		resourceID, b.ResourceType, count, offset,
	)
	if err != nil {
		return nil, 0, fhirerrors.NewProcessingError("failed to get history entries")
	}

	return resources, total, nil
}

// Create inserts a new FHIR resource and writes history. Dual-write to ES.
func (b *BaseRepository) Create(ctx context.Context, resource *FHIRResource) (*FHIRResource, error) {
	tx, err := b.DB.BeginTxx(ctx, nil)
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
	if resource.Status == "" {
		resource.Status = "active"
	}

	resource.Data, err = SetServerFields(resource.Data, resourceID, 1, now, b.ResourceType)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to set server fields")
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resources (resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		resource.ResourceType, resource.ResourceID, resource.VersionID,
		resource.Data, resource.PatientRef, resource.Status,
		resource.CreatedAt, resource.UpdatedAt, resource.CreatedBy, resource.UpdatedBy,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError(fmt.Sprintf("failed to create %s resource", b.ResourceType))
	}

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

	// Dual-write: index in ES (best-effort)
	b.IndexInES(ctx, resource)

	return resource, nil
}

// Update creates a new version of the resource. Dual-write to ES.
func (b *BaseRepository) Update(ctx context.Context, resourceID string, resource *FHIRResource) (*FHIRResource, error) {
	tx, err := b.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to begin transaction")
	}
	defer tx.Rollback()

	var currentVersion int
	err = tx.GetContext(ctx, &currentVersion, `
		SELECT version_id FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL
		FOR UPDATE`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return nil, fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}

	newVersion := currentVersion + 1
	now := time.Now().UTC()

	resource.Data, err = SetServerFields(resource.Data, resourceID, newVersion, now, b.ResourceType)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to set server fields")
	}

	resource.ResourceID = resourceID
	resource.VersionID = newVersion
	resource.UpdatedAt = now

	_, err = tx.ExecContext(ctx, `
		UPDATE fhir_resources
		SET version_id = $1, data = $2, patient_ref = $3, status = $4, updated_at = $5, updated_by = $6
		WHERE resource_id = $7 AND resource_type = $8 AND deleted_at IS NULL`,
		newVersion, resource.Data, resource.PatientRef, resource.Status, now, resource.UpdatedBy, resourceID, b.ResourceType,
	)
	if err != nil {
		return nil, fhirerrors.NewProcessingError(fmt.Sprintf("failed to update %s resource", b.ResourceType))
	}

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

	// Dual-write: index in ES (best-effort)
	b.IndexInES(ctx, resource)

	return resource, nil
}

// SoftDelete soft-deletes a resource by setting deleted_at = NOW().
func (b *BaseRepository) SoftDelete(ctx context.Context, resourceID string, deletedBy uuid.UUID) error {
	tx, err := b.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to begin transaction")
	}
	defer tx.Rollback()

	var resource FHIRResource
	err = tx.GetContext(ctx, &resource, `
		SELECT id, resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at, deleted_at, created_by, updated_by
		FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL
		FOR UPDATE`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}

	now := time.Now().UTC()

	result, err := tx.ExecContext(ctx, `
		UPDATE fhir_resources SET deleted_at = $1, updated_at = $2, updated_by = $3
		WHERE resource_id = $4 AND resource_type = $5 AND deleted_at IS NULL`,
		now, now, deletedBy, resourceID, b.ResourceType,
	)
	if err != nil {
		return fhirerrors.NewProcessingError(fmt.Sprintf("failed to delete %s resource", b.ResourceType))
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO fhir_resource_history (resource_type, resource_id, version_id, data, operation, operated_at, operated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		b.ResourceType, resourceID, resource.VersionID+1, resource.Data, "delete", now, deletedBy,
	)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to create delete history entry")
	}

	if err := tx.Commit(); err != nil {
		return fhirerrors.NewProcessingError("failed to commit transaction")
	}

	// Dual-write: delete from ES (best-effort)
	if err := b.Search.DeleteResource(ctx, b.ResourceType, resourceID); err != nil {
		b.Logger.Error().Err(err).
			Str("resource_type", b.ResourceType).
			Str("resource_id", resourceID).
			Msg("elasticsearch delete failed")
	}

	return nil
}

// Exists returns true if a resource exists and is not soft-deleted.
func (b *BaseRepository) Exists(ctx context.Context, resourceID string) (bool, error) {
	var exists bool
	err := b.DB.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL)`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return false, fhirerrors.NewProcessingError("failed to check resource existence")
	}
	return exists, nil
}

// ResourceExists checks any resource type existence (used by reference validator).
func ResourceExists(ctx context.Context, db *sqlx.DB, resourceType, resourceID string) (bool, error) {
	var exists bool
	err := db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL)`,
		resourceID, resourceType,
	)
	if err != nil {
		return false, fhirerrors.NewProcessingError("failed to check resource existence")
	}
	return exists, nil
}

// GetCurrentStatus returns the current status from the FHIR data for status transition validation.
func (b *BaseRepository) GetCurrentStatus(ctx context.Context, resourceID string) (string, error) {
	var data json.RawMessage
	err := b.DB.GetContext(ctx, &data, `
		SELECT data FROM fhir_resources
		WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL`,
		resourceID, b.ResourceType,
	)
	if err != nil {
		return "", fhirerrors.NewNotFoundError(b.ResourceType, resourceID)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fhirerrors.NewProcessingError("failed to parse resource data")
	}

	status, _ := m["status"].(string)
	return status, nil
}

// IndexInES indexes the resource in Elasticsearch (best-effort).
func (b *BaseRepository) IndexInES(ctx context.Context, resource *FHIRResource) {
	if err := b.Search.IndexResource(ctx, resource.ResourceType, resource.ResourceID, resource.Data); err != nil {
		b.Logger.Error().Err(err).
			Str("resource_type", resource.ResourceType).
			Str("resource_id", resource.ResourceID).
			Msg("elasticsearch indexing failed — resource saved to postgres, will need reindex")
	}
}

// SetServerFields updates the FHIR JSON data with server-controlled fields.
func SetServerFields(data json.RawMessage, resourceID string, versionID int, lastUpdated time.Time, resourceType string) (json.RawMessage, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	m["id"] = resourceID
	m["meta"] = map[string]interface{}{
		"versionId":   fmt.Sprintf("%d", versionID),
		"lastUpdated": lastUpdated.Format(time.RFC3339),
		"profile":     []string{fmt.Sprintf("http://hl7.org/fhir/StructureDefinition/%s", resourceType)},
	}

	return json.Marshal(m)
}
