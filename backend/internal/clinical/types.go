// Package clinical implements the drug interaction checker and allergy conflict detection.
package clinical

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InteractionSeverity classifies the severity of a drug interaction.
type InteractionSeverity string

const (
	SeverityContraindicated InteractionSeverity = "contraindicated"
	SeverityMajor           InteractionSeverity = "major"
	SeverityModerate        InteractionSeverity = "moderate"
	SeverityMinor           InteractionSeverity = "minor"
	SeverityNone            InteractionSeverity = "none"
	SeverityUnknown         InteractionSeverity = "unknown"
)

// SeverityRank returns a numeric rank for comparing severities (higher = more severe).
func SeverityRank(s InteractionSeverity) int {
	switch s {
	case SeverityContraindicated:
		return 5
	case SeverityMajor:
		return 4
	case SeverityModerate:
		return 3
	case SeverityMinor:
		return 2
	case SeverityNone:
		return 0
	case SeverityUnknown:
		return 1
	default:
		return 0
	}
}

// DrugInteraction represents a drug-drug interaction result.
type DrugInteraction struct {
	DrugA          MedicationInfo      `json:"drugA"`
	DrugB          MedicationInfo      `json:"drugB"`
	Severity       InteractionSeverity `json:"severity"`
	Description    string              `json:"description"`
	Mechanism      string              `json:"mechanism,omitempty"`
	ClinicalEffect string             `json:"clinicalEffect,omitempty"`
	Management     string              `json:"management,omitempty"`
	Source         string              `json:"source"`
	Cached         bool                `json:"cached"`
}

// MedicationInfo identifies a medication by its RxNorm code and name.
type MedicationInfo struct {
	RxNormCode string `json:"rxnormCode"`
	Name       string `json:"name"`
}

// AllergyConflict represents a conflict between a new medication and a known allergy.
type AllergyConflict struct {
	Allergen      MedicationInfo      `json:"allergen"`
	NewMedication MedicationInfo      `json:"newMedication"`
	Severity      InteractionSeverity `json:"severity"`
	Mechanism     string              `json:"mechanism"`
	DrugClass     string              `json:"drugClass,omitempty"`
	Reaction      string              `json:"reaction,omitempty"`
}

// DrugCheckResult is the complete result of a drug interaction check.
type DrugCheckResult struct {
	NewMedication    MedicationInfo      `json:"newMedication"`
	Interactions     []DrugInteraction   `json:"interactions"`
	AllergyConflicts []AllergyConflict   `json:"allergyConflicts"`
	HighestSeverity  InteractionSeverity `json:"highestSeverity"`
	HasContraindication bool             `json:"hasContraindication"`
	CheckComplete    bool                `json:"checkComplete"`
	CheckError       string              `json:"checkError,omitempty"`
}

// DrugChecker is the main interface for drug interaction checking.
type DrugChecker interface {
	CheckInteractions(ctx context.Context, newRxNormCode, patientFHIRID string) (*DrugCheckResult, error)
	CheckSinglePair(ctx context.Context, rxNormA, rxNormB string) (*DrugInteraction, error)
	AcknowledgeInteraction(ctx context.Context, req AcknowledgmentRequest) (*Acknowledgment, error)
}

// AcknowledgmentRequest represents a physician's acknowledgment of a drug interaction.
type AcknowledgmentRequest struct {
	PatientFHIRID    string   `json:"patientId" binding:"required"`
	NewRxNormCode    string   `json:"newMedication" binding:"required"`
	ConflictingCodes []string `json:"conflictingMedications" binding:"required"`
	PhysicianReason  string   `json:"reason" binding:"required,min=20"`
	PhysicianID      string   // set from JWT, never from client
}

// Acknowledgment represents a stored physician acknowledgment with an expiry.
type Acknowledgment struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// DrugCheckRequest is the HTTP request body for the drug check endpoint.
type DrugCheckRequest struct {
	PatientID     string         `json:"patientId" binding:"required"`
	NewMedication MedicationInfo `json:"newMedication" binding:"required"`
}

// NormalizeDrugPair ensures alphabetical ordering of drug pair for cache consistency.
func NormalizeDrugPair(rxNormA, rxNormB string) (string, string) {
	if rxNormA < rxNormB {
		return rxNormA, rxNormB
	}
	return rxNormB, rxNormA
}
