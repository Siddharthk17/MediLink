// Package services provides FHIR resource business logic.
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// ReferenceValidator validates cross-resource references.
type ReferenceValidator struct {
	db *sqlx.DB
}

// NewReferenceValidator creates a new ReferenceValidator.
func NewReferenceValidator(db *sqlx.DB) *ReferenceValidator {
	return &ReferenceValidator{db: db}
}

// ValidateReference checks that a referenced resource exists and is active.
// ref is the full FHIR reference string (e.g. "Patient/550e8400-...")
// expectedType is the expected resource type prefix (e.g. "Patient").
func (rv *ReferenceValidator) ValidateReference(ctx context.Context, ref, expectedType string) error {
	if ref == "" {
		return nil
	}

	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return fhirerrors.NewValidationError(
			fmt.Sprintf("Invalid reference format '%s': expected '<ResourceType>/<id>'", ref),
		)
	}

	refType := parts[0]
	refID := parts[1]

	if refType != expectedType {
		return fhirerrors.NewValidationError(
			fmt.Sprintf("Reference type mismatch: expected '%s' but got '%s'", expectedType, refType),
		)
	}

	if refID == "" {
		return fhirerrors.NewValidationError(
			fmt.Sprintf("Reference ID is empty in '%s'", ref),
		)
	}

	var exists bool
	err := rv.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL)`,
		refID, expectedType,
	)
	if err != nil {
		return fhirerrors.NewProcessingError("failed to validate reference")
	}

	if !exists {
		return fhirerrors.NewValidationErrorWithStatus(
			fmt.Sprintf("Referenced %s/%s does not exist or has been deleted", expectedType, refID),
			422,
		)
	}

	return nil
}

// ValidateOptionalReference validates a reference only if it's non-empty.
func (rv *ReferenceValidator) ValidateOptionalReference(ctx context.Context, ref, expectedType string) error {
	if ref == "" {
		return nil
	}
	return rv.ValidateReference(ctx, ref, expectedType)
}
