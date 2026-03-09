package documents

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/auth"
	"github.com/Siddharthk17/MediLink/pkg/storage"
)

// DocumentHandler handles HTTP requests for document operations.
type DocumentHandler struct {
	jobs        DocumentJobRepository
	storage     storage.StorageClient
	asynqClient *asynq.Client
	bucket      string
	auditLogger audit.AuditLogger
	logger      zerolog.Logger
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(
	jobs DocumentJobRepository,
	store storage.StorageClient,
	asynqClient *asynq.Client,
	bucket string,
	auditLogger audit.AuditLogger,
	logger zerolog.Logger,
) *DocumentHandler {
	return &DocumentHandler{
		jobs:        jobs,
		storage:     store,
		asynqClient: asynqClient,
		bucket:      bucket,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// allowedContentTypes for document upload.
var allowedContentTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
}

// UploadDocument handles POST /documents/upload.
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	// Step 1: Parse multipart form (max 20MB)
	if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "too-costly", "diagnostics": "File too large, max 20MB"}},
		})
		return
	}

	// Step 2: Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "No file provided"}},
		})
		return
	}
	defer file.Close()

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if !allowedContentTypes[contentType] {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-supported",
				"diagnostics": "Unsupported file type. Allowed: PDF, JPEG, PNG, WEBP"}},
		})
		return
	}

	// Step 3: Validate file size
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "too-costly", "diagnostics": "File exceeds 20MB limit"}},
		})
		return
	}

	// Step 4: Get actor from context
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	// Determine patientFHIRID
	var patientFHIRID string
	var patientUserID uuid.UUID

	if actorRole == "patient" {
		// Patient uploads for themselves — look up their FHIR patient ID
		patientFHIRID = c.Query("patientId")
		if patientFHIRID == "" {
			patientFHIRID = actorID.String()
		}
		patientUserID = actorID
	} else {
		// Physician uploads on patient's behalf
		patientFHIRID = c.Query("patientId")
		if patientFHIRID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{"severity": "error", "code": "required",
					"diagnostics": "patientId query parameter required for physician uploads"}},
			})
			return
		}
		patientUserID = actorID // for DB foreign key, use actor
	}

	// Step 5: Upload to MinIO FIRST
	minioKey, err := h.storage.UploadFile(c.Request.Context(), patientFHIRID, file,
		header.Filename, contentType, header.Size)
	if err != nil {
		h.logger.Error().Err(err).Msg("MinIO upload failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": "File storage failed"}},
		})
		return
	}

	// Step 6: Create document_jobs record
	job := &DocumentJob{
		ID:               uuid.New(),
		PatientID:        patientUserID,
		PatientFHIRID:    patientFHIRID,
		OriginalFilename: header.Filename,
		MinioBucket:      h.bucket,
		MinioKey:         minioKey,
		FileSize:         header.Size,
		ContentType:      contentType,
		Status:           "pending",
		UploadedBy:       actorID,
	}

	if err := h.jobs.Create(c.Request.Context(), job); err != nil {
		h.logger.Error().Err(err).Msg("failed to create document job")
		// Clean up MinIO file
		_ = h.storage.DeleteFile(c.Request.Context(), h.bucket, minioKey)
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": "Failed to create job"}},
		})
		return
	}

	// Step 7: Queue Asynq task
	payload, _ := json.Marshal(ProcessDocumentPayload{JobID: job.ID.String()})
	task := asynq.NewTask(TaskProcessDocument, payload)
	taskInfo, err := h.asynqClient.Enqueue(task,
		asynq.Queue("documents"),
		asynq.MaxRetry(3),
		asynq.Timeout(5 * time.Minute),
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to enqueue document task")
		// Job is created but not queued — update status
		errMsg := "failed to queue processing: " + err.Error()
		_ = h.jobs.UpdateStatus(c.Request.Context(), job.ID, "failed", &errMsg)
	} else {
		_ = h.jobs.UpdateAsynqTaskID(c.Request.Context(), job.ID, taskInfo.ID)
	}

	// Step 8: Audit log
	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "DocumentReference",
		Action:       "upload",
		PatientRef:   "Patient/" + patientFHIRID,
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusCreated,
	})

	// Step 9: Return job ID
	c.JSON(http.StatusCreated, gin.H{
		"jobId":                   job.ID,
		"status":                  "pending",
		"estimatedProcessingTime": "2-3 minutes",
		"message":                 "Your document is being processed. You'll be notified when done.",
	})
}

// GetJobStatus handles GET /documents/jobs/:jobId.
func (h *DocumentHandler) GetJobStatus(c *gin.Context) {
	jobIDStr := c.Param("jobId")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "Invalid job ID"}},
		})
		return
	}

	job, err := h.jobs.GetByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found", "diagnostics": "Job not found"}},
		})
		return
	}

	// Access control: patient can only see own jobs
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	if actorRole == "patient" && job.UploadedBy != actorID && job.PatientID != actorID {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found", "diagnostics": "Job not found"}},
		})
		return
	}

	c.JSON(http.StatusOK, job.ToStatusResponse())
}

// ListJobs handles GET /documents/jobs.
func (h *DocumentHandler) ListJobs(c *gin.Context) {
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	patientFHIRID := c.Query("patientId")
	listByUploader := false

	if actorRole == "patient" {
		patientFHIRID = actorID.String()
	} else if patientFHIRID == "" {
		listByUploader = true
	}

	status := c.DefaultQuery("status", "all")
	count := 20
	offset := 0
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}
	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var jobs []*DocumentJob
	var total int
	var err error

	if listByUploader {
		jobs, total, err = h.jobs.ListByUploader(c.Request.Context(), actorID, status, count, offset)
	} else {
		jobs, total, err = h.jobs.ListByPatient(c.Request.Context(), patientFHIRID, status, count, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	responses := make([]*JobStatusResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, job.ToStatusResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   responses,
		"total":  total,
		"count":  count,
		"offset": offset,
	})
}

// DeleteJob handles DELETE /documents/jobs/:jobId.
func (h *DocumentHandler) DeleteJob(c *gin.Context) {
	jobIDStr := c.Param("jobId")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "Invalid job ID"}},
		})
		return
	}

	job, err := h.jobs.GetByID(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found", "diagnostics": "Job not found"}},
		})
		return
	}

	// Access control: patient can only delete own jobs
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	if actorRole == "patient" && job.PatientID != actorID {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Not authorized"}},
		})
		return
	}

	// Only allow deletion of failed or needs-manual-review jobs
	if job.Status != "failed" && job.Status != "needs-manual-review" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "business-rule",
				"diagnostics": "Only failed or needs-manual-review jobs can be deleted"}},
		})
		return
	}

	// Delete MinIO file
	_ = h.storage.DeleteFile(c.Request.Context(), job.MinioBucket, job.MinioKey)

	// Delete job record
	if err := h.jobs.Delete(c.Request.Context(), jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "DocumentReference",
		ResourceID:   jobIDStr,
		Action:       "delete",
		Success:      true,
		StatusCode:   http.StatusNoContent,
	})

	c.Status(http.StatusNoContent)
}
