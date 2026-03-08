package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type TokenCleanupTask struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewTokenCleanupTask(db *sqlx.DB, logger zerolog.Logger) *TokenCleanupTask {
	return &TokenCleanupTask{db: db, logger: logger}
}

func (t *TokenCleanupTask) Process(ctx context.Context, _ *asynq.Task) error {
	t.logger.Info().Msg("starting expired token cleanup")

	res, err := t.db.ExecContext(ctx,
		"DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '1 day'")
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to clean refresh_tokens")
	} else {
		rows, _ := res.RowsAffected()
		t.logger.Info().Int64("deleted", rows).Msg("cleaned refresh_tokens")
	}

	res, err = t.db.ExecContext(ctx,
		"DELETE FROM email_otps WHERE expires_at < NOW() - INTERVAL '1 day'")
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to clean email_otps")
	} else {
		rows, _ := res.RowsAffected()
		t.logger.Info().Int64("deleted", rows).Msg("cleaned email_otps")
	}

	res, err = t.db.ExecContext(ctx,
		"DELETE FROM login_attempts WHERE attempted_at < NOW() - INTERVAL '30 days'")
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to clean login_attempts")
	} else {
		rows, _ := res.RowsAffected()
		t.logger.Info().Int64("deleted", rows).Msg("cleaned login_attempts")
	}

	t.logger.Info().Msg("expired token cleanup complete")
	return nil
}
