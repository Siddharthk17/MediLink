package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/fhir/repository"
	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// fhirBaseURL is the canonical base URL used in FHIR Bundle fullUrl fields.
var fhirBaseURL = initFHIRBaseURL()

func initFHIRBaseURL() string {
	if u := os.Getenv("FHIR_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://api.medilink.health"
}

// TimelineService provides the patient timeline aggregation API.
type TimelineService struct {
	db    *sqlx.DB
	audit audit.AuditLogger
}

// NewTimelineService creates a new TimelineService.
func NewTimelineService(db *sqlx.DB, al audit.AuditLogger) *TimelineService {
	return &TimelineService{db: db, audit: al}
}

// TimelineParams holds query parameters for the timeline endpoint.
type TimelineParams struct {
	PatientID     string
	Count         int
	Offset        int
	Since         string
	ResourceTypes []string
}

// GetTimeline returns a chronological Bundle of all resources for a patient.
func (s *TimelineService) GetTimeline(ctx context.Context, params TimelineParams, actorID uuid.UUID) (*BundleResponse, error) {
	if params.Count <= 0 {
		params.Count = 50
	}
	if params.Count > 200 {
		params.Count = 200
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Patient",
		ResourceID:   params.PatientID,
		Action:       "timeline",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   200,
	})

	// Verify patient exists
	patientRef := "Patient/" + params.PatientID
	var exists bool
	err := s.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL)`,
		params.PatientID,
	)
	if err != nil || !exists {
		return nil, fhirerrors.NewNotFoundError("Patient", params.PatientID)
	}

	// Build the timeline query
	var args []interface{}
	args = append(args, patientRef)
	argIdx := 2

	typeFilter := ""
	if len(params.ResourceTypes) > 0 {
		typeFilter = fmt.Sprintf("AND resource_type = ANY($%d)", argIdx)
		args = append(args, params.ResourceTypes)
		argIdx++
	}

	sinceFilter := ""
	if params.Since != "" {
		sinceFilter = fmt.Sprintf(`AND COALESCE(
			(data->>'effectiveDateTime')::timestamptz,
			(data->'period'->>'start')::timestamptz,
			(data->>'authoredOn')::timestamptz,
			(data->>'occurrenceDateTime')::timestamptz,
			(data->>'onsetDateTime')::timestamptz,
			(data->>'recordedDate')::timestamptz,
			created_at
		) >= $%d::timestamptz`, argIdx)
		args = append(args, params.Since)
		argIdx++
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM fhir_resources
		WHERE patient_ref = $1 AND deleted_at IS NULL %s %s`,
		typeFilter, sinceFilter,
	)
	var total int
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, fhirerrors.NewProcessingError("failed to count timeline entries")
	}

	// Get timeline entries
	dataQuery := fmt.Sprintf(`
		SELECT
			resource_type, resource_id, data,
			COALESCE(
				(data->>'effectiveDateTime')::timestamptz,
				(data->'period'->>'start')::timestamptz,
				(data->>'authoredOn')::timestamptz,
				(data->>'occurrenceDateTime')::timestamptz,
				(data->>'onsetDateTime')::timestamptz,
				(data->>'recordedDate')::timestamptz,
				created_at
			) AS created_at
		FROM fhir_resources
		WHERE patient_ref = $1 AND deleted_at IS NULL %s %s
		ORDER BY COALESCE(
			(data->>'effectiveDateTime')::timestamptz,
			(data->'period'->>'start')::timestamptz,
			(data->>'authoredOn')::timestamptz,
			(data->>'occurrenceDateTime')::timestamptz,
			(data->>'onsetDateTime')::timestamptz,
			(data->>'recordedDate')::timestamptz,
			created_at
		) DESC
		LIMIT $%d OFFSET $%d`,
		typeFilter, sinceFilter, argIdx, argIdx+1,
	)
	args = append(args, params.Count, params.Offset)

	rows, err := s.db.QueryxContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to query timeline")
	}
	defer rows.Close()

	entries := make([]map[string]interface{}, 0)
	for rows.Next() {
		var resourceType, resourceID string
		var data json.RawMessage
		var eventDate time.Time

		if err := rows.Scan(&resourceType, &resourceID, &data, &eventDate); err != nil {
			continue
		}

		var resourceData interface{}
		json.Unmarshal(data, &resourceData)

		entry := map[string]interface{}{
			"fullUrl":  fmt.Sprintf("%s/fhir/R4/%s/%s", fhirBaseURL, resourceType, resourceID),
			"resource": resourceData,
			"search": map[string]interface{}{
				"mode": "match",
				"extension": []map[string]interface{}{
					{
						"url":           "https://medilink.health/fhir/extensions/timeline-date",
						"valueDateTime": eventDate.Format(time.RFC3339),
					},
					{
						"url":         "https://medilink.health/fhir/extensions/timeline-type",
						"valueString": resourceType,
					},
				},
			},
		}
		entries = append(entries, entry)
	}

	bundle := map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        total,
		"id":           uuid.New().String(),
		"meta": map[string]interface{}{
			"lastUpdated": time.Now().UTC().Format(time.RFC3339),
		},
		"entry": entries,
	}

	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal timeline bundle")
		return nil, fhirerrors.NewProcessingError("failed to build timeline response")
	}

	return &BundleResponse{Data: bundleJSON}, nil
}

// LabTrendsService provides the lab trends API.
type LabTrendsService struct {
	db    *sqlx.DB
	audit audit.AuditLogger
}

// NewLabTrendsService creates a new LabTrendsService.
func NewLabTrendsService(db *sqlx.DB, al audit.AuditLogger) *LabTrendsService {
	return &LabTrendsService{db: db, audit: al}
}

// LabTrendsParams holds query parameters for the lab trends endpoint.
type LabTrendsParams struct {
	PatientRef string
	Code       string
	Since      string
	Count      int
}

// GetLabTrends returns Observations for a LOINC code, sorted chronologically.
func (s *LabTrendsService) GetLabTrends(ctx context.Context, params LabTrendsParams, actorID uuid.UUID) (*BundleResponse, error) {
	if params.Count <= 0 {
		params.Count = 50
	}
	if params.Count > 200 {
		params.Count = 200
	}

	if params.PatientRef == "" || params.Code == "" {
		return nil, fhirerrors.NewValidationError("patient and code parameters are required")
	}

	s.audit.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "Observation",
		Action:       "lab-trends",
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   200,
	})

	// Verify patient exists
	parts := strings.SplitN(params.PatientRef, "/", 2)
	if len(parts) != 2 || parts[0] != "Patient" {
		return nil, fhirerrors.NewValidationError("patient parameter must be in format Patient/{id}")
	}
	patientID := parts[1]

	var exists bool
	err := s.db.GetContext(ctx, &exists, `
		SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND resource_type = 'Patient' AND deleted_at IS NULL)`,
		patientID,
	)
	if err != nil || !exists {
		return nil, fhirerrors.NewNotFoundError("Patient", patientID)
	}

	// Query observations
	var queryArgs []interface{}
	queryArgs = append(queryArgs, params.PatientRef, params.Code)
	argIdx := 3

	sinceFilter := ""
	if params.Since != "" {
		sinceFilter = fmt.Sprintf("AND (data->>'effectiveDateTime')::date >= $%d::date", argIdx)
		queryArgs = append(queryArgs, params.Since)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at
		FROM fhir_resources
		WHERE patient_ref = $1
			AND resource_type = 'Observation'
			AND deleted_at IS NULL
			AND data->'code'->'coding'->0->>'code' = $2
			%s
		ORDER BY (data->>'effectiveDateTime')::timestamptz ASC
		LIMIT $%d`,
		sinceFilter, argIdx,
	)
	queryArgs = append(queryArgs, params.Count)

	var resources []*repository.FHIRResource
	err = s.db.SelectContext(ctx, &resources, query, queryArgs...)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to query lab trends")
	}

	// Calculate trend direction from last 3 values
	trendDirection := calculateTrendDirection(resources)

	// Build bundle
	entries := make([]map[string]interface{}, 0, len(resources))
	for _, r := range resources {
		var data interface{}
		json.Unmarshal(r.Data, &data)
		entry := map[string]interface{}{
			"fullUrl":  fmt.Sprintf("%s/fhir/R4/Observation/%s", fhirBaseURL, r.ResourceID),
			"resource": data,
			"search": map[string]interface{}{
				"mode": "match",
				"extension": []map[string]interface{}{
					{
						"url":       "https://medilink.health/fhir/extensions/trend-direction",
						"valueCode": trendDirection,
					},
				},
			},
		}
		entries = append(entries, entry)
	}

	bundle := map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(resources),
		"id":           uuid.New().String(),
		"meta": map[string]interface{}{
			"lastUpdated": time.Now().UTC().Format(time.RFC3339),
		},
		"entry": entries,
	}

	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return nil, fhirerrors.NewProcessingError("failed to build lab trends response")
	}

	return &BundleResponse{Data: bundleJSON}, nil
}

// calculateTrendDirection compares the last 3 values to determine trend.
func calculateTrendDirection(resources []*repository.FHIRResource) string {
	if len(resources) < 2 {
		return "stable"
	}

	// Get the last 3 (or fewer) values
	start := len(resources) - 3
	if start < 0 {
		start = 0
	}
	recent := resources[start:]

	values := make([]float64, 0, len(recent))
	for _, r := range recent {
		var m map[string]interface{}
		if err := json.Unmarshal(r.Data, &m); err != nil {
			continue
		}
		if vq, ok := m["valueQuantity"].(map[string]interface{}); ok {
			if v, ok := vq["value"].(float64); ok {
				values = append(values, v)
			}
		}
	}

	if len(values) < 2 {
		return "stable"
	}

	increasing := 0
	decreasing := 0
	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1] {
			increasing++
		} else if values[i] < values[i-1] {
			decreasing++
		}
	}

	if increasing > decreasing {
		return "increasing"
	}
	if decreasing > increasing {
		return "decreasing"
	}
	return "stable"
}
