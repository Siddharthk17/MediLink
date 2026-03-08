package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/pkg/crypto"
)

// Asynq task type constants for admin-triggered tasks.
const (
	TaskSearchReindex  = "search:reindex"
	TaskCleanupTokens  = "tokens:cleanup"
)

// AdminHandler handles admin API endpoints.
type AdminHandler struct {
	service     Service
	crypto      crypto.Encryptor
	asynqClient *asynq.Client
	logger      zerolog.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(service Service, cryptoSvc crypto.Encryptor, asynqClient *asynq.Client, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{
		service:     service,
		crypto:      cryptoSvc,
		asynqClient: asynqClient,
		logger:      logger.With().Str("component", "admin-handler").Logger(),
	}
}

// writeFHIRError writes a FHIR OperationOutcome error response.
func writeFHIRError(c *gin.Context, status int, code, message string) {
	severity := "error"
	if status >= 500 {
		severity = "fatal"
	}
	c.JSON(status, gin.H{
		"resourceType": "OperationOutcome",
		"issue": []gin.H{{
			"severity": severity,
			"code":     code,
			"details":  gin.H{"text": message},
		}},
	})
}

// parsePagination extracts _count and _offset from query parameters with bounds.
func parsePagination(c *gin.Context, defaultCount, maxCount int) (int, int) {
	count := defaultCount
	offset := 0

	if v := c.Query("_count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			count = n
		}
	}
	if count > maxCount {
		count = maxCount
	}

	if v := c.Query("_offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return count, offset
}

// decryptUserNames decrypts full_name_enc for a slice of UserSummary.
func (h *AdminHandler) decryptUserNames(users []UserSummary) {
	for i := range users {
		if len(users[i].FullNameEnc) > 0 {
			name, err := h.crypto.DecryptString(users[i].FullNameEnc)
			if err != nil {
				h.logger.Warn().Err(err).Str("userId", users[i].ID).Msg("failed to decrypt user name")
				users[i].FullName = "[encrypted]"
			} else {
				users[i].FullName = name
			}
		}
	}
}

// ListUsers handles GET /admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	count, offset := parsePagination(c, 20, 100)

	filters := UserFilters{
		Role:   c.Query("role"),
		Status: c.Query("status"),
		Query:  c.Query("q"),
		Sort:   c.DefaultQuery("_sort", "-created_at"),
	}

	if v := c.Query("created_after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.CreatedAfter = &t
		}
	}
	if v := c.Query("created_before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.CreatedBefore = &t
		}
	}

	users, total, err := h.service.ListUsers(c.Request.Context(), filters, count, offset)
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to list users")
		return
	}

	h.decryptUserNames(users)

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"count":  count,
		"offset": offset,
	})
}

// GetUser handles GET /admin/users/:userId
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "userId is required")
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		writeFHIRError(c, http.StatusNotFound, "not-found", "user not found")
		return
	}

	// Decrypt full name
	if len(user.FullNameEnc) > 0 {
		name, decErr := h.crypto.DecryptString(user.FullNameEnc)
		if decErr != nil {
			h.logger.Warn().Err(decErr).Str("userId", userID).Msg("failed to decrypt user name")
			user.FullName = "[encrypted]"
		} else {
			user.FullName = name
		}
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUserRole handles PUT /admin/users/:userId/role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "userId is required")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Role == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "request body must include a valid 'role' field")
		return
	}

	if err := h.service.UpdateUserRole(c.Request.Context(), userID, body.Role); err != nil {
		if _, ok := err.(*ValidationError); ok {
			writeFHIRError(c, http.StatusBadRequest, "invalid", err.Error())
			return
		}
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to update user role")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// InviteResearcher handles POST /admin/researchers/invite
func (h *AdminHandler) InviteResearcher(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "request body must include a valid 'email' field")
		return
	}

	actorID := c.GetString("actor_id")
	if err := h.service.InviteResearcher(c.Request.Context(), body.Email, actorID); err != nil {
		if _, ok := err.(*ValidationError); ok {
			writeFHIRError(c, http.StatusBadRequest, "invalid", err.Error())
			return
		}
		if _, ok := err.(*DuplicateError); ok {
			writeFHIRError(c, http.StatusConflict, "duplicate", err.Error())
			return
		}
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to invite researcher")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "researcher invitation created"})
}

// GetAuditLogs handles GET /admin/audit-logs
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	count, offset := parsePagination(c, 50, 200)

	filters := AuditLogFilters{
		ActorID:      c.Query("actor_id"),
		ResourceType: c.Query("resource_type"),
		Action:       c.Query("action"),
		PatientRef:   c.Query("patient_ref"),
		Sort:         c.DefaultQuery("_sort", "-created_at"),
	}

	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.To = &t
		}
	}
	if c.Query("break_glass_only") == "true" {
		filters.BreakGlassOnly = true
	}

	entries, total, err := h.service.GetAuditLogs(c.Request.Context(), filters, count, offset)
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to retrieve audit logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"count":   count,
		"offset":  offset,
	})
}

// GetPatientAuditLog handles GET /admin/audit-logs/patient/:id
func (h *AdminHandler) GetPatientAuditLog(c *gin.Context) {
	patientID := c.Param("id")
	if patientID == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "patient id is required")
		return
	}

	count, offset := parsePagination(c, 50, 200)

	patientRef := "Patient/" + patientID
	entries, total, err := h.service.GetPatientAuditLog(c.Request.Context(), patientRef, count, offset)
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to retrieve patient audit log")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"count":   count,
		"offset":  offset,
	})
}

// GetActorAuditLog handles GET /admin/audit-logs/actor/:id
func (h *AdminHandler) GetActorAuditLog(c *gin.Context) {
	actorID := c.Param("id")
	if actorID == "" {
		writeFHIRError(c, http.StatusBadRequest, "invalid", "actor id is required")
		return
	}

	count, offset := parsePagination(c, 50, 200)

	entries, total, err := h.service.GetActorAuditLog(c.Request.Context(), actorID, count, offset)
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to retrieve actor audit log")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"count":   count,
		"offset":  offset,
	})
}

// GetBreakGlassEvents handles GET /admin/audit-logs/break-glass
func (h *AdminHandler) GetBreakGlassEvents(c *gin.Context) {
	count, offset := parsePagination(c, 50, 200)

	entries, total, err := h.service.GetBreakGlassEvents(c.Request.Context(), count, offset)
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to retrieve break-glass events")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"count":   count,
		"offset":  offset,
	})
}

// GetStats handles GET /admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to retrieve platform stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSystemHealth handles GET /admin/system/health
func (h *AdminHandler) GetSystemHealth(c *gin.Context) {
	components := map[string]string{
		"api":      "ok",
		"database": "ok",
		"queue":    "ok",
	}

	// Check database via a lightweight query
	if err := h.service.Ping(c.Request.Context()); err != nil {
		components["database"] = "unavailable"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"components": components,
	})
}

// TriggerReindex handles POST /admin/search/reindex
func (h *AdminHandler) TriggerReindex(c *gin.Context) {
	task := asynq.NewTask(TaskSearchReindex, nil)
	info, err := h.asynqClient.Enqueue(task, asynq.Queue("admin"), asynq.MaxRetry(1), asynq.Timeout(10*time.Minute))
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to enqueue reindex task")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "reindex task queued",
		"taskId":  info.ID,
	})
}

// TriggerTokenCleanup handles POST /admin/tasks/cleanup-tokens
func (h *AdminHandler) TriggerTokenCleanup(c *gin.Context) {
	payload, _ := json.Marshal(map[string]string{"triggered_by": c.GetString("actor_id")})
	task := asynq.NewTask(TaskCleanupTokens, payload)
	info, err := h.asynqClient.Enqueue(task, asynq.Queue("admin"), asynq.MaxRetry(1), asynq.Timeout(5*time.Minute))
	if err != nil {
		writeFHIRError(c, http.StatusInternalServerError, "processing", "failed to enqueue token cleanup task")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "token cleanup task queued",
		"taskId":  info.ID,
	})
}
