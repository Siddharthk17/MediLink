package tasks

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/pkg/search"
)

type ESReindexTask struct {
	db     *sqlx.DB
	es     search.SearchClient
	logger zerolog.Logger
}

func NewESReindexTask(db *sqlx.DB, es search.SearchClient, logger zerolog.Logger) *ESReindexTask {
	return &ESReindexTask{db: db, es: es, logger: logger}
}

type fhirResourceRow struct {
	ResourceType string          `db:"resource_type"`
	ResourceID   string          `db:"resource_id"`
	Data         json.RawMessage `db:"data"`
}

func (t *ESReindexTask) Process(ctx context.Context, _ *asynq.Task) error {
	t.logger.Info().Msg("starting ES reindex of recently updated resources")

	var resources []fhirResourceRow
	err := t.db.SelectContext(ctx, &resources,
		`SELECT resource_type, resource_id, data FROM fhir_resources
		 WHERE updated_at >= NOW() - INTERVAL '3 hours' AND deleted_at IS NULL`)
	if err != nil {
		t.logger.Error().Err(err).Msg("failed to query fhir_resources for reindex")
		return nil
	}

	t.logger.Info().Int("count", len(resources)).Msg("resources to reindex")

	var indexed, failed int
	for _, r := range resources {
		if err := t.es.IndexResource(ctx, r.ResourceType, r.ResourceID, r.Data); err != nil {
			t.logger.Warn().Err(err).
				Str("resource_type", r.ResourceType).
				Str("resource_id", r.ResourceID).
				Msg("failed to reindex resource")
			failed++
		} else {
			indexed++
		}
	}

	t.logger.Info().
		Int("indexed", indexed).
		Int("failed", failed).
		Msg("ES reindex complete")
	return nil
}
