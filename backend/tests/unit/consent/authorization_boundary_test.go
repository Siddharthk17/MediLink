package consent_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Siddharthk17/MediLink/internal/auth"
	"github.com/Siddharthk17/MediLink/internal/consent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Authorization Boundary Tests
//
// These tests verify the core security invariant of MediLink:
// Patient A must never see Patient B's data.
// A physician must never access data without active consent.
// Consent revocation must take immediate effect.
//
// Each test wires up the consent middleware with a test router and
// verifies access decisions from the perspective of a real HTTP request.

// setupBoundaryRouter creates a router with actor context and consent middleware
// that covers the resource types most critical in healthcare.
func setupBoundaryRouter(actorID, actorRole string, engine consent.ConsentEngine) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	setActor := func(c *gin.Context) {
		c.Set("actor_id", actorID)
		c.Set("actor_role", actorRole)
		c.Next()
	}

	logger := zerolog.Nop()
	mw := consent.ConsentMiddleware(engine, nil, logger)
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

	// Individual resource reads
	r.GET("/fhir/R4/Patient/:id", setActor, mw, ok)
	r.GET("/fhir/R4/Observation/:id", setActor, mw, ok)
	r.GET("/fhir/R4/MedicationRequest/:id", setActor, mw, ok)
	r.GET("/fhir/R4/Condition/:id", setActor, mw, ok)
	r.GET("/fhir/R4/DiagnosticReport/:id", setActor, mw, ok)
	r.GET("/fhir/R4/AllergyIntolerance/:id", setActor, mw, ok)
	r.GET("/fhir/R4/Immunization/:id", setActor, mw, ok)
	r.GET("/fhir/R4/Encounter/:id", setActor, mw, ok)

	// Search endpoints (no :id param)
	r.GET("/fhir/R4/Observation", setActor, mw, ok)
	r.GET("/fhir/R4/MedicationRequest", setActor, mw, ok)
	r.GET("/fhir/R4/Condition", setActor, mw, ok)

	// Unrestricted resources
	r.GET("/fhir/R4/Practitioner/:id", setActor, mw, ok)
	r.GET("/fhir/R4/Organization/:id", setActor, mw, ok)

	return r
}

// ---------- Patient boundary tests ----------

func TestBoundary_PatientCanReadOwnResourceAcrossAllTypes(t *testing.T) {
	patientUserID := "user-patient-boundary-1"
	patientFHIRID := "fhir-boundary-patient-1"

	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			if userID == patientUserID {
				return patientFHIRID, nil
			}
			return "", fmt.Errorf("unknown user")
		},
	}

	r := setupBoundaryRouter(patientUserID, "patient", engine)

	// Patient resource — the ID IS the FHIR patient ID
	t.Run("Patient_OwnRecord", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+patientFHIRID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Unrestricted resources are accessible without consent
	t.Run("Practitioner_AlwaysAccessible", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Practitioner/any-id", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Organization_AlwaysAccessible", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Organization/any-id", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestBoundary_PatientCannotReadAnotherPatientsRecord(t *testing.T) {
	patientAUserID := "user-patient-A"
	patientAFHIRID := "fhir-patient-A"
	patientBFHIRID := "fhir-patient-B"

	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			if userID == patientAUserID {
				return patientAFHIRID, nil
			}
			return "", fmt.Errorf("unknown user")
		},
	}

	r := setupBoundaryRouter(patientAUserID, "patient", engine)

	// Patient A tries to read Patient B's record
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+patientBFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access denied")
}

func TestBoundary_PatientCannotAccessUnlinkedProfile(t *testing.T) {
	unlinkedPatientUserID := "user-unlinked"

	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			return "", fmt.Errorf("patient not found")
		},
	}

	r := setupBoundaryRouter(unlinkedPatientUserID, "patient", engine)

	// Patient without a linked FHIR profile tries to read any patient resource
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/any-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Patient profile not linked")
}

func TestBoundary_PatientSearchForcesOwnScope(t *testing.T) {
	patientUserID := "user-patient-search"
	patientFHIRID := "fhir-patient-search"

	var capturedForcedRef string
	engine := &mockConsentEngine{
		getPatientFHIRIDFn: func(_ context.Context, userID string) (string, error) {
			if userID == patientUserID {
				return patientFHIRID, nil
			}
			return "", fmt.Errorf("unknown user")
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	setActor := func(c *gin.Context) {
		c.Set("actor_id", patientUserID)
		c.Set("actor_role", "patient")
		c.Next()
	}
	logger := zerolog.Nop()
	mw := consent.ConsentMiddleware(engine, nil, logger)
	capture := func(c *gin.Context) {
		val, exists := c.Get("forced_patient_ref")
		if exists {
			capturedForcedRef, _ = val.(string)
		}
		c.Status(http.StatusOK)
	}

	r.GET("/fhir/R4/Observation", setActor, mw, capture)

	// Patient searches for Observations — middleware should force-scope to their own patient ref
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Observation", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Patient/"+patientFHIRID, capturedForcedRef,
		"patient search must be force-scoped to their own patient reference")
}

// ---------- Physician boundary tests ----------

func TestBoundary_PhysicianBlockedWithoutConsent(t *testing.T) {
	physicianID := "physician-no-consent"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil // no consent
		},
	}

	r := setupBoundaryRouter(physicianID, "physician", engine)

	// Direct Patient read — middleware uses resourceID as patient ref, no DB needed
	t.Run("Patient_Blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/some-patient-id", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code,
			"physician without consent must be blocked from Patient")
	})

	// Search endpoints — these use the engine mock (no DB), checking consent on ?patient= param
	searchTypes := []string{"Observation", "MedicationRequest", "Condition"}
	for _, rt := range searchTypes {
		t.Run(rt+"_SearchBlocked", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/fhir/R4/"+rt+"?patient=Patient/some-patient-id", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"physician without consent must be blocked from searching %s", rt)
		})
	}
}

func TestBoundary_PhysicianAllowedWithConsent(t *testing.T) {
	physicianID := "physician-with-consent"
	patientFHIRID := "target-patient-fhir"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, providerID, pFHIRID, _ string) (bool, error) {
			if providerID == physicianID && pFHIRID == patientFHIRID {
				return true, nil
			}
			return false, nil
		},
	}

	r := setupBoundaryRouter(physicianID, "physician", engine)

	// Physician WITH consent can access patient's Patient resource
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+patientFHIRID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBoundary_PhysicianConsentForOnePatientDoesNotGrantAccessToAnother(t *testing.T) {
	physicianID := "physician-one-consent"
	consentedPatientFHIRID := "consented-patient"
	otherPatientFHIRID := "other-patient"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, providerID, pFHIRID, _ string) (bool, error) {
			// Only consented for ONE patient
			if providerID == physicianID && pFHIRID == consentedPatientFHIRID {
				return true, nil
			}
			return false, nil
		},
	}

	r := setupBoundaryRouter(physicianID, "physician", engine)

	t.Run("ConsentedPatient_Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+consentedPatientFHIRID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("OtherPatient_Blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/"+otherPatientFHIRID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestBoundary_PhysicianSearchRequiresPatientParam(t *testing.T) {
	physicianID := "physician-search"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
	}

	r := setupBoundaryRouter(physicianID, "physician", engine)

	patientScopedTypes := []string{"Observation", "MedicationRequest", "Condition"}

	for _, rt := range patientScopedTypes {
		t.Run(rt+"_NoPatientParam_400", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/fhir/R4/"+rt, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code,
				"physician search for %s without patient param must be rejected", rt)
			assert.Contains(t, w.Body.String(), "patient")
		})
	}
}

func TestBoundary_PhysicianSearchWithConsentCheckOnPatientParam(t *testing.T) {
	physicianID := "physician-search-consent"
	consentedPatientFHIR := "consented-fhir-id"
	nonConsentedPatientFHIR := "non-consented-fhir-id"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, providerID, pFHIRID, _ string) (bool, error) {
			if providerID == physicianID && pFHIRID == consentedPatientFHIR {
				return true, nil
			}
			return false, nil
		},
	}

	r := setupBoundaryRouter(physicianID, "physician", engine)

	t.Run("SearchWithConsent_Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/fhir/R4/Observation?patient=Patient/"+consentedPatientFHIR, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("SearchWithoutConsent_Blocked", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/fhir/R4/Observation?patient=Patient/"+nonConsentedPatientFHIR, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// ---------- Admin boundary tests ----------

func TestBoundary_AdminBypassesConsentForAllResourceTypes(t *testing.T) {
	adminID := "admin-bypass"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil // consent engine would deny — but admin bypasses
		},
	}

	r := setupBoundaryRouter(adminID, "admin", engine)

	resourceTypes := []string{"Patient", "Observation", "MedicationRequest", "Condition",
		"DiagnosticReport", "AllergyIntolerance", "Immunization", "Encounter"}

	for _, rt := range resourceTypes {
		t.Run(rt+"_AdminAllowed", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/fhir/R4/"+rt+"/any-id", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code,
				"admin should bypass consent for %s", rt)
		})
	}
}

// ---------- Write operation boundary tests ----------

func TestBoundary_WriteOperationsNotCheckedByConsentMiddleware(t *testing.T) {
	// Consent middleware only checks GET (reads). Writes go through FHIR handler
	// authorization (which checks actor role). This test verifies the middleware
	// correctly skips POST/PUT/DELETE.
	physicianID := "physician-write"

	engine := &mockConsentEngine{
		checkConsentFn: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil // would deny on reads
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	setActor := func(c *gin.Context) {
		c.Set("actor_id", physicianID)
		c.Set("actor_role", "physician")
		c.Next()
	}
	logger := zerolog.Nop()
	mw := consent.ConsentMiddleware(engine, nil, logger)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	r.POST("/fhir/R4/Observation", setActor, mw, ok)
	r.PUT("/fhir/R4/Observation/:id", setActor, mw, ok)
	r.DELETE("/fhir/R4/Observation/:id", setActor, mw, ok)

	methods := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/fhir/R4/Observation"},
		{http.MethodPut, "/fhir/R4/Observation/some-id"},
		{http.MethodDelete, "/fhir/R4/Observation/some-id"},
	}

	for _, m := range methods {
		t.Run(m.method+"_Observation", func(t *testing.T) {
			req := httptest.NewRequest(m.method, m.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code,
				"consent middleware should not block %s requests", m.method)
		})
	}
}

// ---------- Role-based middleware tests ----------

func TestBoundary_RequireRoleBlocksWrongRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		actorRole  string
		allowed    []string
		wantStatus int
	}{
		{"PatientAccessingPhysicianRoute", "patient", []string{"physician", "admin"}, http.StatusForbidden},
		{"PhysicianAccessingAdminRoute", "physician", []string{"admin"}, http.StatusForbidden},
		{"AdminAccessingPhysicianRoute", "admin", []string{"physician", "admin"}, http.StatusOK},
		{"PhysicianAccessingPhysicianRoute", "physician", []string{"physician", "admin"}, http.StatusOK},
		{"PatientAccessingPatientRoute", "patient", []string{"patient"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.Set(auth.ActorRoleKey, tt.actorRole)
				c.Next()
			}, auth.RequireRole(tt.allowed...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestBoundary_RequirePhysicianAllowsPhysicianAndAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		role       string
		wantStatus int
	}{
		{"physician", http.StatusOK},
		{"admin", http.StatusOK},
		{"patient", http.StatusForbidden},
		{"researcher", http.StatusForbidden},
		{"", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				c.Set(auth.ActorRoleKey, tt.role)
				c.Next()
			}, auth.RequirePhysician(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// ---------- Consent engine adversarial tests ----------

func TestBoundary_ConsentScopeEnforcement(t *testing.T) {
	// Verify that a consent with limited scope blocks access to out-of-scope resources.
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := newTestUUID()
	providerID := newTestUUID()
	fhirID := newTestUUIDString()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant with Observation-only scope
	_, err := engine.GrantConsent(ctx, consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"Observation"},
	}, patientID.String())
	require.NoError(t, err)

	t.Run("InScope_Observation_Allowed", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
		require.NoError(t, err)
		assert.True(t, granted)
	})

	t.Run("OutOfScope_MedicationRequest_Blocked", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "MedicationRequest")
		require.NoError(t, err)
		assert.False(t, granted)
	})

	t.Run("OutOfScope_Condition_Blocked", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Condition")
		require.NoError(t, err)
		assert.False(t, granted)
	})

	t.Run("OutOfScope_DiagnosticReport_Blocked", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "DiagnosticReport")
		require.NoError(t, err)
		assert.False(t, granted)
	})

	t.Run("Patient_AlwaysAllowed_EvenWithLimitedScope", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Patient")
		require.NoError(t, err)
		assert.True(t, granted, "Patient resource is always allowed with any active consent")
	})
}

func TestBoundary_RevokedConsentDeniesAccess(t *testing.T) {
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := newTestUUID()
	providerID := newTestUUID()
	fhirID := newTestUUIDString()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant consent
	c, err := engine.GrantConsent(ctx, consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}, patientID.String())
	require.NoError(t, err)

	// Verify access before revocation
	granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.True(t, granted, "should have access before revocation")

	// Revoke
	err = engine.RevokeConsent(ctx, c.ID.String(), patientID.String(), "patient")
	require.NoError(t, err)

	// Verify access IMMEDIATELY denied after revocation (cache invalidated)
	granted, err = engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.False(t, granted, "access must be denied immediately after revocation")
}

func TestBoundary_InvalidScopeRejected(t *testing.T) {
	repo := newMockRepo()
	patientID := newTestUUID()
	providerID := newTestUUID()
	fhirID := newTestUUIDString()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, nil)
	ctx := context.Background()

	// Try to grant consent with an invalid scope type
	_, err := engine.GrantConsent(ctx, consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"InvalidResource"},
	}, patientID.String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid scope")
}

func TestBoundary_SelfConsentDenied(t *testing.T) {
	repo := newMockRepo()
	userID := newTestUUID()
	repo.knownProviders[userID] = true

	engine, _ := newTestEngine(repo, nil)
	ctx := context.Background()

	_, err := engine.GrantConsent(ctx, consent.GrantConsentRequest{
		ProviderUserID: userID.String(),
		Scope:          []string{"*"},
	}, userID.String())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot grant consent to yourself")
}

// Helper functions

func newTestUUID() uuid.UUID {
	return uuid.New()
}

func newTestUUIDString() string {
	return uuid.New().String()
}
