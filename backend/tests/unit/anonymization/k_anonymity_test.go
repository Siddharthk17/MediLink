package anonymization_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/anonymization"
	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/auth"
)

// Mocks

type mockExportRepo struct{}

func (m *mockExportRepo) Create(_ context.Context, _ *anonymization.ResearchExport) error {
	return nil
}
func (m *mockExportRepo) GetByID(_ context.Context, _ string) (*anonymization.ResearchExport, error) {
	return nil, nil
}
func (m *mockExportRepo) List(_ context.Context, _ string, _ bool, _, _ int) ([]*anonymization.ResearchExport, int, error) {
	return nil, 0, nil
}
func (m *mockExportRepo) UpdateStatus(_ context.Context, _, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockExportRepo) Delete(_ context.Context, _ string) error { return nil }

type mockStorage struct{}

func (m *mockStorage) UploadFile(_ context.Context, _ string, _ io.Reader, _ string, _ string, _ int64) (string, error) {
	return "", nil
}
func (m *mockStorage) UploadWithKey(_ context.Context, _, _ string, _ io.Reader, _ string, _ int64) error {
	return nil
}
func (m *mockStorage) GetPresignedURL(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockStorage) DeleteFile(_ context.Context, _, _ string) error { return nil }
func (m *mockStorage) Health(_ context.Context) bool                   { return true }

type mockAuditLogger struct{}

func (m *mockAuditLogger) Log(_ context.Context, _ audit.AuditEntry) error { return nil }
func (m *mockAuditLogger) LogAsync(_ audit.AuditEntry)                     {}
func (m *mockAuditLogger) Close()                                          {}

func init() { gin.SetMode(gin.TestMode) }

// newTestRouter wires a ResearchHandler with mocks and returns a Gin engine
// with POST /research/export pre-authenticated as a researcher.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:        "localhost:59999",
		DialTimeout: 100 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	handler := anonymization.NewResearchHandler(
		&mockExportRepo{},
		&mockStorage{},
		client,
		&mockAuditLogger{},
		zerolog.Nop(),
	)

	r := gin.New()
	r.POST("/research/export", func(c *gin.Context) {
		c.Set(auth.ActorIDKey, uuid.New())
		c.Set(auth.ActorRoleKey, "researcher")
		handler.RequestExport(c)
	})
	return r
}

// Age Band

func TestAgeBand_AllRanges(t *testing.T) {
	now := time.Now().Year()
	cases := []struct {
		birthYear string
		want      string
	}{
		{strconv.Itoa(now - 5), "0-17"},
		{strconv.Itoa(now - 25), "18-29"},
		{strconv.Itoa(now - 35), "30-39"},
		{strconv.Itoa(now - 45), "40-49"},
		{strconv.Itoa(now - 55), "50-59"},
		{strconv.Itoa(now - 65), "60-69"},
		{strconv.Itoa(now - 75), "70+"},
		{"invalid", "unknown"},
		{strconv.Itoa(now + 5), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.birthYear+"→"+tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, anonymization.AgeBand(tc.birthYear))
		})
	}
}

func TestAgeBand_InvalidInput(t *testing.T) {
	assert.Equal(t, "unknown", anonymization.AgeBand("abc"))
	assert.Equal(t, "unknown", anonymization.AgeBand(""))
}

// K-Anonymity suppression

func makeRecords(ageBand, gender, condition string, n int) []anonymization.AnonymizedRecord {
	recs := make([]anonymization.AnonymizedRecord, n)
	for i := range recs {
		recs[i] = anonymization.AnonymizedRecord{
			ResourceType:         "Observation",
			ResourceID:           ageBand + "-" + gender + "-" + strconv.Itoa(i),
			Data:                 map[string]interface{}{"status": "final"},
			AgeBand:              ageBand,
			Gender:               gender,
			PrimaryConditionCode: condition,
		}
	}
	return recs
}

func TestKAnonymity_SuppressesSmallGroups(t *testing.T) {
	records := append(
		makeRecords("30-39", "male", "E11", 2),
		makeRecords("30-39", "female", "E11", 5)...,
	)

	kept, stats := anonymization.ApplyKAnonymity(records, 5)

	assert.Len(t, kept, 5, "only the group with >= k members should remain")
	assert.Equal(t, 2, stats.SuppressedRecords)
	assert.Equal(t, 1, stats.SuppressedGroups)
}

func TestKAnonymity_KeepsLargeGroups(t *testing.T) {
	records := append(
		makeRecords("40-49", "male", "I10", 10),
		makeRecords("50-59", "female", "J45", 7)...,
	)

	kept, _ := anonymization.ApplyKAnonymity(records, 5)
	assert.Len(t, kept, 17, "all records from groups >= k should be kept")
}

func TestKAnonymity_StatsAccurate(t *testing.T) {
	records := append(
		makeRecords("18-29", "male", "J06", 6),
		append(
			makeRecords("30-39", "female", "E11", 3),
			makeRecords("40-49", "male", "I10", 8)...,
		)...,
	)

	_, stats := anonymization.ApplyKAnonymity(records, 5)

	assert.Equal(t, 17, stats.TotalRecords)
	assert.Equal(t, 14, stats.KeptRecords)
	assert.Equal(t, 3, stats.SuppressedRecords)
	assert.Equal(t, 1, stats.SuppressedGroups)
	assert.Equal(t, 6, stats.SmallestGroupKept)
	assert.Equal(t, 5, stats.KValue)
}

func TestKAnonymity_EmptyInput_ReturnsEmpty(t *testing.T) {
	kept, stats := anonymization.ApplyKAnonymity(nil, 5)
	assert.Empty(t, kept)
	assert.Equal(t, 0, stats.TotalRecords)
	assert.Equal(t, 0, stats.KeptRecords)
	assert.Equal(t, 0, stats.SuppressedRecords)
}

// Handler tests

func TestExportRequest_201(t *testing.T) {
	router := newTestRouter(t)

	body, _ := json.Marshal(map[string]interface{}{
		"resourceTypes": []string{"Patient", "Observation"},
		"purpose":       "Studying diabetes outcomes in a large urban population cohort",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/research/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "queued", resp["status"])
}

func TestExportRequest_EmptyResourceTypes_Returns400(t *testing.T) {
	router := newTestRouter(t)

	body, _ := json.Marshal(map[string]interface{}{
		"resourceTypes": []string{},
		"purpose":       "A sufficiently long purpose string for validation",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/research/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExportRequest_PurposeTooShort(t *testing.T) {
	router := newTestRouter(t)

	body, _ := json.Marshal(map[string]interface{}{
		"resourceTypes": []string{"Patient"},
		"purpose":       "Too short",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/research/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
