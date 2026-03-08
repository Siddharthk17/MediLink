package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

const (
	// TaskSendEmail is the Asynq task type for sending emails.
	TaskSendEmail = "notification:email"
	// TaskSendPush is the Asynq task type for sending push notifications.
	TaskSendPush = "notification:push"
)

// PushTaskPayload is the JSON payload for a push notification task.
type PushTaskPayload struct {
	UserID string           `json:"userId"`
	Notif  PushNotification `json:"notification"`
	Type   string           `json:"type"`
}

// EmailTaskPayload is the JSON payload for an email notification task.
type EmailTaskPayload struct {
	To           string                 `json:"to"`
	TemplateName string                 `json:"template"`
	Language     string                 `json:"lang"`
	Data         map[string]interface{} `json:"data"`
}

// NotificationWorker processes async notification tasks via Asynq.
type NotificationWorker struct {
	pushSvc  PushService
	emailSvc EmailService
	logger   zerolog.Logger
}

// NewNotificationWorker creates a NotificationWorker.
func NewNotificationWorker(pushSvc PushService, emailSvc EmailService, logger zerolog.Logger) *NotificationWorker {
	return &NotificationWorker{pushSvc: pushSvc, emailSvc: emailSvc, logger: logger}
}

// ProcessPushTask handles a push notification Asynq task.
func (w *NotificationWorker) ProcessPushTask(ctx context.Context, t *asynq.Task) error {
	var payload PushTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal push task payload")
		return fmt.Errorf("unmarshal push payload: %w", err)
	}

	w.logger.Info().
		Str("user_id", payload.UserID).
		Str("type", payload.Type).
		Str("title", payload.Notif.Title).
		Msg("processing push notification task")

	if err := w.pushSvc.SendToUser(ctx, payload.UserID, payload.Notif, payload.Type); err != nil {
		w.logger.Error().Err(err).Str("user_id", payload.UserID).Msg("push task failed")
		// Push failures are non-fatal — don't retry
		return nil
	}
	return nil
}

// ProcessEmailTask handles an email notification Asynq task.
func (w *NotificationWorker) ProcessEmailTask(ctx context.Context, t *asynq.Task) error {
	var payload EmailTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal email task payload")
		return fmt.Errorf("unmarshal email payload: %w", err)
	}

	w.logger.Info().
		Str("to", payload.To).
		Str("template", payload.TemplateName).
		Str("lang", payload.Language).
		Msg("processing email notification task")

	if err := w.emailSvc.SendTemplated(ctx, payload.To, payload.TemplateName, payload.Language, payload.Data); err != nil {
		w.logger.Error().Err(err).
			Str("to", payload.To).
			Str("template", payload.TemplateName).
			Msg("email task failed")
		return fmt.Errorf("send templated email: %w", err)
	}
	return nil
}

// NewPushTask creates an Asynq task for sending a push notification.
func NewPushTask(userID string, notif PushNotification, notifType string) (*asynq.Task, error) {
	payload, err := json.Marshal(PushTaskPayload{
		UserID: userID,
		Notif:  notif,
		Type:   notifType,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskSendPush, payload, asynq.Queue("notifications")), nil
}

// NewEmailTask creates an Asynq task for sending a templated email.
func NewEmailTask(to, templateName, lang string, data map[string]interface{}) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailTaskPayload{
		To:           to,
		TemplateName: templateName,
		Language:     lang,
		Data:         data,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskSendEmail, payload, asynq.Queue("notifications")), nil
}
