package notifications

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// PostgresPrefsRepository implements NotificationPrefsRepository with PostgreSQL.
type PostgresPrefsRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewPostgresPrefsRepository creates a new repository backed by PostgreSQL.
func NewPostgresPrefsRepository(db *sqlx.DB, logger zerolog.Logger) *PostgresPrefsRepository {
	return &PostgresPrefsRepository{db: db, logger: logger}
}

// defaultPreferences returns a NotificationPreferences with sensible defaults.
func defaultPreferences(userID string) *NotificationPreferences {
	return &NotificationPreferences{
		ID:                    uuid.New().String(),
		UserID:                userID,
		EmailDocumentComplete: true,
		EmailDocumentFailed:   true,
		EmailConsentGranted:   true,
		EmailConsentRevoked:   true,
		EmailBreakGlass:       true,
		EmailAccountLocked:    true,
		PushEnabled:           true,
		PushDocumentComplete:  true,
		PushNewPrescription:   true,
		PushLabResultReady:    true,
		PushConsentRequest:    false,
		PushCriticalLab:       true,
		PreferredLanguage:     "en",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// GetPreferences retrieves preferences for a user, creating defaults if none exist.
func (r *PostgresPrefsRepository) GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error) {
	var prefs NotificationPreferences
	err := r.db.GetContext(ctx, &prefs,
		`SELECT * FROM notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create defaults via INSERT ON CONFLICT
			defaults := defaultPreferences(userID)
			_, insertErr := r.db.ExecContext(ctx,
				`INSERT INTO notification_preferences (
					id, user_id,
					email_document_complete, email_document_failed,
					email_consent_granted, email_consent_revoked,
					email_break_glass, email_account_locked,
					push_enabled, push_document_complete,
					push_new_prescription, push_lab_result_ready,
					push_consent_request, push_critical_lab,
					fcm_token, preferred_language,
					created_at, updated_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
				ON CONFLICT (user_id) DO NOTHING`,
				defaults.ID, defaults.UserID,
				defaults.EmailDocumentComplete, defaults.EmailDocumentFailed,
				defaults.EmailConsentGranted, defaults.EmailConsentRevoked,
				defaults.EmailBreakGlass, defaults.EmailAccountLocked,
				defaults.PushEnabled, defaults.PushDocumentComplete,
				defaults.PushNewPrescription, defaults.PushLabResultReady,
				defaults.PushConsentRequest, defaults.PushCriticalLab,
				defaults.FCMToken, defaults.PreferredLanguage,
				defaults.CreatedAt, defaults.UpdatedAt,
			)
			if insertErr != nil {
				r.logger.Error().Err(insertErr).Str("user_id", userID).Msg("failed to insert default preferences")
				return defaults, nil
			}
			// Re-fetch to get the actual row (handles race with ON CONFLICT)
			err2 := r.db.GetContext(ctx, &prefs,
				`SELECT * FROM notification_preferences WHERE user_id = $1`, userID)
			if err2 != nil {
				return defaults, nil
			}
			return &prefs, nil
		}
		return nil, err
	}
	return &prefs, nil
}

// UpsertPreferences updates or inserts notification preferences for a user.
func (r *PostgresPrefsRepository) UpsertPreferences(ctx context.Context, userID string, prefs *NotificationPreferences) error {
	if prefs.ID == "" {
		prefs.ID = uuid.New().String()
	}
	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_preferences (
			id, user_id,
			email_document_complete, email_document_failed,
			email_consent_granted, email_consent_revoked,
			email_break_glass, email_account_locked,
			push_enabled, push_document_complete,
			push_new_prescription, push_lab_result_ready,
			push_consent_request, push_critical_lab,
			preferred_language, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW(),$16)
		ON CONFLICT (user_id) DO UPDATE SET
			email_document_complete = EXCLUDED.email_document_complete,
			email_document_failed = EXCLUDED.email_document_failed,
			email_consent_granted = EXCLUDED.email_consent_granted,
			email_consent_revoked = EXCLUDED.email_consent_revoked,
			email_break_glass = EXCLUDED.email_break_glass,
			email_account_locked = EXCLUDED.email_account_locked,
			push_enabled = EXCLUDED.push_enabled,
			push_document_complete = EXCLUDED.push_document_complete,
			push_new_prescription = EXCLUDED.push_new_prescription,
			push_lab_result_ready = EXCLUDED.push_lab_result_ready,
			push_consent_request = EXCLUDED.push_consent_request,
			push_critical_lab = EXCLUDED.push_critical_lab,
			preferred_language = EXCLUDED.preferred_language,
			updated_at = EXCLUDED.updated_at`,
		prefs.ID, prefs.UserID,
		prefs.EmailDocumentComplete, prefs.EmailDocumentFailed,
		prefs.EmailConsentGranted, prefs.EmailConsentRevoked,
		prefs.EmailBreakGlass, prefs.EmailAccountLocked,
		prefs.PushEnabled, prefs.PushDocumentComplete,
		prefs.PushNewPrescription, prefs.PushLabResultReady,
		prefs.PushConsentRequest, prefs.PushCriticalLab,
		prefs.PreferredLanguage, prefs.UpdatedAt,
	)
	return err
}

// UpdateFCMToken sets the FCM push token for a user, upserting the preferences row.
func (r *PostgresPrefsRepository) UpdateFCMToken(ctx context.Context, userID, token string) error {
	defaults := defaultPreferences(userID)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification_preferences (
			id, user_id, fcm_token,
			email_document_complete, email_document_failed,
			email_consent_granted, email_consent_revoked,
			email_break_glass, email_account_locked,
			push_enabled, push_document_complete,
			push_new_prescription, push_lab_result_ready,
			push_consent_request, push_critical_lab,
			preferred_language, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (user_id) DO UPDATE SET fcm_token = $3, updated_at = NOW()`,
		defaults.ID, userID, token,
		defaults.EmailDocumentComplete, defaults.EmailDocumentFailed,
		defaults.EmailConsentGranted, defaults.EmailConsentRevoked,
		defaults.EmailBreakGlass, defaults.EmailAccountLocked,
		defaults.PushEnabled, defaults.PushDocumentComplete,
		defaults.PushNewPrescription, defaults.PushLabResultReady,
		defaults.PushConsentRequest, defaults.PushCriticalLab,
		defaults.PreferredLanguage, defaults.CreatedAt, defaults.UpdatedAt,
	)
	return err
}

// ClearFCMToken removes the FCM push token for a user.
func (r *PostgresPrefsRepository) ClearFCMToken(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notification_preferences SET fcm_token = '', updated_at = NOW() WHERE user_id = $1`,
		userID)
	return err
}
