package notifications

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NotificationHandler handles HTTP requests for notification preferences and FCM tokens.
type NotificationHandler struct {
	prefRepo NotificationPrefsRepository
	pushSvc  PushService
	logger   zerolog.Logger
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(prefRepo NotificationPrefsRepository, pushSvc PushService, logger zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{prefRepo: prefRepo, pushSvc: pushSvc, logger: logger}
}

func getActorID(c *gin.Context) string {
	if id, exists := c.Get("actor_id"); exists {
		switch v := id.(type) {
		case string:
			return v
		case uuid.UUID:
			return v.String()
		}
	}
	return ""
}

// GetPreferences returns the current user's notification preferences.
// GET /notifications/preferences
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := getActorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	prefs, err := h.prefRepo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve preferences"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences updates the current user's notification preferences.
// Accepts a partial JSON body — only the provided fields are changed.
// PUT /notifications/preferences
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := getActorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse partial update as a map so missing fields are not zeroed
	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Fetch existing preferences (creates defaults if none exist)
	existing, err := h.prefRepo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get existing preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update preferences"})
		return
	}

	// Apply partial updates
	if v, ok := patch["emailDocumentComplete"]; ok {
		existing.EmailDocumentComplete = toBool(v)
	}
	if v, ok := patch["emailDocumentFailed"]; ok {
		existing.EmailDocumentFailed = toBool(v)
	}
	if v, ok := patch["emailConsentGranted"]; ok {
		existing.EmailConsentGranted = toBool(v)
	}
	if v, ok := patch["emailConsentRevoked"]; ok {
		existing.EmailConsentRevoked = toBool(v)
	}
	if v, ok := patch["pushEnabled"]; ok {
		existing.PushEnabled = toBool(v)
	}
	if v, ok := patch["pushDocumentComplete"]; ok {
		existing.PushDocumentComplete = toBool(v)
	}
	if v, ok := patch["pushNewPrescription"]; ok {
		existing.PushNewPrescription = toBool(v)
	}
	if v, ok := patch["pushLabResultReady"]; ok {
		existing.PushLabResultReady = toBool(v)
	}
	if v, ok := patch["pushConsentRequest"]; ok {
		existing.PushConsentRequest = toBool(v)
	}
	if v, ok := patch["pushCriticalLab"]; ok {
		existing.PushCriticalLab = toBool(v)
	}
	if v, ok := patch["preferredLanguage"]; ok {
		if s, ok := v.(string); ok {
			existing.PreferredLanguage = s
		}
	}

	// Security-critical notifications are always enabled
	existing.EmailBreakGlass = true
	existing.EmailAccountLocked = true

	if err := h.prefRepo.UpsertPreferences(c.Request.Context(), userID, existing); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to update preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update preferences"})
		return
	}

	// Re-fetch to return the stored state
	saved, err := h.prefRepo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, existing)
		return
	}
	c.JSON(http.StatusOK, saved)
}

// toBool safely converts an interface{} to bool.
func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// RegisterFCMToken registers an FCM push token for the current user.
// POST /notifications/fcm-token
func (h *NotificationHandler) RegisterFCMToken(c *gin.Context) {
	userID := getActorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := h.pushSvc.RegisterToken(c.Request.Context(), userID, body.Token); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to register FCM token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token registered"})
}

// RevokeFCMToken clears the FCM push token for the current user.
// DELETE /notifications/fcm-token
func (h *NotificationHandler) RevokeFCMToken(c *gin.Context) {
	userID := getActorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.pushSvc.RevokeToken(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to revoke FCM token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	c.Status(http.StatusNoContent)
}
