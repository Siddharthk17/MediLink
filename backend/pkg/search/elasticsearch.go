// Package search provides Elasticsearch integration for FHIR resource indexing and search.
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog"
)

// Index names — one per FHIR resource type.
const (
	IndexPatient            = "fhir-patients"
	IndexPractitioner       = "fhir-practitioners"
	IndexOrganization       = "fhir-organizations"
	IndexEncounter          = "fhir-encounters"
	IndexCondition          = "fhir-conditions"
	IndexMedicationRequest  = "fhir-medication-requests"
	IndexObservation        = "fhir-observations"
	IndexDiagnosticReport   = "fhir-diagnostic-reports"
	IndexAllergyIntolerance = "fhir-allergy-intolerances"
	IndexImmunization       = "fhir-immunizations"
)

// SearchResult holds a single search hit.
type SearchResult struct {
	ResourceType string
	ResourceID   string
	Score        float64
	Data         map[string]interface{}
	PatientRef   string
	Date         time.Time
}

// RawSearchHit represents a single ES hit with highlight information.
type RawSearchHit struct {
	Index     string
	ID        string
	Score     float64
	Source    map[string]interface{}
	Highlight map[string][]string
}

// MultiSearchResponse holds multi-index search results with total count.
type MultiSearchResponse struct {
	Total int
	Hits  []RawSearchHit
}

// SearchClient defines the interface for search operations.
type SearchClient interface {
	IndexResource(ctx context.Context, resourceType, resourceID string, data json.RawMessage) error
	DeleteResource(ctx context.Context, resourceType, resourceID string) error
	SearchByPatient(ctx context.Context, patientRef string, resourceTypes []string) ([]SearchResult, error)
	SearchGlobal(ctx context.Context, query string, patientRef string, count, offset int) ([]SearchResult, error)
	SearchMultiIndex(ctx context.Context, indices []string, query map[string]interface{}) (*MultiSearchResponse, error)
	SearchObservationsByCode(ctx context.Context, patientRef, loincCode string) ([]SearchResult, error)
	EnsureIndices(ctx context.Context) error
	Health(ctx context.Context) bool
}

// ESClient wraps the official go-elasticsearch client.
type ESClient struct {
	client  *elasticsearch.Client
	indices map[string]string
	logger  zerolog.Logger
}

// NewESClient creates a new Elasticsearch client.
func NewESClient(addresses []string, logger zerolog.Logger) (*ESClient, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	indices := map[string]string{
		"Patient":            IndexPatient,
		"Practitioner":       IndexPractitioner,
		"Organization":       IndexOrganization,
		"Encounter":          IndexEncounter,
		"Condition":          IndexCondition,
		"MedicationRequest":  IndexMedicationRequest,
		"Observation":        IndexObservation,
		"DiagnosticReport":   IndexDiagnosticReport,
		"AllergyIntolerance": IndexAllergyIntolerance,
		"Immunization":       IndexImmunization,
	}

	return &ESClient{
		client:  client,
		indices: indices,
		logger:  logger,
	}, nil
}

// indexName returns the ES index name for a resource type.
func (e *ESClient) indexName(resourceType string) string {
	if idx, ok := e.indices[resourceType]; ok {
		return idx
	}
	return "fhir-" + strings.ToLower(resourceType) + "s"
}

// IndexResource indexes a single FHIR resource.
func (e *ESClient) IndexResource(ctx context.Context, resourceType, resourceID string, data json.RawMessage) error {
	idx := e.indexName(resourceType)

	req := esapi.IndexRequest{
		Index:      idx,
		DocumentID: resourceID,
		Body:       bytes.NewReader(data),
		Refresh:    "false",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("elasticsearch index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch index error [%s]: %s", res.Status(), string(body))
	}

	return nil
}

// DeleteResource removes a resource from the index.
func (e *ESClient) DeleteResource(ctx context.Context, resourceType, resourceID string) error {
	idx := e.indexName(resourceType)

	req := esapi.DeleteRequest{
		Index:      idx,
		DocumentID: resourceID,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("elasticsearch delete request failed: %w", err)
	}
	defer res.Body.Close()

	// 404 is acceptable — resource may not have been indexed
	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch delete error [%s]: %s", res.Status(), string(body))
	}

	return nil
}

// SearchByPatient returns all indexed resources for a patient, sorted by date.
func (e *ESClient) SearchByPatient(ctx context.Context, patientRef string, resourceTypes []string) ([]SearchResult, error) {
	var indices []string
	if len(resourceTypes) > 0 {
		for _, rt := range resourceTypes {
			indices = append(indices, e.indexName(rt))
		}
	} else {
		for _, idx := range e.indices {
			indices = append(indices, idx)
		}
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"patientRef": patientRef,
			},
		},
		"sort": []map[string]interface{}{
			{"lastUpdated": map[string]string{"order": "desc"}},
		},
		"size": 200,
	}

	return e.executeSearch(ctx, indices, query)
}

// SearchGlobal searches across all resource type indices.
func (e *ESClient) SearchGlobal(ctx context.Context, queryStr string, patientRef string, count, offset int) ([]SearchResult, error) {
	var indices []string
	for _, idx := range e.indices {
		indices = append(indices, idx)
	}

	must := []map[string]interface{}{
		{"multi_match": map[string]interface{}{
			"query":  queryStr,
			"fields": []string{"*"},
		}},
	}

	if patientRef != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"patientRef": patientRef},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"from": offset,
		"size": count,
	}

	return e.executeSearch(ctx, indices, query)
}

// SearchMultiIndex executes a custom query against specific indices, returning
// total hit count and per-hit highlights.
func (e *ESClient) SearchMultiIndex(ctx context.Context, indices []string, query map[string]interface{}) (*MultiSearchResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(indices...),
		e.client.Search.WithBody(&buf),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch search error [%s]: %s", res.Status(), string(body))
	}

	var decoded struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Index     string                 `json:"_index"`
				ID        string                 `json:"_id"`
				Score     float64                `json:"_score"`
				Source    map[string]interface{} `json:"_source"`
				Highlight map[string][]string    `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	hits := make([]RawSearchHit, 0, len(decoded.Hits.Hits))
	for _, h := range decoded.Hits.Hits {
		hits = append(hits, RawSearchHit{
			Index:     h.Index,
			ID:        h.ID,
			Score:     h.Score,
			Source:    h.Source,
			Highlight: h.Highlight,
		})
	}

	return &MultiSearchResponse{
		Total: decoded.Hits.Total.Value,
		Hits:  hits,
	}, nil
}

// SearchObservationsByCode returns all observations with a specific LOINC code for a patient.
func (e *ESClient) SearchObservationsByCode(ctx context.Context, patientRef, loincCode string) ([]SearchResult, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]interface{}{"patientRef": patientRef}},
					{"term": map[string]interface{}{"code": loincCode}},
				},
			},
		},
		"sort": []map[string]interface{}{
			{"effectiveDate": map[string]string{"order": "asc"}},
		},
		"size": 200,
	}

	return e.executeSearch(ctx, []string{IndexObservation}, query)
}

func (e *ESClient) executeSearch(ctx context.Context, indices []string, query map[string]interface{}) ([]SearchResult, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(indices...),
		e.client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch search error [%s]: %s", res.Status(), string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Index  string                 `json:"_index"`
				ID     string                 `json:"_id"`
				Score  float64                `json:"_score"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	var results []SearchResult
	for _, hit := range result.Hits.Hits {
		sr := SearchResult{
			ResourceID: hit.ID,
			Score:      hit.Score,
			Data:       hit.Source,
		}
		if rt, ok := hit.Source["resourceType"].(string); ok {
			sr.ResourceType = rt
		}
		if pr, ok := hit.Source["patientRef"].(string); ok {
			sr.PatientRef = pr
		}
		results = append(results, sr)
	}

	return results, nil
}

// EnsureIndices creates all indices with correct mappings if they don't exist.
func (e *ESClient) EnsureIndices(ctx context.Context) error {
	for resourceType, indexName := range e.indices {
		mapping := e.getMappingForResource(resourceType)
		body, err := json.Marshal(mapping)
		if err != nil {
			return fmt.Errorf("failed to marshal mapping for %s: %w", resourceType, err)
		}

		res, err := e.client.Indices.Exists([]string{indexName})
		if err != nil {
			return fmt.Errorf("failed to check index existence for %s: %w", indexName, err)
		}
		res.Body.Close()

		if res.StatusCode == 200 {
			e.logger.Debug().Str("index", indexName).Msg("index already exists")
			continue
		}

		res, err = e.client.Indices.Create(
			indexName,
			e.client.Indices.Create.WithBody(bytes.NewReader(body)),
		)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
		res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("failed to create index %s: %s", indexName, res.Status())
		}

		e.logger.Info().Str("index", indexName).Str("resource", resourceType).Msg("created elasticsearch index")
	}

	return nil
}

// Health returns true if Elasticsearch is reachable.
func (e *ESClient) Health(ctx context.Context) bool {
	res, err := e.client.Ping()
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return !res.IsError()
}

func (e *ESClient) getMappingForResource(resourceType string) map[string]interface{} {
	common := map[string]interface{}{
		"resourceType": map[string]string{"type": "keyword"},
		"id":           map[string]string{"type": "keyword"},
		"patientRef":   map[string]string{"type": "keyword"},
		"status":       map[string]string{"type": "keyword"},
		"date":         map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis"},
		"lastUpdated":  map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis"},
	}

	extra := map[string]interface{}{}

	switch resourceType {
	case "Patient":
		extra["family"] = map[string]interface{}{"type": "text", "analyzer": "standard"}
		extra["given"] = map[string]interface{}{"type": "text", "analyzer": "standard"}
		extra["birthDate"] = map[string]interface{}{"type": "date", "format": "strict_date", "ignore_malformed": true}
		extra["gender"] = map[string]string{"type": "keyword"}
		extra["identifier"] = map[string]string{"type": "keyword"}
	case "Practitioner":
		extra["family"] = map[string]interface{}{"type": "text", "analyzer": "standard"}
		extra["given"] = map[string]interface{}{"type": "text", "analyzer": "standard"}
		extra["gender"] = map[string]string{"type": "keyword"}
		extra["identifier"] = map[string]string{"type": "keyword"}
	case "Organization":
		extra["name"] = map[string]interface{}{"type": "text", "analyzer": "standard"}
		extra["orgType"] = map[string]string{"type": "keyword"}
		extra["identifier"] = map[string]string{"type": "keyword"}
	case "Encounter":
		extra["class"] = map[string]string{"type": "keyword"}
		extra["type"] = map[string]interface{}{"type": "text"}
		extra["start"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
		extra["end"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
		extra["providerRef"] = map[string]string{"type": "keyword"}
	case "Condition":
		extra["code"] = map[string]string{"type": "keyword"}
		extra["codeDisplay"] = map[string]interface{}{"type": "text"}
		extra["clinicalStatus"] = map[string]string{"type": "keyword"}
		extra["onsetDate"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
		extra["encounterRef"] = map[string]string{"type": "keyword"}
	case "Observation":
		extra["code"] = map[string]string{"type": "keyword"}
		extra["codeDisplay"] = map[string]interface{}{"type": "text"}
		extra["valueQuantity"] = map[string]string{"type": "float"}
		extra["unit"] = map[string]string{"type": "keyword"}
		extra["effectiveDate"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
		extra["encounterRef"] = map[string]string{"type": "keyword"}
		extra["category"] = map[string]string{"type": "keyword"}
	case "MedicationRequest":
		extra["medicationCode"] = map[string]string{"type": "keyword"}
		extra["medicationDisplay"] = map[string]interface{}{"type": "text"}
		extra["authoredOn"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
		extra["requesterRef"] = map[string]string{"type": "keyword"}
	case "DiagnosticReport":
		extra["code"] = map[string]string{"type": "keyword"}
		extra["codeDisplay"] = map[string]interface{}{"type": "text"}
		extra["category"] = map[string]string{"type": "keyword"}
		extra["effectiveDate"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
	case "AllergyIntolerance":
		extra["code"] = map[string]string{"type": "keyword"}
		extra["codeDisplay"] = map[string]interface{}{"type": "text"}
		extra["severity"] = map[string]string{"type": "keyword"}
		extra["criticality"] = map[string]string{"type": "keyword"}
	case "Immunization":
		extra["vaccineCode"] = map[string]string{"type": "keyword"}
		extra["vaccineDisplay"] = map[string]interface{}{"type": "text"}
		extra["occurrenceDate"] = map[string]interface{}{"type": "date", "format": "strict_date_optional_time||epoch_millis", "ignore_malformed": true}
	}

	for k, v := range extra {
		common[k] = v
	}

	return map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": common,
		},
	}
}

// NoopSearchClient is a no-op implementation for testing without Elasticsearch.
type NoopSearchClient struct{}

// IndexResource is a no-op.
func (n *NoopSearchClient) IndexResource(_ context.Context, _, _ string, _ json.RawMessage) error {
	return nil
}

// DeleteResource is a no-op.
func (n *NoopSearchClient) DeleteResource(_ context.Context, _, _ string) error { return nil }

// SearchByPatient returns empty results.
func (n *NoopSearchClient) SearchByPatient(_ context.Context, _ string, _ []string) ([]SearchResult, error) {
	return nil, nil
}

// SearchGlobal returns empty results.
func (n *NoopSearchClient) SearchGlobal(_ context.Context, _ string, _ string, _, _ int) ([]SearchResult, error) {
	return nil, nil
}

// SearchMultiIndex returns empty results.
func (n *NoopSearchClient) SearchMultiIndex(_ context.Context, _ []string, _ map[string]interface{}) (*MultiSearchResponse, error) {
	return &MultiSearchResponse{}, nil
}

// SearchObservationsByCode returns empty results.
func (n *NoopSearchClient) SearchObservationsByCode(_ context.Context, _, _ string) ([]SearchResult, error) {
	return nil, nil
}

// EnsureIndices is a no-op.
func (n *NoopSearchClient) EnsureIndices(_ context.Context) error { return nil }

// Health always returns true.
func (n *NoopSearchClient) Health(_ context.Context) bool { return true }
