package consent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ConsentHandler struct {
	engine ConsentEngine
}

func NewConsentHandler(engine ConsentEngine) *ConsentHandler {
	return &ConsentHandler{engine: engine}
}

// GrantConsent handles POST /consent/grant
func (h *ConsentHandler) GrantConsent(c *gin.Context) {
	actorID := c.GetString("actor_id")
	actorRole := c.GetString("actor_role")

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
		if strings.Contains(err.Error(), "cannot grant") || strings.Contains(err.Error(), "invalid scope") {
			status = http.StatusBadRequest
		}
		if strings.Contains(err.Error(), "not found") {
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
	actorID := c.GetString("actor_id")
	actorRole := c.GetString("actor_role")

	err := h.engine.RevokeConsent(c.Request.Context(), consentID, actorID, actorRole)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		if strings.Contains(err.Error(), "only the patient") {
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
	actorID := c.GetString("actor_id")

	consents, err := h.engine.GetPatientConsents(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"consents": consents, "total": len(consents)})
}

// GetMyPatients handles GET /consent/my-patients
func (h *ConsentHandler) GetMyPatients(c *gin.Context) {
	actorID := c.GetString("actor_id")
	actorRole := c.GetString("actor_role")

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

	c.JSON(http.StatusOK, gin.H{"patients": patients, "total": len(patients)})
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

	consent, err := h.engine.(*consentEngine).repo.GetByID(c.Request.Context(), id)
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

// BreakGlass handles POST /consent/break-glass
func (h *ConsentHandler) BreakGlass(c *gin.Context) {
	actorID := c.GetString("actor_id")
	actorRole := c.GetString("actor_role")

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
		if strings.Contains(err.Error(), "at least 20") {
			status = http.StatusBadRequest
		}
		if strings.Contains(err.Error(), "not found") {
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
	actorID := c.GetString("actor_id")

	// Get patient's FHIR ID
	fhirID, err := h.engine.GetPatientFHIRID(c.Request.Context(), actorID)
	if err != nil || fhirID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "No patient record linked"}},
		})
		return
	}

	// Query audit logs for this patient's resources
	// Return access log showing who accessed what
	c.JSON(http.StatusOK, gin.H{
		"patientId": fhirID,
		"message":   "Access log query",
	})
}
