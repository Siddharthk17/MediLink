package anonymization

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

const exportBucket = "medilink-exports"

// ResearchHandler serves the /research/export* endpoints.
type ResearchHandler struct {
	repo        ExportRepository
	storage     storage.StorageClient
	asynqClient *asynq.Client
	auditLogger audit.AuditLogger
	logger      zerolog.Logger
}

// NewResearchHandler creates a new ResearchHandler.
func NewResearchHandler(
	repo ExportRepository,
	store storage.StorageClient,
	asynqClient *asynq.Client,
	auditLogger audit.AuditLogger,
	logger zerolog.Logger,
) *ResearchHandler {
	return &ResearchHandler{
		repo:        repo,
		storage:     store,
		asynqClient: asynqClient,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// Request / response types

type exportRequest struct {
	ResourceTypes []string `json:"resourceTypes" binding:"required,min=1"`
	DateFrom      string   `json:"dateFrom"`
	DateTo        string   `json:"dateTo"`
	KValue        int      `json:"kValue"`
	Purpose       string   `json:"purpose" binding:"required,min=20"`
}

type exportTaskPayload struct {
	ExportID string `json:"exportId"`
}

// Handlers
// RequestExport handles POST /research/export.
// It validates the body, persists the export record, enqueues the async task,
// and returns 202 Accepted immediately.
func (h *ResearchHandler) RequestExport(c *gin.Context) {
	var req exportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid",
				"diagnostics": "Invalid request: resourceTypes (array, required) and purpose (min 20 chars) are required"}},
		})
		return
	}

	if len(req.Purpose) < 20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid",
				"diagnostics": "purpose must be at least 20 characters"}},
		})
		return
	}

	kValue := req.KValue
	if kValue <= 0 {
		kValue = 5
	}
	if kValue > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid",
				"diagnostics": "kValue must be between 1 and 100"}},
		})
		return
	}

	salt, err := GenerateExportSalt()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate export salt")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception",
				"diagnostics": "Internal error"}},
		})
		return
	}

	actorID := auth.GetActorID(c)
	exportID := uuid.New().String()
	now := time.Now().UTC()

	export := &ResearchExport{
		ID:            exportID,
		RequestedBy:   actorID.String(),
		ResourceTypes: req.ResourceTypes,
		KValue:        kValue,
		Status:        "queued",
		Purpose:       req.Purpose,
		ExportSalt:    salt,
		MinioBucket:   exportBucket,
		CreatedAt:     now,
	}

	if req.DateFrom != "" {
		if t, err := time.Parse("2006-01-02", req.DateFrom); err == nil {
			export.DateFrom = &t
		}
	}
	if req.DateTo != "" {
		if t, err := time.Parse("2006-01-02", req.DateTo); err == nil {
			export.DateTo = &t
		}
	}

	if err := h.repo.Create(c.Request.Context(), export); err != nil {
		h.logger.Error().Err(err).Msg("failed to create export record")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception",
				"diagnostics": "Failed to create export"}},
		})
		return
	}

	payload, _ := json.Marshal(exportTaskPayload{ExportID: exportID})
	task := asynq.NewTask(TaskAnonymizedExport, payload)
	if _, err := h.asynqClient.Enqueue(task,
		asynq.Queue("exports"),
		asynq.MaxRetry(2),
		asynq.Timeout(30*time.Minute),
	); err != nil {
		h.logger.Error().Err(err).Msg("failed to enqueue export task")
		_ = h.repo.UpdateStatus(c.Request.Context(), exportID, "failed", map[string]interface{}{
			"error_message": "failed to queue processing: " + err.Error(),
		})
	}

	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "ResearchExport",
		ResourceID:   exportID,
		Action:       "create",
		Purpose:      "research",
		Success:      true,
		StatusCode:   http.StatusAccepted,
	})

	c.JSON(http.StatusAccepted, gin.H{
		"id":      exportID,
		"status":  "queued",
		"message": "Export queued for processing",
	})
}

// GetExportStatus handles GET /research/export/:exportId.
// Researchers can only view their own exports; admins see all.
// When the export is completed, presigned download URLs (1h expiry) are included.
func (h *ResearchHandler) GetExportStatus(c *gin.Context) {
	exportID := c.Param("exportId")
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	export, err := h.repo.GetByID(c.Request.Context(), exportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found",
				"diagnostics": "Export not found"}},
		})
		return
	}

	if actorRole != "admin" && export.RequestedBy != actorID.String() {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found",
				"diagnostics": "Export not found"}},
		})
		return
	}

	resp := gin.H{
		"id":            export.ID,
		"status":        export.Status,
		"resourceTypes": export.ResourceTypes,
		"kValue":        export.KValue,
		"purpose":       export.Purpose,
		"createdAt":     export.CreatedAt,
	}

	if export.RecordCount != nil {
		resp["recordCount"] = *export.RecordCount
	}
	if export.ExportedCount != nil {
		resp["exportedCount"] = *export.ExportedCount
	}
	if export.SuppressedCount != nil {
		resp["suppressedCount"] = *export.SuppressedCount
	}
	if export.FileSize != nil {
		resp["fileSize"] = *export.FileSize
	}
	if export.ErrorMessage != "" {
		resp["errorMessage"] = export.ErrorMessage
	}
	if export.CompletedAt != nil {
		resp["completedAt"] = *export.CompletedAt
	}

	// Generate presigned download URLs for completed exports.
	if export.Status == "completed" && export.MinioKey != "" {
		urls := make(map[string]string)
		for _, rt := range export.ResourceTypes {
			key := "exports/" + export.ID + "/" + rt + ".ndjson"
			url, err := h.storage.GetPresignedURL(c.Request.Context(), export.MinioBucket, key)
			if err == nil {
				urls[rt] = url
			}
		}
		manifestKey := "exports/" + export.ID + "/manifest.json"
		if url, err := h.storage.GetPresignedURL(c.Request.Context(), export.MinioBucket, manifestKey); err == nil {
			urls["manifest"] = url
		}
		resp["downloadUrls"] = urls
	}

	c.JSON(http.StatusOK, resp)
}

// ListExports handles GET /research/exports.
// Paginated with ?_count=20&_offset=0.  Researchers see only own exports.
func (h *ResearchHandler) ListExports(c *gin.Context) {
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	count := 20
	offset := 0
	if v := c.Query("_count"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			count = parsed
			if count > 100 {
				count = 100
			}
		}
	}
	if v := c.Query("_offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	isAdmin := actorRole == "admin"
	exports, total, err := h.repo.List(c.Request.Context(), actorID.String(), isAdmin, count, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list exports")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception",
				"diagnostics": "Failed to list exports"}},
		})
		return
	}

	items := make([]gin.H, 0, len(exports))
	for _, e := range exports {
		item := gin.H{
			"id":            e.ID,
			"status":        e.Status,
			"resourceTypes": e.ResourceTypes,
			"kValue":        e.KValue,
			"purpose":       e.Purpose,
			"createdAt":     e.CreatedAt,
		}
		if e.RecordCount != nil {
			item["recordCount"] = *e.RecordCount
		}
		if e.ExportedCount != nil {
			item["exportedCount"] = *e.ExportedCount
		}
		if e.CompletedAt != nil {
			item["completedAt"] = *e.CompletedAt
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"exports": items,
		"total":   total,
		"count":   count,
		"offset":  offset,
	})
}

// DeleteExport handles DELETE /research/export/:exportId.
// Admin only.  Cannot delete an export that is currently processing.
func (h *ResearchHandler) DeleteExport(c *gin.Context) {
	actorRole := auth.GetActorRole(c)
	if actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden",
				"diagnostics": "Only administrators can delete exports"}},
		})
		return
	}

	exportID := c.Param("exportId")
	export, err := h.repo.GetByID(c.Request.Context(), exportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found",
				"diagnostics": "Export not found"}},
		})
		return
	}

	if export.Status == "processing" {
		c.JSON(http.StatusConflict, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "conflict",
				"diagnostics": "Cannot delete an export that is currently processing"}},
		})
		return
	}

	// Clean up MinIO objects.
	if export.MinioKey != "" {
		for _, rt := range export.ResourceTypes {
			key := "exports/" + export.ID + "/" + rt + ".ndjson"
			_ = h.storage.DeleteFile(c.Request.Context(), export.MinioBucket, key)
		}
		manifestKey := "exports/" + export.ID + "/manifest.json"
		_ = h.storage.DeleteFile(c.Request.Context(), export.MinioBucket, manifestKey)
	}

	if err := h.repo.Delete(c.Request.Context(), exportID); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete export")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception",
				"diagnostics": "Failed to delete export"}},
		})
		return
	}

	actorID := auth.GetActorID(c)
	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "ResearchExport",
		ResourceID:   exportID,
		Action:       "delete",
		Purpose:      "admin",
		Success:      true,
		StatusCode:   http.StatusNoContent,
	})

	c.Status(http.StatusNoContent)
}
