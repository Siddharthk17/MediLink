package notifications_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/notifications"
)

// --- Mock EmailService ---

type mockEmailService struct {
	sendTemplateCalled bool
	sentTo             string
	sentTemplate       string
	sentLang           string
	sentData           map[string]interface{}
	sendErr            error
}

func (m *mockEmailService) SendOTP(_ context.Context, _, _, _, _ string, _ string) error { return nil }
func (m *mockEmailService) SendBreakGlassNotification(_ context.Context, _ notifications.BreakGlassNotification) error {
	return nil
}
func (m *mockEmailService) SendConsentGranted(_ context.Context, _ notifications.ConsentNotification) error {
	return nil
}
func (m *mockEmailService) SendConsentRevoked(_ context.Context, _ notifications.ConsentNotification) error {
	return nil
}
func (m *mockEmailService) SendWelcomePhysician(_ context.Context, _, _ string) error { return nil }
func (m *mockEmailService) SendPhysicianApproved(_ context.Context, _, _ string) error { return nil }
func (m *mockEmailService) SendAccountLocked(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}
func (m *mockEmailService) SendDocumentProcessingComplete(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockEmailService) SendDocumentProcessingFailed(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (m *mockEmailService) SendDocumentNeedsReview(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockEmailService) SendTemplated(_ context.Context, to, tmpl, lang string, data map[string]interface{}) error {
	m.sendTemplateCalled = true
	m.sentTo = to
	m.sentTemplate = tmpl
	m.sentLang = lang
	m.sentData = data
	return m.sendErr
}

func TestProcessPushTask_ValidPayload(t *testing.T) {
	push := &mockPushService{}
	email := &mockEmailService{}
	logger := zerolog.Nop()
	worker := notifications.NewNotificationWorker(push, email, logger)

	payload, err := json.Marshal(map[string]interface{}{
		"userId": "user-42",
		"notification": map[string]interface{}{
			"title": "Document Ready",
			"body":  "Your document has been processed.",
		},
		"type": notifications.PushTypeDocumentComplete,
	})
	require.NoError(t, err)

	task := asynq.NewTask(notifications.TaskSendPush, payload)
	err = worker.ProcessPushTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, push.sendCalled, "PushService.SendToUser should be called")
	assert.Equal(t, "user-42", push.sendUserID)
	assert.Equal(t, "Document Ready", push.sendNotif.Title)
	assert.Equal(t, notifications.PushTypeDocumentComplete, push.sendType)
}

func TestProcessPushTask_InvalidPayload(t *testing.T) {
	push := &mockPushService{}
	email := &mockEmailService{}
	logger := zerolog.Nop()
	worker := notifications.NewNotificationWorker(push, email, logger)

	task := asynq.NewTask(notifications.TaskSendPush, []byte(`{invalid json`))
	err := worker.ProcessPushTask(context.Background(), task)

	require.Error(t, err, "invalid JSON payload must return an error")
	assert.False(t, push.sendCalled, "PushService should not be called on bad payload")
}

func TestProcessEmailTask_ValidPayload(t *testing.T) {
	push := &mockPushService{}
	email := &mockEmailService{}
	logger := zerolog.Nop()
	worker := notifications.NewNotificationWorker(push, email, logger)

	payload, err := json.Marshal(map[string]interface{}{
		"to":       "patient@example.com",
		"template": "consent_granted",
		"lang":     "en",
		"data": map[string]interface{}{
			"patientName":   "Jane Doe",
			"physicianName": "Dr. Smith",
		},
	})
	require.NoError(t, err)

	task := asynq.NewTask(notifications.TaskSendEmail, payload)
	err = worker.ProcessEmailTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, email.sendTemplateCalled, "EmailService.SendTemplated should be called")
	assert.Equal(t, "patient@example.com", email.sentTo)
	assert.Equal(t, "consent_granted", email.sentTemplate)
	assert.Equal(t, "en", email.sentLang)
}

func TestProcessEmailTask_InvalidPayload(t *testing.T) {
	push := &mockPushService{}
	email := &mockEmailService{}
	logger := zerolog.Nop()
	worker := notifications.NewNotificationWorker(push, email, logger)

	task := asynq.NewTask(notifications.TaskSendEmail, []byte(`not json!!!`))
	err := worker.ProcessEmailTask(context.Background(), task)

	require.Error(t, err, "invalid JSON payload must return an error")
	assert.False(t, email.sendTemplateCalled, "EmailService should not be called on bad payload")
}
