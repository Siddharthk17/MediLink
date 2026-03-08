package tasks

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type StatsSnapshotTask struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewStatsSnapshotTask(db *sqlx.DB, logger zerolog.Logger) *StatsSnapshotTask {
	return &StatsSnapshotTask{db: db, logger: logger}
}

type roleStat struct {
	Role  string `db:"role"`
	Count int64  `db:"count"`
}

type resourceStat struct {
	ResourceType string `db:"resource_type"`
	Count        int64  `db:"count"`
}

type jobStat struct {
	Status string `db:"status"`
	Count  int64  `db:"count"`
}

func (t *StatsSnapshotTask) Process(ctx context.Context, _ *asynq.Task) error {
	t.logger.Info().Msg("starting daily stats snapshot")

	// User counts by role
	var roles []roleStat
	if err := t.db.SelectContext(ctx, &roles,
		"SELECT role, COUNT(*) AS count FROM users WHERE deleted_at IS NULL GROUP BY role"); err != nil {
		t.logger.Error().Err(err).Msg("failed to query user stats")
	} else {
		ev := t.logger.Info()
		for _, r := range roles {
			ev = ev.Int64(r.Role, r.Count)
		}
		ev.Msg("user counts by role")
	}

	// FHIR resource counts by type
	var resources []resourceStat
	if err := t.db.SelectContext(ctx, &resources,
		"SELECT resource_type, COUNT(*) AS count FROM fhir_resources WHERE deleted_at IS NULL GROUP BY resource_type"); err != nil {
		t.logger.Error().Err(err).Msg("failed to query FHIR resource stats")
	} else {
		ev := t.logger.Info()
		for _, r := range resources {
			ev = ev.Int64(r.ResourceType, r.Count)
		}
		ev.Msg("FHIR resource counts by type")
	}

	// Document job counts by status
	var jobs []jobStat
	if err := t.db.SelectContext(ctx, &jobs,
		"SELECT status, COUNT(*) AS count FROM document_jobs GROUP BY status"); err != nil {
		t.logger.Error().Err(err).Msg("failed to query document job stats")
	} else {
		ev := t.logger.Info()
		for _, j := range jobs {
			ev = ev.Int64(j.Status, j.Count)
		}
		ev.Msg("document job counts by status")
	}

	t.logger.Info().Msg("daily stats snapshot complete")
	return nil
}
