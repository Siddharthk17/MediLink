package anonymization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Siddharthk17/MediLink/pkg/storage"
	"github.com/rs/zerolog"
)

// Exporter writes anonymized NDJSON files and manifests to object storage.
type Exporter struct {
	storage storage.StorageClient
	logger  zerolog.Logger
}

// NewExporter creates a new Exporter.
func NewExporter(sc storage.StorageClient, logger zerolog.Logger) *Exporter {
	return &Exporter{storage: sc, logger: logger}
}

// WriteNDJSON writes each resource type as an NDJSON file under
// exports/{exportID}/{ResourceType}.ndjson in the given bucket.
// Returns total bytes written across all files.
func (e *Exporter) WriteNDJSON(ctx context.Context, bucket, exportID string, records map[string][]AnonymizedRecord) (int64, error) {
	var totalSize int64
	for resourceType, recs := range records {
		var buf bytes.Buffer
		for _, r := range recs {
			line, err := json.Marshal(r.Data)
			if err != nil {
				e.logger.Warn().Err(err).
					Str("resourceType", resourceType).
					Msg("skipping record that failed to marshal")
				continue
			}
			buf.Write(line)
			buf.WriteByte('\n')
		}

		key := fmt.Sprintf("exports/%s/%s.ndjson", exportID, resourceType)
		size := int64(buf.Len())
		if err := e.storage.UploadWithKey(ctx, bucket, key, &buf, "application/x-ndjson", size); err != nil {
			return totalSize, fmt.Errorf("upload %s: %w", key, err)
		}
		totalSize += size

		e.logger.Info().
			Str("bucket", bucket).
			Str("key", key).
			Int("records", len(recs)).
			Msg("NDJSON file written")
	}
	return totalSize, nil
}

// WriteManifest writes a manifest.json describing the export contents.
func (e *Exporter) WriteManifest(ctx context.Context, bucket, exportID string, manifest map[string]interface{}) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	key := fmt.Sprintf("exports/%s/manifest.json", exportID)
	buf := bytes.NewReader(data)
	if err := e.storage.UploadWithKey(ctx, bucket, key, buf, "application/json", int64(len(data))); err != nil {
		return fmt.Errorf("upload manifest: %w", err)
	}
	return nil
}
