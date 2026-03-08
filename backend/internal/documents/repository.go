package documents

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// DocumentJobRepository provides data access for document processing jobs.
type DocumentJobRepository interface {
	Create(ctx context.Context, job *DocumentJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*DocumentJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	UpdateProcessingStarted(ctx context.Context, id uuid.UUID) error
	UpdateCompleted(ctx context.Context, id uuid.UUID, reportID string, obsCreated, loincMapped int, ocrConf float64, llmProvider string) error
	UpdateAsynqTaskID(ctx context.Context, id uuid.UUID, taskID string) error
	ListByPatient(ctx context.Context, patientFHIRID string, status string, count, offset int) ([]*DocumentJob, int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresDocumentJobRepository implements DocumentJobRepository.
type PostgresDocumentJobRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewDocumentJobRepository creates a new PostgresDocumentJobRepository.
func NewDocumentJobRepository(db *sqlx.DB, logger zerolog.Logger) *PostgresDocumentJobRepository {
	return &PostgresDocumentJobRepository{db: db, logger: logger}
}

// Create inserts a new document job.
func (r *PostgresDocumentJobRepository) Create(ctx context.Context, job *DocumentJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO document_jobs (id, patient_id, patient_fhir_id, original_filename, minio_bucket, minio_key, file_size, content_type, status, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		job.ID, job.PatientID, job.PatientFHIRID, job.OriginalFilename,
		job.MinioBucket, job.MinioKey, job.FileSize, job.ContentType,
		job.Status, job.UploadedBy)
	return err
}

// GetByID retrieves a document job by ID.
func (r *PostgresDocumentJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*DocumentJob, error) {
	var job DocumentJob
	err := r.db.GetContext(ctx, &job,
		`SELECT * FROM document_jobs WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document job not found: %s", id)
		}
		return nil, err
	}
	return &job, nil
}

// UpdateStatus updates the job status and optionally sets an error message.
func (r *PostgresDocumentJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	if errorMsg != nil {
		_, err := r.db.ExecContext(ctx,
			`UPDATE document_jobs SET status = $1, error_message = $2 WHERE id = $3`,
			status, *errorMsg, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE document_jobs SET status = $1 WHERE id = $2`, status, id)
	return err
}

// UpdateProcessingStarted marks a job as processing.
func (r *PostgresDocumentJobRepository) UpdateProcessingStarted(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE document_jobs SET status = 'processing', processing_started_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

// UpdateCompleted marks a job as completed with results.
func (r *PostgresDocumentJobRepository) UpdateCompleted(ctx context.Context, id uuid.UUID, reportID string, obsCreated, loincMapped int, ocrConf float64, llmProvider string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE document_jobs SET status = 'completed', fhir_report_id = $1, observations_created = $2,
		 loinc_mapped = $3, ocr_confidence = $4, llm_provider = $5, completed_at = $6 WHERE id = $7`,
		reportID, obsCreated, loincMapped, ocrConf, llmProvider, time.Now(), id)
	return err
}

// UpdateAsynqTaskID sets the Asynq task ID for tracking.
func (r *PostgresDocumentJobRepository) UpdateAsynqTaskID(ctx context.Context, id uuid.UUID, taskID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE document_jobs SET asynq_task_id = $1 WHERE id = $2`, taskID, id)
	return err
}

// ListByPatient returns paginated document jobs for a patient.
func (r *PostgresDocumentJobRepository) ListByPatient(ctx context.Context, patientFHIRID string, status string, count, offset int) ([]*DocumentJob, int, error) {
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100
	}

	var total int
	var jobs []*DocumentJob

	if status != "" && status != "all" {
		err := r.db.GetContext(ctx, &total,
			`SELECT COUNT(*) FROM document_jobs WHERE patient_fhir_id = $1 AND status = $2`, patientFHIRID, status)
		if err != nil {
			return nil, 0, err
		}
		err = r.db.SelectContext(ctx, &jobs,
			`SELECT * FROM document_jobs WHERE patient_fhir_id = $1 AND status = $2 ORDER BY uploaded_at DESC LIMIT $3 OFFSET $4`,
			patientFHIRID, status, count, offset)
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := r.db.GetContext(ctx, &total,
			`SELECT COUNT(*) FROM document_jobs WHERE patient_fhir_id = $1`, patientFHIRID)
		if err != nil {
			return nil, 0, err
		}
		err = r.db.SelectContext(ctx, &jobs,
			`SELECT * FROM document_jobs WHERE patient_fhir_id = $1 ORDER BY uploaded_at DESC LIMIT $2 OFFSET $3`,
			patientFHIRID, count, offset)
		if err != nil {
			return nil, 0, err
		}
	}

	return jobs, total, nil
}

// Delete removes a document job.
func (r *PostgresDocumentJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM document_jobs WHERE id = $1`, id)
	return err
}
