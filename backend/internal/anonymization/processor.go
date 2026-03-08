package anonymization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// ExportProcessor handles the asynchronous anonymized-export Asynq task.
type ExportProcessor struct {
	db       *sqlx.DB
	repo     ExportRepository
	exporter *Exporter
	logger   zerolog.Logger
}

// NewExportProcessor creates a new ExportProcessor.
func NewExportProcessor(db *sqlx.DB, repo ExportRepository, exporter *Exporter, logger zerolog.Logger) *ExportProcessor {
	return &ExportProcessor{
		db:       db,
		repo:     repo,
		exporter: exporter,
		logger:   logger,
	}
}

// Process is the Asynq handler for TaskAnonymizedExport.
func (p *ExportProcessor) Process(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		ExportID string `json:"exportId"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	exportID := payload.ExportID
	p.logger.Info().Str("exportId", exportID).Msg("starting anonymized export")

	// 1. Load the export record.
	export, err := p.repo.GetByID(ctx, exportID)
	if err != nil {
		return fmt.Errorf("load export: %w", err)
	}

	// 2. Update status to "processing".
	if err := p.repo.UpdateStatus(ctx, exportID, "processing", nil); err != nil {
		return fmt.Errorf("set processing: %w", err)
	}

	// Helper to mark failure and return nil (no Asynq retry).
	fail := func(msg string) error {
		now := time.Now().UTC()
		_ = p.repo.UpdateStatus(ctx, exportID, "failed", map[string]interface{}{
			"error_message": msg,
			"completed_at":  now,
		})
		p.logger.Error().Str("exportId", exportID).Msg(msg)
		return nil
	}

	// 3. Query fhir_resources for the requested types (with optional date filters).
	type fhirRow struct {
		ResourceType string          `db:"resource_type"`
		ResourceID   string          `db:"resource_id"`
		Data         json.RawMessage `db:"data"`
	}

	query := `SELECT resource_type, resource_id, data FROM fhir_resources
		WHERE resource_type = ANY($1) AND deleted_at IS NULL`
	args := []interface{}{export.ResourceTypes}
	argIdx := 2

	if export.DateFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *export.DateFrom)
		argIdx++
	}
	if export.DateTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *export.DateTo)
		argIdx++
	}

	var rows []fhirRow
	if err := p.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return fail("query fhir_resources: " + err.Error())
	}

	totalRecords := len(rows)
	if totalRecords == 0 {
		return fail("no records matched the requested resource types / date range")
	}

	// 4. Anonymize each resource and build patient quasi-identifier maps.
	//    Patient quasi-identifiers (ageBand, gender) are extracted from Patient
	//    resources and then propagated to related resources via subject reference.
	type patientQI struct {
		AgeBand string
		Gender  string
	}
	patientMap := make(map[string]patientQI) // original Patient resource_id → QI

	// First pass: collect patient QIs from raw (pre-anonymize) data.
	for _, row := range rows {
		if row.ResourceType != "Patient" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(row.Data, &raw); err != nil {
			continue
		}
		birthYear := ExtractBirthYear(raw)
		gender := ExtractGender(raw)
		patientMap[row.ResourceID] = patientQI{
			AgeBand: AgeBand(birthYear),
			Gender:  gender,
		}
	}

	// Build a function that extracts the primary condition code for a patient.
	conditionMap := make(map[string]string) // patient resourceID → primary condition code
	for _, row := range rows {
		if row.ResourceType != "Condition" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(row.Data, &raw); err != nil {
			continue
		}
		patientRef := extractSubjectRef(raw)
		if patientRef == "" {
			continue
		}
		// Keep first condition only as "primary".
		if _, exists := conditionMap[patientRef]; !exists {
			conditionMap[patientRef] = ExtractPrimaryConditionCode(raw)
		}
	}

	// Second pass: anonymize and build AnonymizedRecord list.
	allRecords := make([]AnonymizedRecord, 0, totalRecords)
	for _, row := range rows {
		var raw map[string]interface{}
		if err := json.Unmarshal(row.Data, &raw); err != nil {
			p.logger.Warn().Err(err).Str("resourceId", row.ResourceID).Msg("skip unmarshal error")
			continue
		}

		anon := AnonymizeResource(row.ResourceType, raw, export.ExportSalt)

		// Determine quasi-identifiers for this record.
		var qi patientQI
		if row.ResourceType == "Patient" {
			qi = patientMap[row.ResourceID]
		} else {
			patientRef := extractSubjectRef(raw)
			qi = patientMap[patientRef]
		}

		rec := AnonymizedRecord{
			ResourceType:         row.ResourceType,
			ResourceID:           row.ResourceID,
			Data:                 anon,
			AgeBand:              qi.AgeBand,
			Gender:               qi.Gender,
			PrimaryConditionCode: conditionMap[extractSubjectRef(raw)],
		}
		allRecords = append(allRecords, rec)
	}

	// 5. Apply k-anonymity.
	kept, stats := ApplyKAnonymity(allRecords, export.KValue)

	p.logger.Info().
		Str("exportId", exportID).
		Int("total", stats.TotalRecords).
		Int("kept", stats.KeptRecords).
		Int("suppressed", stats.SuppressedRecords).
		Msg("k-anonymity applied")

	// 6. Group kept records by resource type for NDJSON output.
	grouped := make(map[string][]AnonymizedRecord)
	for _, r := range kept {
		grouped[r.ResourceType] = append(grouped[r.ResourceType], r)
	}

	// 7. Write NDJSON files.
	totalSize, err := p.exporter.WriteNDJSON(ctx, export.MinioBucket, exportID, grouped)
	if err != nil {
		return fail("write NDJSON: " + err.Error())
	}

	// 8. Write manifest.json.
	fileSummary := make(map[string]int)
	for rt, recs := range grouped {
		fileSummary[rt] = len(recs)
	}

	manifest := map[string]interface{}{
		"exportId":    exportID,
		"exportedAt":  time.Now().UTC().Format(time.RFC3339),
		"kValue":      export.KValue,
		"purpose":     export.Purpose,
		"statistics":  stats,
		"files":       fileSummary,
		"totalSizeBytes": totalSize,
	}
	if err := p.exporter.WriteManifest(ctx, export.MinioBucket, exportID, manifest); err != nil {
		return fail("write manifest: " + err.Error())
	}

	// 9. Update DB with final results.
	now := time.Now().UTC()
	minioKey := "exports/" + exportID
	_ = p.repo.UpdateStatus(ctx, exportID, "completed", map[string]interface{}{
		"record_count":    totalRecords,
		"exported_count":  stats.KeptRecords,
		"suppressed_count": stats.SuppressedRecords,
		"minio_key":       minioKey,
		"file_size":       totalSize,
		"completed_at":    now,
	})

	p.logger.Info().
		Str("exportId", exportID).
		Int64("fileSize", totalSize).
		Msg("anonymized export completed")

	return nil
}

// extractSubjectRef extracts the patient resource ID from a FHIR resource's
// subject or patient reference field (e.g. "Patient/abc-123" → "abc-123").
func extractSubjectRef(data map[string]interface{}) string {
	for _, field := range []string{"subject", "patient"} {
		switch v := data[field].(type) {
		case map[string]interface{}:
			if ref, ok := v["reference"].(string); ok {
				return stripResourcePrefix(ref)
			}
		case string:
			return stripResourcePrefix(v)
		}
	}
	return ""
}

func stripResourcePrefix(ref string) string {
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			return ref[i+1:]
		}
	}
	return ref
}
