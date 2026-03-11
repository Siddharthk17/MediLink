package consent

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Siddharthk17/MediLink/internal/auth"
)

type ConsentHandler struct {
	engine ConsentEngine
}

func NewConsentHandler(engine ConsentEngine) *ConsentHandler {
	return &ConsentHandler{engine: engine}
}

// GrantConsent handles POST /consent/grant
func (h *ConsentHandler) GrantConsent(c *gin.Context) {
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	// Only patients and admins can grant consent
	if actorRole != "patient" && actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Only patients can grant consent"}},
		})
		return
	}

	var req GrantConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
		})
		return
	}

	consent, err := h.engine.GrantConsent(c.Request.Context(), req, actorID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrSelfConsent) || errors.Is(err, ErrInvalidScope) || errors.Is(err, ErrExpiredDate) || errors.Is(err, ErrProviderNotPhysician) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, ErrProviderNotFound) || errors.Is(err, ErrPatientNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusCreated, consent)
}

// RevokeConsent handles DELETE /consent/:consentId/revoke
func (h *ConsentHandler) RevokeConsent(c *gin.Context) {
	consentID := c.Param("consentId")
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	err := h.engine.RevokeConsent(c.Request.Context(), consentID, actorID, actorRole)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrConsentNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, ErrUnauthorizedAction) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent revoked successfully"})
}

// GetMyGrants handles GET /consent/my-grants
func (h *ConsentHandler) GetMyGrants(c *gin.Context) {
	actorID := auth.GetActorID(c).String()

	consents, err := h.engine.GetPatientConsents(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	type enrichedConsent struct {
		*Consent
		ProviderName string `json:"providerName,omitempty"`
	}
	enriched := make([]enrichedConsent, 0, len(consents))
	for _, consent := range consents {
		name := h.engine.GetUserDisplayName(c.Request.Context(), consent.ProviderID)
		enriched = append(enriched, enrichedConsent{Consent: consent, ProviderName: name})
	}

	c.JSON(http.StatusOK, gin.H{"consents": enriched, "total": len(enriched)})
}

// GetMyPatients handles GET /consent/my-patients
func (h *ConsentHandler) GetMyPatients(c *gin.Context) {
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	if actorRole != "physician" && actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Only physicians can view their patients"}},
		})
		return
	}

	patients, err := h.engine.GetPhysicianPatients(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	// Transform to nested structure matching frontend ConsentedPatient type
	type patientInfo struct {
		ID        string `json:"id"`
		FhirID    string `json:"fhirId"`
		FullName  string `json:"fullName"`
		Gender    string `json:"gender,omitempty"`
		BirthDate string `json:"birthDate,omitempty"`
	}
	type consentInfo struct {
		ID        string   `json:"id"`
		Status    string   `json:"status"`
		Scope     []string `json:"scope"`
		GrantedAt string   `json:"grantedAt"`
		ExpiresAt string   `json:"expiresAt,omitempty"`
	}
	type enrichedEntry struct {
		Patient patientInfo `json:"patient"`
		Consent consentInfo `json:"consent"`
	}

	entries := make([]enrichedEntry, 0, len(patients))
	for _, p := range patients {
		// Extract patient info from FHIR data
		var fhirData struct {
			Name      []struct{ Text string `json:"text"` } `json:"name"`
			Gender    string                                 `json:"gender"`
			BirthDate string                                 `json:"birthDate"`
		}
		if len(p.PatientData) > 0 {
			json.Unmarshal(p.PatientData, &fhirData)
		}
		fullName := ""
		if len(fhirData.Name) > 0 {
			fullName = fhirData.Name[0].Text
		}

		// Parse scope from JSON
		var scope []string
		json.Unmarshal(p.Scope, &scope)
		if scope == nil {
			scope = []string{}
		}

		status := p.Status
		if status == "" {
			status = "active"
		}

		entry := enrichedEntry{
			Patient: patientInfo{
				ID:        p.PatientID.String(),
				FhirID:    p.FHIRPatientID,
				FullName:  fullName,
				Gender:    fhirData.Gender,
				BirthDate: fhirData.BirthDate,
			},
			Consent: consentInfo{
				ID:        p.ConsentID.String(),
				Status:    status,
				Scope:     scope,
				GrantedAt: p.GrantedAt.Format("2006-01-02T15:04:05Z"),
			},
		}
		if p.ExpiresAt != nil {
			entry.Consent.ExpiresAt = p.ExpiresAt.Format("2006-01-02T15:04:05Z")
		}
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, gin.H{"patients": entries, "total": len(entries)})
}

// GetConsent handles GET /consent/:consentId
func (h *ConsentHandler) GetConsent(c *gin.Context) {
	consentID := c.Param("consentId")

	id, err := uuid.Parse(consentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "Invalid consent ID"}},
		})
		return
	}

	consent, err := h.engine.GetConsentByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}
	if consent == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "not-found", "diagnostics": "Consent not found"}},
		})
		return
	}

	c.JSON(http.StatusOK, consent)
}

// AcceptConsent handles PUT /consent/:consentId/accept
func (h *ConsentHandler) AcceptConsent(c *gin.Context) {
	consentID := c.Param("consentId")
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	if actorRole != "physician" && actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Only physicians can accept consent requests"}},
		})
		return
	}

	if err := h.engine.AcceptConsent(c.Request.Context(), consentID, actorID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrConsentNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, ErrNotPending) || errors.Is(err, ErrUnauthorizedAction) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent request accepted"})
}

// DeclineConsent handles PUT /consent/:consentId/decline
func (h *ConsentHandler) DeclineConsent(c *gin.Context) {
	consentID := c.Param("consentId")
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	if actorRole != "physician" && actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Only physicians can decline consent requests"}},
		})
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&body)

	if err := h.engine.DeclineConsent(c.Request.Context(), consentID, actorID, body.Reason); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrConsentNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, ErrNotPending) || errors.Is(err, ErrUnauthorizedAction) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Consent request declined"})
}

// GetPendingRequests handles GET /consent/pending-requests
func (h *ConsentHandler) GetPendingRequests(c *gin.Context) {
	actorID := auth.GetActorID(c).String()

	patients, err := h.engine.GetPendingRequests(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	type patientInfo struct {
		ID        string `json:"id"`
		FhirID    string `json:"fhirId"`
		FullName  string `json:"fullName"`
		Gender    string `json:"gender,omitempty"`
		BirthDate string `json:"birthDate,omitempty"`
	}
	type consentInfo struct {
		ID        string   `json:"id"`
		Status    string   `json:"status"`
		Scope     []string `json:"scope"`
		GrantedAt string   `json:"grantedAt"`
	}
	type enrichedEntry struct {
		Patient patientInfo `json:"patient"`
		Consent consentInfo `json:"consent"`
	}

	entries := make([]enrichedEntry, 0, len(patients))
	for _, p := range patients {
		var fhirData struct {
			Name      []struct{ Text string `json:"text"` } `json:"name"`
			Gender    string                                 `json:"gender"`
			BirthDate string                                 `json:"birthDate"`
		}
		if len(p.PatientData) > 0 {
			json.Unmarshal(p.PatientData, &fhirData)
		}
		fullName := ""
		if len(fhirData.Name) > 0 {
			fullName = fhirData.Name[0].Text
		}
		var scope []string
		json.Unmarshal(p.Scope, &scope)
		if scope == nil {
			scope = []string{}
		}
		entries = append(entries, enrichedEntry{
			Patient: patientInfo{
				ID: p.PatientID.String(), FhirID: p.FHIRPatientID,
				FullName: fullName, Gender: fhirData.Gender, BirthDate: fhirData.BirthDate,
			},
			Consent: consentInfo{
				ID: p.ConsentID.String(), Status: p.Status, Scope: scope,
				GrantedAt: p.GrantedAt.Format("2006-01-02T15:04:05Z"),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"requests": entries, "total": len(entries)})
}

// BreakGlass handles POST /consent/break-glass
func (h *ConsentHandler) BreakGlass(c *gin.Context) {
	actorID := auth.GetActorID(c).String()
	actorRole := auth.GetActorRole(c)

	if actorRole != "physician" && actorRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Only physicians can use break-glass access"}},
		})
		return
	}

	var req BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
		})
		return
	}

	consent, err := h.engine.RecordBreakGlass(c.Request.Context(), req, actorID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrBreakGlassReason) || errors.Is(err, ErrBreakGlassRateLimit) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, ErrPatientNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"consent":   consent,
		"expiresAt": consent.ExpiresAt,
		"message":   "Emergency access granted. This access has been logged and the patient will be notified.",
	})
}

// GetAccessLog handles GET /consent/access-log
func (h *ConsentHandler) GetAccessLog(c *gin.Context) {
	actorID := auth.GetActorID(c).String()

	// Get patient's FHIR ID
	fhirID, err := h.engine.GetPatientFHIRID(c.Request.Context(), actorID)
	if err != nil || fhirID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "No patient record linked"}},
		})
		return
	}

	entries, err := h.engine.GetAccessLog(c.Request.Context(), fhirID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"patientId": fhirID, "entries": entries, "total": len(entries)})
}
