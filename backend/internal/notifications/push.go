package notifications

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/Siddharthk17/MediLink/internal/config"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

// PushNotification describes a push notification payload.
type PushNotification struct {
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Data     map[string]string `json:"data,omitempty"`
	ImageURL string            `json:"imageUrl,omitempty"`
}

const (
	PushTypeDocumentComplete = "document_complete"
	PushTypeDocumentFailed   = "document_failed"
	PushTypeNewPrescription  = "new_prescription"
	PushTypeLabResultReady   = "lab_result_ready"
	PushTypeConsentRequest   = "consent_request"
	PushTypeCriticalLab      = "critical_lab"
	PushTypeConsentGranted   = "consent_granted"
	PushTypeConsentRevoked   = "consent_revoked"
)

// PushService sends push notifications to user devices.
type PushService interface {
	SendToUser(ctx context.Context, userID string, notif PushNotification, notifType string) error
	RegisterToken(ctx context.Context, userID, fcmToken string) error
	RevokeToken(ctx context.Context, userID string) error
	Health(ctx context.Context) bool
}

type fcmPushService struct {
	client   *messaging.Client
	prefRepo NotificationPrefsRepository
	logger   zerolog.Logger
}

// NewFCMPushService creates a PushService backed by Firebase Cloud Messaging.
func NewFCMPushService(cfg *config.Config, prefRepo NotificationPrefsRepository, logger zerolog.Logger) (PushService, error) {
	opt := option.WithCredentialsFile(cfg.Firebase.ServiceAccountPath)
	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: cfg.Firebase.ProjectID}, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase.NewApp: %w", err)
	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("app.Messaging: %w", err)
	}
	logger.Info().Str("project_id", cfg.Firebase.ProjectID).Msg("Firebase Admin SDK initialized")
	return &fcmPushService{client: client, prefRepo: prefRepo, logger: logger}, nil
}

// SendToUser checks preferences → token → sends. NEVER returns error on FCM failure.
func (s *fcmPushService) SendToUser(ctx context.Context, userID string, notif PushNotification, notifType string) error {
	prefs, err := s.prefRepo.GetPreferences(ctx, userID)
	if err != nil {
		s.logger.Warn().Str("user_id", userID).Msg("no notification preferences found, using defaults")
	}
	if prefs != nil && !prefs.PushEnabled {
		return nil
	}
	if prefs != nil && !s.isTypeEnabled(prefs, notifType) {
		return nil
	}
	if prefs == nil || prefs.FCMToken == nil || *prefs.FCMToken == "" {
		return nil
	}

	badge := 1
	msg := &messaging.Message{
		Token: *prefs.FCMToken,
		Notification: &messaging.Notification{
			Title:    notif.Title,
			Body:     notif.Body,
			ImageURL: notif.ImageURL,
		},
		Data: notif.Data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Sound:       "default",
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "default",
					Badge: &badge,
				},
			},
		},
	}

	_, err = s.client.Send(ctx, msg)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("type", notifType).Msg("FCM send failed")
		return nil // NEVER return error on FCM failure
	}
	return nil
}

// RegisterToken stores an FCM device token for a user.
func (s *fcmPushService) RegisterToken(ctx context.Context, userID, token string) error {
	return s.prefRepo.UpdateFCMToken(ctx, userID, token)
}

// RevokeToken clears the FCM device token for a user.
func (s *fcmPushService) RevokeToken(ctx context.Context, userID string) error {
	return s.prefRepo.ClearFCMToken(ctx, userID)
}

// Health returns true if the FCM client is initialized.
func (s *fcmPushService) Health(_ context.Context) bool { return s.client != nil }

func (s *fcmPushService) isTypeEnabled(prefs *NotificationPreferences, t string) bool {
	switch t {
	case PushTypeDocumentComplete:
		return prefs.PushDocumentComplete
	case PushTypeDocumentFailed:
		return true // always send failure notifications
	case PushTypeNewPrescription:
		return prefs.PushNewPrescription
	case PushTypeLabResultReady:
		return prefs.PushLabResultReady
	case PushTypeConsentRequest:
		return prefs.PushConsentRequest
	case PushTypeCriticalLab:
		return prefs.PushCriticalLab
	default:
		return true
	}
}

// NoopPushService is a no-op implementation for dev/test when Firebase is unavailable.
type NoopPushService struct {
	logger zerolog.Logger
}

// NewNoopPushService creates a no-op push service.
func NewNoopPushService(logger zerolog.Logger) PushService {
	return &NoopPushService{logger: logger}
}

func (n *NoopPushService) SendToUser(_ context.Context, userID string, notif PushNotification, notifType string) error {
	n.logger.Debug().
		Str("user_id", userID).
		Str("type", notifType).
		Str("title", notif.Title).
		Msg("noop push: would send notification")
	return nil
}

func (n *NoopPushService) RegisterToken(_ context.Context, userID, token string) error {
	n.logger.Debug().Str("user_id", userID).Msg("noop push: would register token")
	return nil
}

func (n *NoopPushService) RevokeToken(_ context.Context, userID string) error {
	n.logger.Debug().Str("user_id", userID).Msg("noop push: would revoke token")
	return nil
}

func (n *NoopPushService) Health(_ context.Context) bool { return true }
