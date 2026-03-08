package tasks

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type ConsentExpiryTask struct {
	db     *sqlx.DB
	redis  *redis.Client
	logger zerolog.Logger
}

func NewConsentExpiryTask(db *sqlx.DB, redis *redis.Client, logger zerolog.Logger) *ConsentExpiryTask {
	return &ConsentExpiryTask{db: db, redis: redis, logger: logger}
}

type expiredConsent struct {
	ID             string `db:"id"`
	PatientID      string `db:"patient_id"`
	ProviderID     string `db:"provider_id"`
	FHIRPatientID  string `db:"fhir_patient_id"`
}

func (t *ConsentExpiryTask) Process(ctx context.Context, _ *asynq.Task) error {
	t.logger.Info().Msg("starting expired consent revocation")

	var expired []expiredConsent
	err := t.db.SelectContext(ctx, &expired,
		`WITH expired_consents AS (
			UPDATE consents
			SET status = 'expired', revoked_at = NOW(), revoked_reason = 'auto-expired by system'
			WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < NOW()
			RETURNING id, patient_id, provider_id
		)
		SELECT e.id, e.patient_id, e.provider_id, COALESCE(u.fhir_patient_id, '') AS fhir_patient_id
		FROM expired_consents e
		LEFT JOIN users u ON e.patient_id = u.id`)
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to expire consents")
		return nil
	}

	t.logger.Info().Int("count", len(expired)).Msg("consents expired")

	for _, c := range expired {
		if c.FHIRPatientID == "" {
			t.logger.Warn().Str("consent_id", c.ID).Msg("no FHIR patient ID — skipping cache invalidation")
			continue
		}
		key := fmt.Sprintf("consent:%s:%s", c.ProviderID, c.FHIRPatientID)
		if err := t.redis.Del(ctx, key).Err(); err != nil {
			t.logger.Warn().Err(err).Str("key", key).Msg("failed to invalidate consent cache")
		}
	}

	t.logger.Info().Msg("expired consent revocation complete")
	return nil
}
