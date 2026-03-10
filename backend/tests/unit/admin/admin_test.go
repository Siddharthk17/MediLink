package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/admin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock crypto.Encryptor

type mockEncryptor struct{}

func (m *mockEncryptor) Encrypt(plaintext []byte) ([]byte, error)  { return plaintext, nil }
func (m *mockEncryptor) Decrypt(ciphertext []byte) ([]byte, error) { return ciphertext, nil }
func (m *mockEncryptor) EncryptString(s string) ([]byte, error)    { return []byte(s), nil }
func (m *mockEncryptor) DecryptString(b []byte) (string, error)    { return string(b), nil }

// Mock admin.Service

type mockService struct {
	listUsersFn           func(ctx context.Context, f admin.UserFilters, count, offset int) ([]admin.UserSummary, int, error)
	getUserFn             func(ctx context.Context, id string) (*admin.UserDetail, error)
	updateUserRoleFn      func(ctx context.Context, id, role string) error
	inviteResearcherFn    func(ctx context.Context, email, by string) error
	suspendPhysicianFn    func(ctx context.Context, id string) error
	reinstatePhysicianFn  func(ctx context.Context, id string) error
	getAuditLogsFn        func(ctx context.Context, f admin.AuditLogFilters, count, offset int) ([]admin.AuditLogEntry, int, error)
	getPatientAuditLogFn  func(ctx context.Context, ref string, count, offset int) ([]admin.AuditLogEntry, int, error)
	getActorAuditLogFn    func(ctx context.Context, id string, count, offset int) ([]admin.AuditLogEntry, int, error)
	getBreakGlassEventsFn func(ctx context.Context, count, offset int) ([]admin.AuditLogEntry, int, error)
	getStatsFn            func(ctx context.Context) (*admin.AdminStats, error)
	pingFn                func(ctx context.Context) error
}

func (m *mockService) ListUsers(ctx context.Context, f admin.UserFilters, count, offset int) ([]admin.UserSummary, int, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, f, count, offset)
	}
	return nil, 0, nil
}
func (m *mockService) GetUser(ctx context.Context, id string) (*admin.UserDetail, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, id)
	}
	return nil, nil
}
func (m *mockService) UpdateUserRole(ctx context.Context, id, role string) error {
	if m.updateUserRoleFn != nil {
		return m.updateUserRoleFn(ctx, id, role)
	}
	return nil
}
func (m *mockService) InviteResearcher(ctx context.Context, email, by string) error {
	if m.inviteResearcherFn != nil {
		return m.inviteResearcherFn(ctx, email, by)
	}
	return nil
}
func (m *mockService) SuspendPhysician(ctx context.Context, id string) error {
	if m.suspendPhysicianFn != nil {
		return m.suspendPhysicianFn(ctx, id)
	}
	return nil
}
func (m *mockService) ReinstatePhysician(ctx context.Context, id string) error {
	if m.reinstatePhysicianFn != nil {
		return m.reinstatePhysicianFn(ctx, id)
	}
	return nil
}
func (m *mockService) GetAuditLogs(ctx context.Context, f admin.AuditLogFilters, count, offset int) ([]admin.AuditLogEntry, int, error) {
	if m.getAuditLogsFn != nil {
		return m.getAuditLogsFn(ctx, f, count, offset)
	}
	return nil, 0, nil
}
func (m *mockService) GetPatientAuditLog(ctx context.Context, ref string, count, offset int) ([]admin.AuditLogEntry, int, error) {
	if m.getPatientAuditLogFn != nil {
		return m.getPatientAuditLogFn(ctx, ref, count, offset)
	}
	return nil, 0, nil
}
func (m *mockService) GetActorAuditLog(ctx context.Context, id string, count, offset int) ([]admin.AuditLogEntry, int, error) {
	if m.getActorAuditLogFn != nil {
		return m.getActorAuditLogFn(ctx, id, count, offset)
	}
	return nil, 0, nil
}
func (m *mockService) GetBreakGlassEvents(ctx context.Context, count, offset int) ([]admin.AuditLogEntry, int, error) {
	if m.getBreakGlassEventsFn != nil {
		return m.getBreakGlassEventsFn(ctx, count, offset)
	}
	return nil, 0, nil
}
func (m *mockService) GetStats(ctx context.Context) (*admin.AdminStats, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(ctx)
	}
	return nil, nil
}
func (m *mockService) Ping(ctx context.Context) error {
	if m.pingFn != nil {
		return m.pingFn(ctx)
	}
	return nil
}

// Helpers

func newHandler(svc admin.Service) *admin.AdminHandler {
	logger := zerolog.Nop()
	return admin.NewAdminHandler(svc, &mockEncryptor{}, nil, logger)
}

// setupAdminRouter wires the same routes as production (cmd/api/main.go).
func setupAdminRouter(svc admin.Service) *gin.Engine {
	h := newHandler(svc)
	r := gin.New()
	g := r.Group("/admin")

	// AdminHandler-managed endpoints
	g.GET("/users", h.ListUsers)
	g.GET("/users/:userId", h.GetUser)
	g.PUT("/users/:userId/role", h.UpdateUserRole)
	g.POST("/researchers/invite", h.InviteResearcher)
	g.GET("/audit-logs", h.GetAuditLogs)
	g.GET("/audit-logs/patient/:id", h.GetPatientAuditLog)
	g.GET("/audit-logs/actor/:id", h.GetActorAuditLog)
	g.GET("/audit-logs/break-glass", h.GetBreakGlassEvents)
	g.GET("/stats", h.GetStats)

	// Physician management endpoints (inline in production; delegate to service here)
	g.POST("/physicians/:userId/approve", func(c *gin.Context) {
		userID := c.Param("userId")
		if err := svc.ReinstatePhysician(c.Request.Context(), userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Physician approved"})
	})
	g.POST("/physicians/:userId/suspend", func(c *gin.Context) {
		userID := c.Param("userId")
		if err := svc.SuspendPhysician(c.Request.Context(), userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Physician suspended"})
	})

	return r
}

func parseJSON(t *testing.T, body *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body.Bytes(), &m))
	return m
}

func sampleUsers() []admin.UserSummary {
	now := time.Now().UTC()
	return []admin.UserSummary{
		{ID: "u-1", Role: "patient", Status: "active", FullNameEnc: []byte("Alice"), EmailHash: "hash1", CreatedAt: now},
		{ID: "u-2", Role: "physician", Status: "active", FullNameEnc: []byte("Bob"), EmailHash: "hash2", CreatedAt: now},
	}
}

func sampleAuditEntries() []admin.AuditLogEntry {
	actor := "actor-1"
	now := time.Now().UTC()
	return []admin.AuditLogEntry{
		{ID: "a-1", UserID: &actor, UserRole: "physician", ResourceType: "Patient", ResourceID: "p-1", Action: "read", Purpose: "treatment", Success: true, StatusCode: 200, IPAddress: "10.0.0.1", CreatedAt: now},
		{ID: "a-2", UserID: &actor, UserRole: "physician", ResourceType: "Patient", ResourceID: "p-2", Action: "break_glass", Purpose: "emergency", Success: true, StatusCode: 200, IPAddress: "10.0.0.1", CreatedAt: now},
	}
}

func sampleStats() *admin.AdminStats {
	s := &admin.AdminStats{}
	s.Users.Total = 150
	s.Users.Patients = 100
	s.Users.Physicians = 30
	s.Users.Admins = 5
	s.Users.Researchers = 10
	s.Users.Pending = 5
	s.FHIRResources.Total = 5000
	s.FHIRResources.ByType = map[string]int{"Patient": 100, "Observation": 3000, "MedicationRequest": 1900}
	s.Documents.TotalUploaded = 200
	s.Documents.Completed = 180
	s.Documents.Failed = 5
	s.Documents.Pending = 15
	s.DrugChecks.Total = 80
	s.DrugChecks.Contraindicated = 2
	s.DrugChecks.Major = 10
	s.DrugChecks.Moderate = 30
	s.Consents.ActiveGrants = 90
	s.Consents.RevokedThisMonth = 3
	s.Consents.BreakGlassThisMonth = 1
	s.Exports.Total = 25
	s.Exports.Completed = 20
	s.Exports.Pending = 5
	return s
}

// Tests

func TestListUsers_200_Paginated(t *testing.T) {
	users := sampleUsers()
	svc := &mockService{
		listUsersFn: func(_ context.Context, _ admin.UserFilters, count, offset int) ([]admin.UserSummary, int, error) {
			assert.Equal(t, 20, count, "default page size should be 20")
			assert.Equal(t, 0, offset)
			return users, 50, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w.Body)

	assert.EqualValues(t, 50, body["total"])
	assert.EqualValues(t, 20, body["count"])
	assert.EqualValues(t, 0, body["offset"])

	arr, ok := body["users"].([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 2)

	first := arr[0].(map[string]interface{})
	assert.Equal(t, "Alice", first["fullName"])
}

func TestListUsers_FilterByRole(t *testing.T) {
	var captured admin.UserFilters
	svc := &mockService{
		listUsersFn: func(_ context.Context, f admin.UserFilters, _, _ int) ([]admin.UserSummary, int, error) {
			captured = f
			return sampleUsers()[:1], 1, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?role=physician", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "physician", captured.Role)
}

func TestListUsers_FilterByStatus(t *testing.T) {
	var captured admin.UserFilters
	svc := &mockService{
		listUsersFn: func(_ context.Context, f admin.UserFilters, _, _ int) ([]admin.UserSummary, int, error) {
			captured = f
			return sampleUsers()[:1], 1, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?status=suspended", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "suspended", captured.Status)
}

func TestListUsers_MaxPageCapped(t *testing.T) {
	svc := &mockService{
		listUsersFn: func(_ context.Context, _ admin.UserFilters, count, _ int) ([]admin.UserSummary, int, error) {
			assert.Equal(t, 100, count, "_count=500 must be capped to 100")
			return nil, 0, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?_count=500", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w.Body)
	assert.EqualValues(t, 100, body["count"])
}

func TestGetAuditLogs_200(t *testing.T) {
	entries := sampleAuditEntries()
	svc := &mockService{
		getAuditLogsFn: func(_ context.Context, _ admin.AuditLogFilters, count, offset int) ([]admin.AuditLogEntry, int, error) {
			assert.Equal(t, 50, count, "default audit log page size should be 50")
			assert.Equal(t, 0, offset)
			return entries, 2, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseJSON(t, w.Body)

	assert.EqualValues(t, 2, body["total"])
	assert.EqualValues(t, 50, body["count"])

	arr, ok := body["entries"].([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 2)
}

func TestGetAuditLogs_FilterByActor(t *testing.T) {
	var captured admin.AuditLogFilters
	svc := &mockService{
		getAuditLogsFn: func(_ context.Context, f admin.AuditLogFilters, _, _ int) ([]admin.AuditLogEntry, int, error) {
			captured = f
			return sampleAuditEntries()[:1], 1, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs?actor_id=user-42", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-42", captured.ActorID)
}

func TestGetAuditLogs_BreakGlassFilter(t *testing.T) {
	var captured admin.AuditLogFilters
	svc := &mockService{
		getAuditLogsFn: func(_ context.Context, f admin.AuditLogFilters, _, _ int) ([]admin.AuditLogEntry, int, error) {
			captured = f
			return sampleAuditEntries()[1:], 1, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit-logs?break_glass_only=true", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, captured.BreakGlassOnly, "BreakGlassOnly filter should be true")
}

func TestGetAuditLogs_NoDeleteEndpoint(t *testing.T) {
	r := setupAdminRouter(&mockService{})
	r.HandleMethodNotAllowed = true
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/audit-logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestGetStats_200_AllFields(t *testing.T) {
	expected := sampleStats()
	svc := &mockService{
		getStatsFn: func(_ context.Context) (*admin.AdminStats, error) {
			return expected, nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got admin.AdminStats
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	// Users
	assert.Equal(t, expected.Users.Total, got.Users.Total)
	assert.Equal(t, expected.Users.Patients, got.Users.Patients)
	assert.Equal(t, expected.Users.Physicians, got.Users.Physicians)
	assert.Equal(t, expected.Users.Admins, got.Users.Admins)
	assert.Equal(t, expected.Users.Researchers, got.Users.Researchers)
	assert.Equal(t, expected.Users.Pending, got.Users.Pending)

	// FHIR resources
	assert.Equal(t, expected.FHIRResources.Total, got.FHIRResources.Total)
	assert.Equal(t, expected.FHIRResources.ByType["Patient"], got.FHIRResources.ByType["Patient"])
	assert.Equal(t, expected.FHIRResources.ByType["Observation"], got.FHIRResources.ByType["Observation"])
	assert.Equal(t, expected.FHIRResources.ByType["MedicationRequest"], got.FHIRResources.ByType["MedicationRequest"])

	// Documents
	assert.Equal(t, expected.Documents.TotalUploaded, got.Documents.TotalUploaded)
	assert.Equal(t, expected.Documents.Completed, got.Documents.Completed)
	assert.Equal(t, expected.Documents.Failed, got.Documents.Failed)
	assert.Equal(t, expected.Documents.Pending, got.Documents.Pending)

	// Drug checks
	assert.Equal(t, expected.DrugChecks.Total, got.DrugChecks.Total)
	assert.Equal(t, expected.DrugChecks.Contraindicated, got.DrugChecks.Contraindicated)
	assert.Equal(t, expected.DrugChecks.Major, got.DrugChecks.Major)
	assert.Equal(t, expected.DrugChecks.Moderate, got.DrugChecks.Moderate)

	// Consents
	assert.Equal(t, expected.Consents.ActiveGrants, got.Consents.ActiveGrants)
	assert.Equal(t, expected.Consents.RevokedThisMonth, got.Consents.RevokedThisMonth)
	assert.Equal(t, expected.Consents.BreakGlassThisMonth, got.Consents.BreakGlassThisMonth)

	// Exports
	assert.Equal(t, expected.Exports.Total, got.Exports.Total)
	assert.Equal(t, expected.Exports.Completed, got.Exports.Completed)
	assert.Equal(t, expected.Exports.Pending, got.Exports.Pending)
}

func TestApprovePhysician_200(t *testing.T) {
	called := false
	svc := &mockService{
		reinstatePhysicianFn: func(_ context.Context, id string) error {
			assert.Equal(t, "phys-1", id)
			called = true
			return nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/physicians/phys-1/approve", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	body := parseJSON(t, w.Body)
	assert.Equal(t, "Physician approved", body["message"])
}

func TestSuspendPhysician_200(t *testing.T) {
	called := false
	svc := &mockService{
		suspendPhysicianFn: func(_ context.Context, id string) error {
			assert.Equal(t, "phys-2", id)
			called = true
			return nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/physicians/phys-2/suspend", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	body := parseJSON(t, w.Body)
	assert.Equal(t, "Physician suspended", body["message"])
}

func TestInviteResearcher_201(t *testing.T) {
	called := false
	svc := &mockService{
		inviteResearcherFn: func(_ context.Context, email, _ string) error {
			assert.Equal(t, "new@university.edu", email)
			called = true
			return nil
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	payload := `{"email":"new@university.edu"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/researchers/invite", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, called)
	body := parseJSON(t, w.Body)
	assert.Equal(t, "researcher invitation created", body["message"])
}

func TestInviteResearcher_DuplicateEmail(t *testing.T) {
	svc := &mockService{
		inviteResearcherFn: func(_ context.Context, _ string, _ string) error {
			return &admin.DuplicateError{Message: "email already registered"}
		},
	}

	r := setupAdminRouter(svc)
	w := httptest.NewRecorder()
	payload := `{"email":"existing@university.edu"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/researchers/invite", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}
