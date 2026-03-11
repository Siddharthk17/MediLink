package consent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Siddharthk17/MediLink/internal/consent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// mockConsentEngine implements consent.ConsentEngine for testing.
type mockConsentEngine struct {
	checkConsentFn     func(ctx context.Context, providerID, patientFHIRID, resourceType string) (bool, error)
	getPatientFHIRIDFn func(ctx context.Context, userID string) (string, error)
}

func (m *mockConsentEngine) CheckConsent(ctx context.Context, providerID, patientFHIRID, resourceType string) (bool, error) {
	if m.checkConsentFn != nil {
		return m.checkConsentFn(ctx, providerID, patientFHIRID, resourceType)
	}
	return false, nil
}

func (m *mockConsentEngine) GetConsentScope(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockConsentEngine) GrantConsent(_ context.Context, _ consent.GrantConsentRequest, _ string) (*consent.Consent, error) {
	return nil, nil
}

func (m *mockConsentEngine) RevokeConsent(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockConsentEngine) GetPatientConsents(_ context.Context, _ string) ([]*consent.Consent, error) {
	return nil, nil
}

func (m *mockConsentEngine) GetPhysicianPatients(_ context.Context, _ string) ([]*consent.ConsentedPatient, error) {
	return nil, nil
}

func (m *mockConsentEngine) RecordBreakGlass(_ context.Context, _ consent.BreakGlassRequest, _ string) (*consent.Consent, error) {
	return nil, nil
}

func (m *mockConsentEngine) InvalidateCache(_ context.Context, _, _ string) {}

func (m *mockConsentEngine) AcceptConsent(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockConsentEngine) DeclineConsent(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockConsentEngine) GetPendingRequests(_ context.Context, _ string) ([]*consent.ConsentedPatient, error) {
	return nil, nil
}

func (m *mockConsentEngine) GetConsentByID(_ context.Context, _ uuid.UUID) (*consent.Consent, error) {
	return nil, nil
}

func (m *mockConsentEngine) GetAccessLog(_ context.Context, _ string) ([]consent.AccessLogEntry, error) {
	return []consent.AccessLogEntry{}, nil
}

func (m *mockConsentEngine) GetUserDisplayName(_ context.Context, _ uuid.UUID) string {
	return ""
}

func (m *mockConsentEngine) GetPatientFHIRID(ctx context.Context, userID string) (string, error) {
	if m.getPatientFHIRIDFn != nil {
		return m.getPatientFHIRIDFn(ctx, userID)
	}
	return "", nil
}

func setupConsentRouter(actorID, actorRole string, engine consent.ConsentEngine) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	setActor := func(c *gin.Context) {
		c.Set("actor_id", actorID)
		c.Set("actor_role", actorRole)
		c.Next()
	}

	logger := zerolog.Nop()
	mw := consent.ConsentMiddleware(engine, nil, logger)
	handler := func(c *gin.Context) { c.Status(http.StatusOK) }

	r.GET("/fhir/R4/Patient/:id", setActor, mw, handler)
	r.GET("/fhir/R4/Observation", setActor, mw, handler)
	return r
}

func TestConsentMiddleware_PatientOwnData(t *testing.T) {
	patientUserID := "user-patient-001"
	patientFHIRID := "fhir-patient-001"

	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			if userID == patientUserID {
				return patientFHIRID, nil
			}
			return "", nil
		},
	}

	r := setupConsentRouter(patientUserID, "patient", engine)

	// Patient reads their own Patient resource
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+patientFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsentMiddleware_PatientOtherData(t *testing.T) {
	patientUserID := "user-patient-001"
	patientFHIRID := "fhir-patient-001"
	otherFHIRID := "fhir-patient-999"

	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			if userID == patientUserID {
				return patientFHIRID, nil
			}
			return "", nil
		},
	}

	r := setupConsentRouter(patientUserID, "patient", engine)

	// Patient tries to read another patient's record
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+otherFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access denied")
}

func TestConsentMiddleware_PhysicianConsent(t *testing.T) {
	physicianUserID := "physician-001"
	targetPatientFHIRID := "fhir-patient-050"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, providerID, patientFHIRID, _ string) (bool, error) {
			if providerID == physicianUserID && patientFHIRID == targetPatientFHIRID {
				return true, nil
			}
			return false, nil
		},
	}

	r := setupConsentRouter(physicianUserID, "physician", engine)

	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+targetPatientFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsentMiddleware_PhysicianNoConsent(t *testing.T) {
	physicianUserID := "physician-001"
	targetPatientFHIRID := "fhir-patient-050"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}

	r := setupConsentRouter(physicianUserID, "physician", engine)

	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+targetPatientFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "No active consent")
}

func TestConsentMiddleware_AdminBypass(t *testing.T) {
	adminUserID := "admin-001"

	engine := &mockConsentEngine{}

	r := setupConsentRouter(adminUserID, "admin", engine)

	// Admin should bypass consent completely
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/any-patient-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
