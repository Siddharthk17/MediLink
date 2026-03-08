package clinical

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/auth"
)

// ClinicalHandler handles HTTP requests for the drug interaction checker.
type ClinicalHandler struct {
	checker     *DrugCheckerImpl
	auditLogger audit.AuditLogger
	logger      zerolog.Logger
}

// NewClinicalHandler creates a new ClinicalHandler.
func NewClinicalHandler(checker *DrugCheckerImpl, auditLogger audit.AuditLogger, logger zerolog.Logger) *ClinicalHandler {
	return &ClinicalHandler{
		checker:     checker,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// CheckDrugInteractions handles POST /clinical/drug-check.
func (h *ClinicalHandler) CheckDrugInteractions(c *gin.Context) {
	var req DrugCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
		})
		return
	}

	actorID := auth.GetActorID(c)

	result, err := h.checker.CheckInteractions(c.Request.Context(), req.NewMedication.RxNormCode, req.PatientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	// Set drug name if provided in request
	if req.NewMedication.Name != "" {
		result.NewMedication.Name = req.NewMedication.Name
	}

	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "MedicationRequest",
		Action:       "drug_check",
		PatientRef:   "Patient/" + req.PatientID,
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	c.JSON(http.StatusOK, result)
}

// AcknowledgeInteraction handles POST /clinical/drug-check/acknowledge.
func (h *ClinicalHandler) AcknowledgeInteraction(c *gin.Context) {
	var req AcknowledgmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
		})
		return
	}

	actorID := auth.GetActorID(c)
	req.PhysicianID = actorID.String()

	ack, err := h.checker.AcknowledgeInteraction(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
		})
		return
	}

	h.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &actorID,
		ResourceType: "MedicationRequest",
		Action:       "interaction_acknowledged",
		PatientRef:   "Patient/" + req.PatientFHIRID,
		Purpose:      "treatment",
		Success:      true,
		StatusCode:   http.StatusOK,
	})

	c.JSON(http.StatusOK, ack)
}

// GetCheckHistory handles GET /clinical/drug-check/history/:patientId.
func (h *ClinicalHandler) GetCheckHistory(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": "patientId is required"}},
		})
		return
	}

	limit := 30
	if l := c.Query("_count"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	interactions, err := h.checker.repo.GetAllInteractionsForDrug(c.Request.Context(), patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{"severity": "error", "code": "exception", "diagnostics": err.Error()}},
		})
		return
	}

	if len(interactions) > limit {
		interactions = interactions[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"patientId":    patientID,
		"interactions": interactions,
		"total":        len(interactions),
	})
}
