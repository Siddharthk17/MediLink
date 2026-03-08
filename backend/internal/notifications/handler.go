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
		if uid, ok := id.(uuid.UUID); ok {
			return uid.String()
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
// PUT /notifications/preferences
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := getActorID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var prefs NotificationPreferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Security-critical notifications are always enabled
	prefs.EmailBreakGlass = true
	prefs.EmailAccountLocked = true

	if err := h.prefRepo.UpsertPreferences(c.Request.Context(), userID, &prefs); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("failed to update preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update preferences"})
		return
	}

	// Re-fetch to return the stored state
	saved, err := h.prefRepo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, prefs)
		return
	}
	c.JSON(http.StatusOK, saved)
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
