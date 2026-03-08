package notifications_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/notifications"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ginContextWithActor creates a test gin context with actor_id set.
func ginContextWithActor(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("actor_id", uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	return c, w
}

func TestGetPreferences_ReturnsDefaults(t *testing.T) {
	defaults := &notifications.NotificationPreferences{
		UserID:                "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		PushEnabled:           true,
		PushDocumentComplete:  true,
		PushNewPrescription:   true,
		PushLabResultReady:    true,
		PushConsentRequest:    true,
		PushCriticalLab:       true,
		EmailDocumentComplete: true,
		EmailDocumentFailed:   true,
		EmailConsentGranted:   true,
		EmailConsentRevoked:   true,
		EmailBreakGlass:       true,
		EmailAccountLocked:    true,
		PreferredLanguage:     "en",
	}

	repo := &mockPrefsRepo{prefs: defaults}
	logger := zerolog.Nop()
	handler := notifications.NewNotificationHandler(repo, &mockPushService{}, logger)

	c, w := ginContextWithActor("GET", "/notifications/preferences", nil)
	handler.GetPreferences(c)

	require.Equal(t, http.StatusOK, w.Code, "no-row scenario should return 200, not 404")

	var got notifications.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.PushEnabled, "default push_enabled should be true")
	assert.True(t, got.EmailBreakGlass, "default email_break_glass should be true")
	assert.Equal(t, "en", got.PreferredLanguage)
}

func TestUpdatePreferences_Success(t *testing.T) {
	saved := &notifications.NotificationPreferences{
		UserID:                "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		PushEnabled:           false,
		PushDocumentComplete:  false,
		EmailDocumentComplete: true,
		EmailBreakGlass:       true,
		EmailAccountLocked:    true,
		PreferredLanguage:     "es",
	}

	repo := &mockPrefsRepo{prefs: saved}
	logger := zerolog.Nop()
	handler := notifications.NewNotificationHandler(repo, &mockPushService{}, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"pushEnabled":          false,
		"pushDocumentComplete": false,
		"preferredLanguage":    "es",
	})

	c, w := ginContextWithActor("PUT", "/notifications/preferences", body)
	handler.UpdatePreferences(c)

	require.Equal(t, http.StatusOK, w.Code)

	var got notifications.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.False(t, got.PushEnabled, "pushEnabled should be false as requested")
}

func TestGetPreferences_RowExists_Returns200(t *testing.T) {
	prefs := &notifications.NotificationPreferences{
		UserID:                "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		PushEnabled:           true,
		PushDocumentComplete:  true,
		PushNewPrescription:   false,
		PushLabResultReady:    true,
		PushConsentRequest:    true,
		PushCriticalLab:       true,
		EmailDocumentComplete: true,
		EmailDocumentFailed:   true,
		EmailConsentGranted:   false,
		EmailConsentRevoked:   true,
		EmailBreakGlass:       true,
		EmailAccountLocked:    true,
		PreferredLanguage:     "fr",
		FCMToken:              "fcm-token-123",
	}

	repo := &mockPrefsRepo{prefs: prefs}
	logger := zerolog.Nop()
	handler := notifications.NewNotificationHandler(repo, &mockPushService{}, logger)

	c, w := ginContextWithActor("GET", "/notifications/preferences", nil)
	handler.GetPreferences(c)

	require.Equal(t, http.StatusOK, w.Code)

	var got notifications.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.PushEnabled)
	assert.False(t, got.PushNewPrescription)
	assert.Equal(t, "fr", got.PreferredLanguage)
}

func TestUpdatePreferences_BreakGlassAlwaysTrue(t *testing.T) {
	// Repo returns whatever was upserted, so we can inspect the upserted value.
	repo := &mockPrefsRepo{}
	logger := zerolog.Nop()
	handler := notifications.NewNotificationHandler(repo, &mockPushService{}, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"emailBreakGlass":    false,
		"emailAccountLocked": false,
		"pushEnabled":        true,
	})

	c, w := ginContextWithActor("PUT", "/notifications/preferences", body)

	// After UpsertPreferences is called, repo.upsertCall will hold the prefs
	// that handler passed in. The handler enforces break_glass = true.
	// We also provide a prefs for the re-fetch after upsert.
	repo.prefs = &notifications.NotificationPreferences{
		UserID:             "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		EmailBreakGlass:    true,
		EmailAccountLocked: true,
		PushEnabled:        true,
	}

	handler.UpdatePreferences(c)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify the handler enforced break_glass fields before calling repo
	require.NotNil(t, repo.upsertCall, "UpsertPreferences should have been called")
	assert.True(t, repo.upsertCall.EmailBreakGlass,
		"emailBreakGlass must be forced to true regardless of input")
	assert.True(t, repo.upsertCall.EmailAccountLocked,
		"emailAccountLocked must be forced to true regardless of input")

	// Verify response also reflects the enforcement
	var got notifications.NotificationPreferences
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.EmailBreakGlass)
	assert.True(t, got.EmailAccountLocked)
}
