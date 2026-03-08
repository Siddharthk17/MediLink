package documents

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// DocumentJob represents a document processing job in the database.
type DocumentJob struct {
	ID                  uuid.UUID      `db:"id" json:"id"`
	PatientID           uuid.UUID      `db:"patient_id" json:"patientId"`
	PatientFHIRID       string         `db:"patient_fhir_id" json:"patientFhirId"`
	OriginalFilename    string         `db:"original_filename" json:"originalFilename"`
	MinioBucket         string         `db:"minio_bucket" json:"minioBucket"`
	MinioKey            string         `db:"minio_key" json:"minioKey"`
	FileSize            int64          `db:"file_size" json:"fileSize"`
	ContentType         string         `db:"content_type" json:"contentType"`
	Status              string         `db:"status" json:"status"`
	AsynqTaskID         sql.NullString `db:"asynq_task_id" json:"asynqTaskId,omitempty"`
	FHIRReportID        sql.NullString `db:"fhir_report_id" json:"fhirReportId,omitempty"`
	ObservationsCreated sql.NullInt32  `db:"observations_created" json:"observationsCreated,omitempty"`
	LOINCMapped         sql.NullInt32  `db:"loinc_mapped" json:"loincMapped,omitempty"`
	OCRConfidence       sql.NullFloat64 `db:"ocr_confidence" json:"ocrConfidence,omitempty"`
	LLMProvider         sql.NullString `db:"llm_provider" json:"llmProvider,omitempty"`
	ErrorMessage        sql.NullString `db:"error_message" json:"errorMessage,omitempty"`
	RetryCount          int            `db:"retry_count" json:"retryCount"`
	UploadedAt          time.Time      `db:"uploaded_at" json:"uploadedAt"`
	ProcessingStartedAt sql.NullTime   `db:"processing_started_at" json:"processingStartedAt,omitempty"`
	CompletedAt         sql.NullTime   `db:"completed_at" json:"completedAt,omitempty"`
	UploadedBy          uuid.UUID      `db:"uploaded_by" json:"uploadedBy"`
}

// JobStatusResponse is the API response for a document job status.
type JobStatusResponse struct {
	JobID               uuid.UUID `json:"jobId"`
	Status              string    `json:"status"`
	FHIRReportID        string    `json:"fhirReportId,omitempty"`
	ObservationsCreated int       `json:"observationsCreated,omitempty"`
	LOINCMapped         int       `json:"loincMapped,omitempty"`
	OCRConfidence       float64   `json:"ocrConfidence,omitempty"`
	LLMProvider         string    `json:"llmProvider,omitempty"`
	ErrorMessage        string    `json:"errorMessage,omitempty"`
	UploadedAt          time.Time `json:"uploadedAt"`
	CompletedAt         *time.Time `json:"completedAt,omitempty"`
	EstimatedProcessingTime string `json:"estimatedProcessingTime,omitempty"`
}

// ToStatusResponse converts a DocumentJob to a JobStatusResponse.
func (j *DocumentJob) ToStatusResponse() *JobStatusResponse {
	resp := &JobStatusResponse{
		JobID:      j.ID,
		Status:     j.Status,
		UploadedAt: j.UploadedAt,
	}
	if j.FHIRReportID.Valid {
		resp.FHIRReportID = j.FHIRReportID.String
	}
	if j.ObservationsCreated.Valid {
		resp.ObservationsCreated = int(j.ObservationsCreated.Int32)
	}
	if j.LOINCMapped.Valid {
		resp.LOINCMapped = int(j.LOINCMapped.Int32)
	}
	if j.OCRConfidence.Valid {
		resp.OCRConfidence = j.OCRConfidence.Float64
	}
	if j.LLMProvider.Valid {
		resp.LLMProvider = j.LLMProvider.String
	}
	if j.ErrorMessage.Valid {
		resp.ErrorMessage = j.ErrorMessage.String
	}
	if j.CompletedAt.Valid {
		resp.CompletedAt = &j.CompletedAt.Time
	}
	if j.Status == "pending" {
		resp.EstimatedProcessingTime = "2-3 minutes"
	}
	return resp
}
