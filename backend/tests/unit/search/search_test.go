package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalsearch "github.com/Siddharthk17/MediLink/internal/search"
	pkgsearch "github.com/Siddharthk17/MediLink/pkg/search"
)

// Mock SearchClient

type mockSearchClient struct {
	searchMultiIndexFn func(ctx context.Context, indices []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error)
}

func (m *mockSearchClient) IndexResource(_ context.Context, _, _ string, _ json.RawMessage) error {
	return nil
}
func (m *mockSearchClient) DeleteResource(_ context.Context, _, _ string) error { return nil }
func (m *mockSearchClient) SearchByPatient(_ context.Context, _ string, _ []string) ([]pkgsearch.SearchResult, error) {
	return nil, nil
}
func (m *mockSearchClient) SearchGlobal(_ context.Context, _ string, _ string, _, _ int) ([]pkgsearch.SearchResult, error) {
	return nil, nil
}
func (m *mockSearchClient) SearchMultiIndex(ctx context.Context, indices []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
	if m.searchMultiIndexFn != nil {
		return m.searchMultiIndexFn(ctx, indices, query)
	}
	return &pkgsearch.MultiSearchResponse{}, nil
}
func (m *mockSearchClient) SearchObservationsByCode(_ context.Context, _, _ string) ([]pkgsearch.SearchResult, error) {
	return nil, nil
}
func (m *mockSearchClient) EnsureIndices(_ context.Context) error { return nil }
func (m *mockSearchClient) Health(_ context.Context) bool         { return true }

// Helpers

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(mock *mockSearchClient, db *sqlx.DB) *internalsearch.SearchHandler {
	logger := zerolog.Nop()
	svc := internalsearch.NewSearchService(mock, db, logger)
	return internalsearch.NewSearchHandler(svc, db, logger)
}

func defaultESResponse() *pkgsearch.MultiSearchResponse {
	return &pkgsearch.MultiSearchResponse{
		Total: 2,
		Hits: []pkgsearch.RawSearchHit{
			{
				Index: "fhir-conditions",
				ID:    "cond-1",
				Score: 4.5,
				Source: map[string]interface{}{
					"resourceType": "Condition",
					"id":           "cond-1",
					"codeDisplay":  "Hypertension",
					"patientRef":   "Patient/p-123",
				},
			},
			{
				Index: "fhir-observations",
				ID:    "obs-1",
				Score: 3.2,
				Source: map[string]interface{}{
					"resourceType": "Observation",
					"id":           "obs-1",
					"codeDisplay":  "Blood pressure",
					"patientRef":   "Patient/p-123",
				},
			},
		},
	}
}

func performRequest(handler *internalsearch.SearchHandler, url string, actorID, actorRole string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.GET("/search", func(c *gin.Context) {
		c.Set("actor_id", actorID)
		c.Set("actor_role", actorRole)
		handler.UnifiedSearch(c)
	})

	req := httptest.NewRequest(http.MethodGet, url, nil)
	engine.ServeHTTP(w, req)
	return w
}

func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// Handler tests

func TestUnifiedSearch_200_AllTypes(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Admin role needs no DB lookup.
	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, _ map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			return defaultESResponse(), nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=hypertension", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	body := parseBody(t, w)
	assert.Equal(t, "Bundle", body["resourceType"])
	assert.Equal(t, "searchset", body["type"])
	assert.Equal(t, float64(2), body["total"])

	entries, ok := body["entry"].([]interface{})
	require.True(t, ok)
	assert.Len(t, entries, 2)
}

func TestUnifiedSearch_PatientScopedToOwn(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	patientActorID := "00000000-0000-0000-0000-000000000001"

	// Handler resolves FHIR patient ID from DB.
	rows := sqlmock.NewRows([]string{"coalesce"}).AddRow("p-123")
	dbMock.ExpectQuery("SELECT COALESCE").
		WithArgs(patientActorID).
		WillReturnRows(rows)

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var capturedQuery map[string]interface{}
	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			capturedQuery = query
			return &pkgsearch.MultiSearchResponse{
				Total: 1,
				Hits: []pkgsearch.RawSearchHit{
					{
						Index:  "fhir-conditions",
						ID:     "cond-1",
						Score:  4.0,
						Source: map[string]interface{}{"resourceType": "Condition", "patientRef": "Patient/p-123"},
					},
				},
			}, nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=diabetes", patientActorID, "patient")

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the ES query filters by the patient's own FHIR ID.
	boolQ := capturedQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filters := boolQ["filter"].([]interface{})
	require.NotEmpty(t, filters)

	termFilter := filters[0].(map[string]interface{})
	term := termFilter["term"].(map[string]interface{})
	assert.Equal(t, "Patient/p-123", term["patientRef"])
}

func TestUnifiedSearch_PhysicianRequiresPatient(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	_ = dbMock // no DB expectations needed — error before any DB call for consent

	mock := &mockSearchClient{}
	handler := newTestHandler(mock, sqlxDB)

	physicianID := "00000000-0000-0000-0000-000000000002"
	w := performRequest(handler, "/search?q=test", physicianID, "physician")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	body := parseBody(t, w)
	assert.Equal(t, "OperationOutcome", body["resourceType"])

	issues := body["issue"].([]interface{})
	diag := issues[0].(map[string]interface{})["diagnostics"].(string)
	assert.Contains(t, diag, "patient parameter required")
}

func TestUnifiedSearch_AdminUnrestricted(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var capturedQuery map[string]interface{}
	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			capturedQuery = query
			return defaultESResponse(), nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=test", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	// Admin with no ?patient= should have no patientRef filter.
	boolQ := capturedQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filters := boolQ["filter"].([]interface{})
	assert.Empty(t, filters, "admin without patient param should have no patientRef filter")
}

func TestUnifiedSearch_FuzzyMatch(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			// Verify the query uses fuzzy matching.
			boolQ := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
			musts := boolQ["must"].([]interface{})
			mm := musts[0].(map[string]interface{})["multi_match"].(map[string]interface{})
			assert.Equal(t, "AUTO", mm["fuzziness"])
			assert.Equal(t, "sharrma", mm["query"])

			return &pkgsearch.MultiSearchResponse{
				Total: 1,
				Hits: []pkgsearch.RawSearchHit{
					{
						Index: "fhir-patients",
						ID:    "pat-1",
						Score: 3.0,
						Source: map[string]interface{}{
							"resourceType": "Patient",
							"family":       "Sharma",
						},
						Highlight: map[string][]string{
							"family": {"<em>Sharma</em>"},
						},
					},
				},
			}, nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=sharrma", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	body := parseBody(t, w)
	entries := body["entry"].([]interface{})
	require.Len(t, entries, 1)

	entry := entries[0].(map[string]interface{})
	resource := entry["resource"].(map[string]interface{})
	assert.Equal(t, "Sharma", resource["family"])
}

func TestUnifiedSearch_TypeFilter(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var capturedIndices []string
	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, indices []string, _ map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			capturedIndices = indices
			return &pkgsearch.MultiSearchResponse{
				Total: 1,
				Hits: []pkgsearch.RawSearchHit{
					{
						Index:  "fhir-observations",
						ID:     "obs-1",
						Score:  5.0,
						Source: map[string]interface{}{"resourceType": "Observation"},
					},
				},
			}, nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=bp&type=Observation", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, capturedIndices, 1)
	assert.Equal(t, "fhir-observations", capturedIndices[0])
}

func TestUnifiedSearch_DateFilter(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var capturedQuery map[string]interface{}
	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			capturedQuery = query
			return &pkgsearch.MultiSearchResponse{Total: 0, Hits: nil}, nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=labs&date=ge2024-01-01&date=le2024-12-31", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	boolQ := capturedQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filters := boolQ["filter"].([]interface{})
	require.NotEmpty(t, filters)

	// Find the range filter.
	var rangeFilter map[string]interface{}
	for _, f := range filters {
		fm := f.(map[string]interface{})
		if _, ok := fm["range"]; ok {
			rangeFilter = fm["range"].(map[string]interface{})
			break
		}
	}
	require.NotNil(t, rangeFilter, "expected a range filter in the query")

	dateRange := rangeFilter["date"].(map[string]interface{})
	gteStr, ok := dateRange["gte"].(string)
	require.True(t, ok)
	gte, err := time.Parse(time.RFC3339, gteStr)
	require.NoError(t, err)
	assert.Equal(t, 2024, gte.Year())
	assert.Equal(t, time.January, gte.Month())
	assert.Equal(t, 1, gte.Day())

	lteStr, ok := dateRange["lte"].(string)
	require.True(t, ok)
	lte, err := time.Parse(time.RFC3339, lteStr)
	require.NoError(t, err)
	assert.Equal(t, 2024, lte.Year())
	assert.Equal(t, time.December, lte.Month())
	assert.Equal(t, 31, lte.Day())
}

func TestUnifiedSearch_Pagination(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	var capturedQuery map[string]interface{}
	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, query map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			capturedQuery = query
			return &pkgsearch.MultiSearchResponse{Total: 0, Hits: nil}, nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=test&_count=10&_offset=20", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	// The ES query should have size=10 and from=20.
	assert.Equal(t, 10, capturedQuery["size"])
	assert.Equal(t, 20, capturedQuery["from"])
}

func TestSearch_QueriesLogged(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	dbMock.ExpectExec("INSERT INTO search_queries").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock := &mockSearchClient{
		searchMultiIndexFn: func(_ context.Context, _ []string, _ map[string]interface{}) (*pkgsearch.MultiSearchResponse, error) {
			return defaultESResponse(), nil
		},
	}

	handler := newTestHandler(mock, sqlxDB)
	w := performRequest(handler, "/search?q=diabetes", "admin-uuid", "admin")

	assert.Equal(t, http.StatusOK, w.Code)

	// The search handler logs queries asynchronously via goroutine.
	// Give it a moment to complete.
	time.Sleep(200 * time.Millisecond)

	// Verify the INSERT expectation was met (query was logged).
	err = dbMock.ExpectationsWereMet()
	assert.NoError(t, err, "search query should be logged to search_queries table")
}

// QueryBuilder tests

func TestQueryBuilder_NoType_AllIndices(t *testing.T) {
	indices := internalsearch.IndicesForTypes(nil)
	assert.Len(t, indices, 10, "no type filter should return all 10 FHIR indices")
}

func TestQueryBuilder_SpecificType(t *testing.T) {
	indices := internalsearch.IndicesForTypes([]string{"Observation"})
	require.Len(t, indices, 1)
	assert.Equal(t, "fhir-observations", indices[0])
}

func TestQueryBuilder_PatientFilter(t *testing.T) {
	filters := internalsearch.SearchFilters{
		ActorRole:     "patient",
		PatientFHIRID: "p-abc",
	}
	query := internalsearch.BuildQuery("test", 20, 0, filters)

	boolQ := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filterSlice := boolQ["filter"].([]interface{})
	require.NotEmpty(t, filterSlice)

	termFilter := filterSlice[0].(map[string]interface{})
	term := termFilter["term"].(map[string]interface{})
	assert.Equal(t, "Patient/p-abc", term["patientRef"])
}

// ResultBuilder tests

func TestResultBuilder_ValidFHIRBundle(t *testing.T) {
	hits := []internalsearch.SearchHit{
		{
			ResourceType: "Condition",
			ResourceID:   "cond-1",
			Score:        4.5,
			Source:       map[string]interface{}{"resourceType": "Condition", "id": "cond-1"},
		},
		{
			ResourceType: "Observation",
			ResourceID:   "obs-1",
			Score:        3.2,
			Source:       map[string]interface{}{"resourceType": "Observation", "id": "obs-1"},
		},
	}

	bundle := internalsearch.BuildSearchBundle(hits, 2, "test query")

	assert.Equal(t, "Bundle", bundle["resourceType"])
	assert.Equal(t, "searchset", bundle["type"])
	assert.Equal(t, 2, bundle["total"])

	entries, ok := bundle["entry"].([]interface{})
	require.True(t, ok)
	require.Len(t, entries, 2)

	first := entries[0].(map[string]interface{})
	assert.Equal(t, "Condition/cond-1", first["fullUrl"])

	resource := first["resource"].(map[string]interface{})
	assert.Equal(t, "Condition", resource["resourceType"])
}

func TestResultBuilder_ScoreInSearchExtension(t *testing.T) {
	hits := []internalsearch.SearchHit{
		{
			ResourceType: "Observation",
			ResourceID:   "obs-1",
			Score:        7.89,
			Source:       map[string]interface{}{"resourceType": "Observation"},
		},
	}

	bundle := internalsearch.BuildSearchBundle(hits, 1, "bp")

	entries := bundle["entry"].([]interface{})
	require.Len(t, entries, 1)

	entry := entries[0].(map[string]interface{})
	searchField := entry["search"].(map[string]interface{})
	assert.Equal(t, "match", searchField["mode"])
	assert.Equal(t, 7.89, searchField["score"])
}
