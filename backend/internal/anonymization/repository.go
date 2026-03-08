package anonymization

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ResearchExport represents a single anonymized data export request.
type ResearchExport struct {
	ID              string     `json:"id" db:"id"`
	RequestedBy     string     `json:"requestedBy" db:"requested_by"`
	ResourceTypes   []string   `json:"resourceTypes" db:"resource_types"`
	DateFrom        *time.Time `json:"dateFrom,omitempty" db:"date_from"`
	DateTo          *time.Time `json:"dateTo,omitempty" db:"date_to"`
	KValue          int        `json:"kValue" db:"k_value"`
	Status          string     `json:"status" db:"status"`
	RecordCount     *int       `json:"recordCount,omitempty" db:"record_count"`
	ExportedCount   *int       `json:"exportedCount,omitempty" db:"exported_count"`
	SuppressedCount *int       `json:"suppressedCount,omitempty" db:"suppressed_count"`
	MinioBucket     string     `json:"-" db:"minio_bucket"`
	MinioKey        string     `json:"-" db:"minio_key"`
	FileSize        *int64     `json:"fileSize,omitempty" db:"file_size"`
	Purpose         string     `json:"purpose" db:"purpose"`
	ExportSalt      string     `json:"-" db:"export_salt"`
	ErrorMessage    string     `json:"errorMessage,omitempty" db:"error_message"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	CompletedAt     *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

// ExportRepository defines CRUD operations for research exports.
type ExportRepository interface {
	Create(ctx context.Context, export *ResearchExport) error
	GetByID(ctx context.Context, id string) (*ResearchExport, error)
	List(ctx context.Context, requesterID string, isAdmin bool, count, offset int) ([]*ResearchExport, int, error)
	UpdateStatus(ctx context.Context, id, status string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
}

// PostgresExportRepository implements ExportRepository with PostgreSQL / sqlx.
type PostgresExportRepository struct {
	db *sqlx.DB
}

// NewPostgresExportRepository creates a new repository backed by the given DB.
func NewPostgresExportRepository(db *sqlx.DB) *PostgresExportRepository {
	return &PostgresExportRepository{db: db}
}

func (r *PostgresExportRepository) Create(ctx context.Context, export *ResearchExport) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO research_exports (
			id, requested_by, resource_types, date_from, date_to,
			k_value, status, purpose, export_salt, minio_bucket, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		export.ID,
		export.RequestedBy,
		pq.Array(export.ResourceTypes),
		export.DateFrom,
		export.DateTo,
		export.KValue,
		export.Status,
		export.Purpose,
		export.ExportSalt,
		export.MinioBucket,
		export.CreatedAt,
	)
	return err
}

func (r *PostgresExportRepository) GetByID(ctx context.Context, id string) (*ResearchExport, error) {
	var export ResearchExport
	err := r.db.QueryRowContext(ctx, `
		SELECT id, requested_by, resource_types, date_from, date_to,
			k_value, status, record_count, exported_count, suppressed_count,
			minio_bucket, minio_key, file_size, purpose, export_salt,
			error_message, created_at, completed_at
		FROM research_exports WHERE id = $1`, id,
	).Scan(
		&export.ID,
		&export.RequestedBy,
		pq.Array(&export.ResourceTypes),
		&export.DateFrom,
		&export.DateTo,
		&export.KValue,
		&export.Status,
		&export.RecordCount,
		&export.ExportedCount,
		&export.SuppressedCount,
		&export.MinioBucket,
		&export.MinioKey,
		&export.FileSize,
		&export.Purpose,
		&export.ExportSalt,
		&export.ErrorMessage,
		&export.CreatedAt,
		&export.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &export, nil
}

func (r *PostgresExportRepository) List(ctx context.Context, requesterID string, isAdmin bool, count, offset int) ([]*ResearchExport, int, error) {
	var total int
	var rows *sqlx.Rows
	var err error

	if isAdmin {
		err = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM research_exports`)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.db.QueryxContext(ctx, `
			SELECT id, requested_by, resource_types, date_from, date_to,
				k_value, status, record_count, exported_count, suppressed_count,
				minio_bucket, minio_key, file_size, purpose, export_salt,
				error_message, created_at, completed_at
			FROM research_exports
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`, count, offset)
	} else {
		err = r.db.GetContext(ctx, &total,
			`SELECT COUNT(*) FROM research_exports WHERE requested_by = $1`, requesterID)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.db.QueryxContext(ctx, `
			SELECT id, requested_by, resource_types, date_from, date_to,
				k_value, status, record_count, exported_count, suppressed_count,
				minio_bucket, minio_key, file_size, purpose, export_salt,
				error_message, created_at, completed_at
			FROM research_exports
			WHERE requested_by = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`, requesterID, count, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var exports []*ResearchExport
	for rows.Next() {
		var e ResearchExport
		if err := rows.Scan(
			&e.ID,
			&e.RequestedBy,
			pq.Array(&e.ResourceTypes),
			&e.DateFrom,
			&e.DateTo,
			&e.KValue,
			&e.Status,
			&e.RecordCount,
			&e.ExportedCount,
			&e.SuppressedCount,
			&e.MinioBucket,
			&e.MinioKey,
			&e.FileSize,
			&e.Purpose,
			&e.ExportSalt,
			&e.ErrorMessage,
			&e.CreatedAt,
			&e.CompletedAt,
		); err != nil {
			return nil, 0, err
		}
		exports = append(exports, &e)
	}
	return exports, total, nil
}

func (r *PostgresExportRepository) UpdateStatus(ctx context.Context, id, status string, updates map[string]interface{}) error {
	// Start with the mandatory status update.
	query := `UPDATE research_exports SET status = $1`
	args := []interface{}{status}
	idx := 2

	for col, val := range updates {
		query += fmt.Sprintf(", %s = $%d", col, idx)
		args = append(args, val)
		idx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *PostgresExportRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM research_exports WHERE id = $1`, id)
	return err
}
