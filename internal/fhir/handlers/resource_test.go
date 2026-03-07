package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Siddharthk17/medilink/internal/auth"
	"github.com/Siddharthk17/medilink/internal/fhir/services"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// ---------------------------------------------------------------------------
// Mock implementation of services.ResourceService
// ---------------------------------------------------------------------------

type mockResourceService struct {
	createFn     func(ctx context.Context, data json.RawMessage, actorID uuid.UUID) (*services.PatientResponse, error)
	getFn        func(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error)
	getVersionFn func(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*services.PatientResponse, error)
	getHistoryFn func(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*services.BundleResponse, error)
	updateFn     func(ctx context.Context, resourceID string, data json.RawMessage, actorID uuid.UUID) (*services.PatientResponse, error)
	deleteFn     func(ctx context.Context, resourceID string, actorID uuid.UUID) error
	searchFn     func(ctx context.Context, conditions []string, args []interface{}, count, offset int, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error)
}

func (m *mockResourceService) Create(ctx context.Context, data json.RawMessage, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.createFn(ctx, data, actorID)
}

func (m *mockResourceService) Get(ctx context.Context, resourceID string, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.getFn(ctx, resourceID, actorID)
}

func (m *mockResourceService) GetVersion(ctx context.Context, resourceID string, versionID int, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.getVersionFn(ctx, resourceID, versionID, actorID)
}

func (m *mockResourceService) GetHistory(ctx context.Context, resourceID string, actorID uuid.UUID, count, offset int) (*services.BundleResponse, error) {
	return m.getHistoryFn(ctx, resourceID, actorID, count, offset)
}

func (m *mockResourceService) Update(ctx context.Context, resourceID string, data json.RawMessage, actorID uuid.UUID) (*services.PatientResponse, error) {
	return m.updateFn(ctx, resourceID, data, actorID)
}

func (m *mockResourceService) Delete(ctx context.Context, resourceID string, actorID uuid.UUID) error {
	return m.deleteFn(ctx, resourceID, actorID)
}

func (m *mockResourceService) Search(ctx context.Context, conditions []string, args []interface{}, count, offset int, actorID uuid.UUID, baseURL string) (*services.BundleResponse, error) {
	return m.searchFn(ctx, conditions, args, count, offset, actorID, baseURL)
}

// ---------------------------------------------------------------------------
// Router / parser helpers
// ---------------------------------------------------------------------------

func setupResourceRouter(resourceType string, svc services.ResourceService, parser SearchParamParser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(auth.AuthMiddleware())
	handler := NewResourceHandler(svc, resourceType, parser)
	group := r.Group("/fhir/R4/" + resourceType)
	group.POST("", handler.Create)
	group.GET("", handler.Search)
	group.GET("/:id", handler.Get)
	group.PUT("/:id", handler.Update)
	group.DELETE("/:id", handler.Delete)
	group.GET("/:id/_history", handler.GetHistory)
	group.GET("/:id/_history/:vid", handler.GetVersion)
	return r
}

func noopParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	return nil, nil, 1, 20, 0
}

// ---------------------------------------------------------------------------
// Reusable test helpers – each exercises a single HTTP verb / status scenario.
// ---------------------------------------------------------------------------

func testCreate201(t *testing.T, resourceType string) {
	t.Helper()
	data := fmt.Sprintf(`{"resourceType":"%s","id":"test-id"}`, resourceType)
	mock := &mockResourceService{
		createFn: func(_ context.Context, _ json.RawMessage, _ uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(data)}, nil
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	body := bytes.NewBufferString(fmt.Sprintf(`{"resourceType":"%s"}`, resourceType))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/fhir/R4/"+resourceType, body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/fhir/R4/"+resourceType+"/test-id")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/fhir+json")
}

func testCreate400InvalidJSON(t *testing.T, resourceType string) {
	t.Helper()
	mock := &mockResourceService{}
	router := setupResourceRouter(resourceType, mock, noopParser)

	body := bytes.NewBufferString(`{invalid json`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/fhir/R4/"+resourceType, body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func testGet200(t *testing.T, resourceType string) {
	t.Helper()
	data := fmt.Sprintf(`{"resourceType":"%s","id":"123"}`, resourceType)
	mock := &mockResourceService{
		getFn: func(_ context.Context, _ string, _ uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(data)}, nil
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/"+resourceType+"/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), resourceType)
}

func testGet404(t *testing.T, resourceType string) {
	t.Helper()
	mock := &mockResourceService{
		getFn: func(_ context.Context, resourceID string, _ uuid.UUID) (*services.PatientResponse, error) {
			return nil, fhirerrors.NewNotFoundError(resourceType, resourceID)
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/"+resourceType+"/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var outcome fhirerrors.OperationOutcome
	err := json.Unmarshal(w.Body.Bytes(), &outcome)
	assert.NoError(t, err)
	assert.Equal(t, "OperationOutcome", outcome.ResourceType)
	assert.Equal(t, "not-found", outcome.Issue[0].Code)
}

func testUpdate200(t *testing.T, resourceType string) {
	t.Helper()
	data := fmt.Sprintf(`{"resourceType":"%s","id":"123","meta":{"versionId":"2"}}`, resourceType)
	mock := &mockResourceService{
		updateFn: func(_ context.Context, _ string, _ json.RawMessage, _ uuid.UUID) (*services.PatientResponse, error) {
			return &services.PatientResponse{Data: json.RawMessage(data)}, nil
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	body := bytes.NewBufferString(fmt.Sprintf(`{"resourceType":"%s","name":"updated"}`, resourceType))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/fhir/R4/"+resourceType+"/123", body)
	req.Header.Set("Content-Type", "application/fhir+json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func testDelete204(t *testing.T, resourceType string) {
	t.Helper()
	mock := &mockResourceService{
		deleteFn: func(_ context.Context, _ string, _ uuid.UUID) error {
			return nil
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/fhir/R4/"+resourceType+"/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func testSearch200(t *testing.T, resourceType string) {
	t.Helper()
	bundleData := `{"resourceType":"Bundle","type":"searchset","total":1,"entry":[{"resource":{"resourceType":"` + resourceType + `"}}]}`
	mock := &mockResourceService{
		searchFn: func(_ context.Context, _ []string, _ []interface{}, _, _ int, _ uuid.UUID, _ string) (*services.BundleResponse, error) {
			return &services.BundleResponse{Data: json.RawMessage(bundleData)}, nil
		},
	}
	router := setupResourceRouter(resourceType, mock, noopParser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fhir/R4/"+resourceType+"?_count=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var bundle map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &bundle)
	assert.Equal(t, "Bundle", bundle["resourceType"])
	assert.Equal(t, "searchset", bundle["type"])
}

// ===========================================================================
// Practitioner
// ===========================================================================

func TestCreatePractitioner_201(t *testing.T)              { testCreate201(t, "Practitioner") }
func TestCreatePractitioner_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Practitioner") }
func TestGetPractitioner_200(t *testing.T)                  { testGet200(t, "Practitioner") }
func TestGetPractitioner_404(t *testing.T)                  { testGet404(t, "Practitioner") }
func TestUpdatePractitioner_200(t *testing.T)               { testUpdate200(t, "Practitioner") }
func TestDeletePractitioner_204(t *testing.T)               { testDelete204(t, "Practitioner") }
func TestSearchPractitioner_200(t *testing.T)               { testSearch200(t, "Practitioner") }

// ===========================================================================
// Organization
// ===========================================================================

func TestCreateOrganization_201(t *testing.T)              { testCreate201(t, "Organization") }
func TestCreateOrganization_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Organization") }
func TestGetOrganization_200(t *testing.T)                  { testGet200(t, "Organization") }
func TestGetOrganization_404(t *testing.T)                  { testGet404(t, "Organization") }
func TestUpdateOrganization_200(t *testing.T)               { testUpdate200(t, "Organization") }
func TestDeleteOrganization_204(t *testing.T)               { testDelete204(t, "Organization") }
func TestSearchOrganization_200(t *testing.T)               { testSearch200(t, "Organization") }

// ===========================================================================
// Encounter
// ===========================================================================

func TestCreateEncounter_201(t *testing.T)              { testCreate201(t, "Encounter") }
func TestCreateEncounter_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Encounter") }
func TestGetEncounter_200(t *testing.T)                  { testGet200(t, "Encounter") }
func TestGetEncounter_404(t *testing.T)                  { testGet404(t, "Encounter") }
func TestUpdateEncounter_200(t *testing.T)               { testUpdate200(t, "Encounter") }
func TestDeleteEncounter_204(t *testing.T)               { testDelete204(t, "Encounter") }
func TestSearchEncounter_200(t *testing.T)               { testSearch200(t, "Encounter") }

// ===========================================================================
// Condition
// ===========================================================================

func TestCreateCondition_201(t *testing.T)              { testCreate201(t, "Condition") }
func TestCreateCondition_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Condition") }
func TestGetCondition_200(t *testing.T)                  { testGet200(t, "Condition") }
func TestGetCondition_404(t *testing.T)                  { testGet404(t, "Condition") }
func TestUpdateCondition_200(t *testing.T)               { testUpdate200(t, "Condition") }
func TestDeleteCondition_204(t *testing.T)               { testDelete204(t, "Condition") }
func TestSearchCondition_200(t *testing.T)               { testSearch200(t, "Condition") }

// ===========================================================================
// MedicationRequest
// ===========================================================================

func TestCreateMedicationRequest_201(t *testing.T)              { testCreate201(t, "MedicationRequest") }
func TestCreateMedicationRequest_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "MedicationRequest") }
func TestGetMedicationRequest_200(t *testing.T)                  { testGet200(t, "MedicationRequest") }
func TestGetMedicationRequest_404(t *testing.T)                  { testGet404(t, "MedicationRequest") }
func TestUpdateMedicationRequest_200(t *testing.T)               { testUpdate200(t, "MedicationRequest") }
func TestDeleteMedicationRequest_204(t *testing.T)               { testDelete204(t, "MedicationRequest") }
func TestSearchMedicationRequest_200(t *testing.T)               { testSearch200(t, "MedicationRequest") }

// ===========================================================================
// Observation
// ===========================================================================

func TestCreateObservation_201(t *testing.T)              { testCreate201(t, "Observation") }
func TestCreateObservation_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Observation") }
func TestGetObservation_200(t *testing.T)                  { testGet200(t, "Observation") }
func TestGetObservation_404(t *testing.T)                  { testGet404(t, "Observation") }
func TestUpdateObservation_200(t *testing.T)               { testUpdate200(t, "Observation") }
func TestDeleteObservation_204(t *testing.T)               { testDelete204(t, "Observation") }
func TestSearchObservation_200(t *testing.T)               { testSearch200(t, "Observation") }

// ===========================================================================
// DiagnosticReport
// ===========================================================================

func TestCreateDiagnosticReport_201(t *testing.T)              { testCreate201(t, "DiagnosticReport") }
func TestCreateDiagnosticReport_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "DiagnosticReport") }
func TestGetDiagnosticReport_200(t *testing.T)                  { testGet200(t, "DiagnosticReport") }
func TestGetDiagnosticReport_404(t *testing.T)                  { testGet404(t, "DiagnosticReport") }
func TestUpdateDiagnosticReport_200(t *testing.T)               { testUpdate200(t, "DiagnosticReport") }
func TestDeleteDiagnosticReport_204(t *testing.T)               { testDelete204(t, "DiagnosticReport") }
func TestSearchDiagnosticReport_200(t *testing.T)               { testSearch200(t, "DiagnosticReport") }

// ===========================================================================
// AllergyIntolerance
// ===========================================================================

func TestCreateAllergyIntolerance_201(t *testing.T)              { testCreate201(t, "AllergyIntolerance") }
func TestCreateAllergyIntolerance_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "AllergyIntolerance") }
func TestGetAllergyIntolerance_200(t *testing.T)                  { testGet200(t, "AllergyIntolerance") }
func TestGetAllergyIntolerance_404(t *testing.T)                  { testGet404(t, "AllergyIntolerance") }
func TestUpdateAllergyIntolerance_200(t *testing.T)               { testUpdate200(t, "AllergyIntolerance") }
func TestDeleteAllergyIntolerance_204(t *testing.T)               { testDelete204(t, "AllergyIntolerance") }
func TestSearchAllergyIntolerance_200(t *testing.T)               { testSearch200(t, "AllergyIntolerance") }

// ===========================================================================
// Immunization
// ===========================================================================

func TestCreateImmunization_201(t *testing.T)              { testCreate201(t, "Immunization") }
func TestCreateImmunization_400_InvalidJSON(t *testing.T)   { testCreate400InvalidJSON(t, "Immunization") }
func TestGetImmunization_200(t *testing.T)                  { testGet200(t, "Immunization") }
func TestGetImmunization_404(t *testing.T)                  { testGet404(t, "Immunization") }
func TestUpdateImmunization_200(t *testing.T)               { testUpdate200(t, "Immunization") }
func TestDeleteImmunization_204(t *testing.T)               { testDelete204(t, "Immunization") }
func TestSearchImmunization_200(t *testing.T)               { testSearch200(t, "Immunization") }
