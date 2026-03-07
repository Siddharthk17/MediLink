package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/medilink/internal/auth"
	"github.com/Siddharthk17/medilink/internal/fhir/repository"
	"github.com/Siddharthk17/medilink/internal/fhir/services"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// mockPatientService implements services.PatientService for testing.
type mockPatientService struct {
	createFn  func(ctx context.Context, req services.CreatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error)
	getFn     func(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error)
	updateFn  func(ctx context.Context, resourceID string, req services.UpdatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error)
	deleteFn  func(ctx context.Context, resourceID string, actorID uuid.UUID) error
	searchFn  func(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error)
	historyFn func(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*services.BundleResponse, error)
	versionFn func(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*services.PatientResponse, error)
}

func (m *mockPatientService) CreatePatient(ctx context.Context, req services.CreatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.createFn(ctx, req, actorID)
}

func (m *mockPatientService) GetPatient(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.getFn(ctx, resourceID, actorID)
}

func (m *mockPatientService) GetPatientVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.versionFn(ctx, resourceID, versionID, actorID)
}

func (m *mockPatientService) GetPatientHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*services.BundleResponse, error) {
	return m.historyFn(ctx, resourceID, actorID, count, offset)
}

func (m *mockPatientService) UpdatePatient(ctx context.Context, resourceID string, req services.UpdatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.updateFn(ctx, resourceID, req, actorID)
}

func (m *mockPatientService) DeletePatient(ctx context.Context, resourceID string, actorID uuid.UUID) error {
	return m.deleteFn(ctx, resourceID, actorID)
}

func (m *mockPatientService) SearchPatients(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error) {
	return m.searchFn(ctx, params, actorID, baseURL)
}

func setupRouter(handler *PatientHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(auth.AuthMiddleware())

	patients := router.Group("/fhir/R4/Patient")
	{
		patients.POST("", handler.CreatePatient)
		patients.GET("", handler.SearchPatients)
		patients.GET("/:id", handler.GetPatient)
		patients.PUT("/:id", handler.UpdatePatient)
		patients.DELETE("/:id", handler.DeletePatient)
		patients.GET("/:id/_history", handler.GetPatientHistory)
		patients.GET("/:id/_history/:vid", handler.GetPatientVersion)
	}
	return router
}

func TestCreatePatient_201(t *testing.T) {
	patientData := `{"resourceType":"Patient","id":"test-id","name":[{"family":"Sharma"}]}`

	mock := &mockPatientService{
		createFn: func(ctx context.Context, req services.CreatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(patientData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	body := bytes.NewBufferString(`{"resourceType":"Patient","name":[{"family":"Sharma"}]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/fhir/R4/Patient", body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/fhir/R4/Patient/test-id")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/fhir+json")
}

func TestCreatePatient_400_NoName(t *testing.T) {
	mock := &mockPatientService{
		createFn: func(ctx context.Context, req services.CreatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error) {
			return nil, fhirerrors.NewValidationError("Patient.name: at least one name is required", "Patient.name")
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	body := bytes.NewBufferString(`{"resourceType":"Patient","gender":"male"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/fhir/R4/Patient", body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var outcome fhirerrors.OperationOutcome
	err := json.Unmarshal(w.Body.Bytes(), &outcome)
	require.NoError(t, err)
	assert.Equal(t, "OperationOutcome", outcome.ResourceType)
}

func TestGetPatient_200(t *testing.T) {
	patientData := `{"resourceType":"Patient","id":"123","name":[{"family":"Test"}]}`

	mock := &mockPatientService{
		getFn: func(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(patientData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Patient")
}

func TestGetPatient_404(t *testing.T) {
	mock := &mockPatientService{
		getFn: func(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error) {
			return nil, fhirerrors.NewNotFoundError("Patient", resourceID)
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var outcome fhirerrors.OperationOutcome
	err := json.Unmarshal(w.Body.Bytes(), &outcome)
	require.NoError(t, err)
	assert.Equal(t, "OperationOutcome", outcome.ResourceType)
	assert.Equal(t, "not-found", outcome.Issue[0].Code)
}

func TestUpdatePatient_200(t *testing.T) {
	updatedData := `{"resourceType":"Patient","id":"123","meta":{"versionId":"2"},"name":[{"family":"Updated"}]}`

	mock := &mockPatientService{
		updateFn: func(ctx context.Context, resourceID string, req services.UpdatePatientRequest, actorID uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(updatedData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	body := bytes.NewBufferString(`{"resourceType":"Patient","name":[{"family":"Updated"}]}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/fhir/R4/Patient/123", body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	meta := result["meta"].(map[string]interface{})
	assert.Equal(t, "2", meta["versionId"])
}

func TestDeletePatient_204(t *testing.T) {
	mock := &mockPatientService{
		deleteFn: func(ctx context.Context, resourceID string, actorID uuid.UUID) error {
			return nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/fhir/R4/Patient/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSearchPatients_200(t *testing.T) {
	bundleData := `{"resourceType":"Bundle","type":"searchset","total":1,"entry":[{"resource":{"resourceType":"Patient"}}]}`

	mock := &mockPatientService{
		searchFn: func(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error) {
			return &services.BundleResponse{Data: json.RawMessage(bundleData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient?family=Sharma", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var bundle map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &bundle)
	assert.Equal(t, "Bundle", bundle["resourceType"])
	assert.Equal(t, "searchset", bundle["type"])
}

func TestSearchPatients_Pagination(t *testing.T) {
	bundleData := `{"resourceType":"Bundle","type":"searchset","total":50,"link":[{"relation":"self","url":"test"},{"relation":"next","url":"test?_offset=20"}],"entry":[]}`

	mock := &mockPatientService{
		searchFn: func(ctx context.Context, params repository.PatientSearchParams, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error) {
			assert.Equal(t, 20, params.Count)
			assert.Equal(t, 0, params.Offset)
			return &services.BundleResponse{Data: json.RawMessage(bundleData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient?_count=20&_offset=0", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var bundle map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &bundle)
	links := bundle["link"].([]interface{})
	assert.GreaterOrEqual(t, len(links), 1)
}

func TestGetPatientHistory_200(t *testing.T) {
	bundleData := `{"resourceType":"Bundle","type":"history","total":2,"entry":[{},{}]}`

	mock := &mockPatientService{
		historyFn: func(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*services.BundleResponse, error) {
			return &services.BundleResponse{Data: json.RawMessage(bundleData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient/123/_history", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var bundle map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &bundle)
	assert.Equal(t, "Bundle", bundle["resourceType"])
	assert.Equal(t, "history", bundle["type"])
}

func TestGetPatientVersion_200(t *testing.T) {
	patientData := `{"resourceType":"Patient","id":"123","meta":{"versionId":"1"}}`

	mock := &mockPatientService{
		versionFn: func(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*services.PatientResponse, error) {
			assert.Equal(t, 1, versionID)
			return &services.PatientResponse{Data: json.RawMessage(patientData)}, nil
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/Patient/123/_history/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreatePatient_400_InvalidJSON(t *testing.T) {
	handler := NewPatientHandler(&mockPatientService{})
	router := setupRouter(handler)

	body := bytes.NewBufferString(`{invalid json`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/fhir/R4/Patient", body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeletePatient_404(t *testing.T) {
	mock := &mockPatientService{
		deleteFn: func(ctx context.Context, resourceID string, actorID uuid.UUID) error {
			return fhirerrors.NewNotFoundError("Patient", resourceID)
		},
	}

	handler := NewPatientHandler(mock)
	router := setupRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/fhir/R4/Patient/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
