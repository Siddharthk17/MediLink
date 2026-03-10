package notifications

import (
	"context"
	"time"
)

// NotificationPreferences stores per-user notification settings.
type NotificationPreferences struct {
	ID                    string    `json:"id" db:"id"`
	UserID                string    `json:"userId" db:"user_id"`
	EmailDocumentComplete bool      `json:"emailDocumentComplete" db:"email_document_complete"`
	EmailDocumentFailed   bool      `json:"emailDocumentFailed" db:"email_document_failed"`
	EmailConsentGranted   bool      `json:"emailConsentGranted" db:"email_consent_granted"`
	EmailConsentRevoked   bool      `json:"emailConsentRevoked" db:"email_consent_revoked"`
	EmailBreakGlass       bool      `json:"emailBreakGlass" db:"email_break_glass"`
	EmailAccountLocked    bool      `json:"emailAccountLocked" db:"email_account_locked"`
	PushEnabled           bool      `json:"pushEnabled" db:"push_enabled"`
	PushDocumentComplete  bool      `json:"pushDocumentComplete" db:"push_document_complete"`
	PushNewPrescription   bool      `json:"pushNewPrescription" db:"push_new_prescription"`
	PushLabResultReady    bool      `json:"pushLabResultReady" db:"push_lab_result_ready"`
	PushConsentRequest    bool      `json:"pushConsentRequest" db:"push_consent_request"`
	PushCriticalLab       bool      `json:"pushCriticalLab" db:"push_critical_lab"`
	FCMToken              *string    `json:"-" db:"fcm_token"`
	FCMTokenUpdatedAt     *time.Time `json:"-" db:"fcm_token_updated_at"`
	PreferredLanguage     string    `json:"preferredLanguage" db:"preferred_language"`
	CreatedAt             time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time `json:"updatedAt" db:"updated_at"`
}

// NotificationPrefsRepository provides data access for notification preferences.
type NotificationPrefsRepository interface {
	GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error)
	UpsertPreferences(ctx context.Context, userID string, prefs *NotificationPreferences) error
	UpdateFCMToken(ctx context.Context, userID, token string) error
	ClearFCMToken(ctx context.Context, userID string) error
}
