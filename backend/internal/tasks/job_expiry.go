package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type JobExpiryTask struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewJobExpiryTask(db *sqlx.DB, logger zerolog.Logger) *JobExpiryTask {
	return &JobExpiryTask{db: db, logger: logger}
}

func (t *JobExpiryTask) Process(ctx context.Context, _ *asynq.Task) error {
	t.logger.Info().Msg("starting stale document job expiry")

	// Stuck in processing > 10 minutes (worker crash recovery)
	res, err := t.db.ExecContext(ctx,
		`UPDATE document_jobs
		 SET status = 'failed', error_message = 'Processing timeout: worker may have crashed', completed_at = NOW()
		 WHERE status = 'processing' AND processing_started_at < NOW() - INTERVAL '10 minutes'`)
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to expire stuck processing jobs")
	} else {
		rows, _ := res.RowsAffected()
		t.logger.Info().Int64("expired", rows).Msg("expired stuck processing jobs")
	}

	// Stuck in pending > 30 minutes (Asynq task lost)
	res, err = t.db.ExecContext(ctx,
		`UPDATE document_jobs
		 SET status = 'failed', error_message = 'Pending timeout: task may have been lost', completed_at = NOW()
		 WHERE status = 'pending' AND uploaded_at < NOW() - INTERVAL '30 minutes'`)
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to expire stuck pending jobs")
	} else {
		rows, _ := res.RowsAffected()
		t.logger.Info().Int64("expired", rows).Msg("expired stuck pending jobs")
	}

	t.logger.Info().Msg("stale document job expiry complete")
	return nil
}
