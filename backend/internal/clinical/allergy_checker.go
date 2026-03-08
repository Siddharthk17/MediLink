package clinical

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// AllergyChecker checks for allergy conflicts with a new medication.
type AllergyChecker struct {
	db       *sqlx.DB
	rxnorm   *RxNormClient
	drugRepo DrugCheckerRepository
	logger   zerolog.Logger
}

// NewAllergyChecker creates a new AllergyChecker.
func NewAllergyChecker(db *sqlx.DB, rxnorm *RxNormClient, drugRepo DrugCheckerRepository, logger zerolog.Logger) *AllergyChecker {
	return &AllergyChecker{
		db:       db,
		rxnorm:   rxnorm,
		drugRepo: drugRepo,
		logger:   logger,
	}
}

// CheckAllergies checks if a new medication conflicts with the patient's known allergies.
func (ac *AllergyChecker) CheckAllergies(ctx context.Context, newRxNormCode, newDrugName, patientFHIRID string) ([]AllergyConflict, error) {
	// Load active AllergyIntolerances for the patient
	var resources []struct {
		Data json.RawMessage `db:"data"`
	}
	err := ac.db.SelectContext(ctx, &resources,
		`SELECT data FROM fhir_resources
		 WHERE resource_type = 'AllergyIntolerance'
		   AND patient_ref = $1
		   AND deleted_at IS NULL
		   AND (data->'clinicalStatus'->'coding'->0->>'code' = 'active'
		        OR data->'clinicalStatus'->'coding'->0->>'code' IS NULL)`,
		"Patient/"+patientFHIRID)
	if err != nil {
		return nil, err
	}

	var conflicts []AllergyConflict
	newDrugNameLower := strings.ToLower(newDrugName)

	for _, resource := range resources {
		var allergy struct {
			Code struct {
				Coding []struct {
					Code    string `json:"code"`
					Display string `json:"display"`
				} `json:"coding"`
				Text string `json:"text"`
			} `json:"code"`
			Criticality string `json:"criticality"`
			Category    []string `json:"category"`
			Reaction    []struct {
				Manifestation []struct {
					Text string `json:"text"`
				} `json:"manifestation"`
				Severity string `json:"severity"`
			} `json:"reaction"`
		}
		if err := json.Unmarshal(resource.Data, &allergy); err != nil {
			ac.logger.Warn().Err(err).Msg("failed to parse AllergyIntolerance")
			continue
		}

		// Check if this is a medication allergy
		isMedAllergy := false
		for _, cat := range allergy.Category {
			if cat == "medication" {
				isMedAllergy = true
				break
			}
		}
		if !isMedAllergy && len(allergy.Category) > 0 {
			continue
		}

		allergenName := allergy.Code.Text
		allergenCode := ""
		if len(allergy.Code.Coding) > 0 {
			allergenCode = allergy.Code.Coding[0].Code
			if allergenName == "" {
				allergenName = allergy.Code.Coding[0].Display
			}
		}
		allergenNameLower := strings.ToLower(allergenName)

		// Determine severity from criticality
		severity := SeverityMajor
		if allergy.Criticality == "high" {
			severity = SeverityContraindicated
		} else if allergy.Criticality == "low" {
			severity = SeverityModerate
		}

		// Get known reaction text
		reaction := ""
		if len(allergy.Reaction) > 0 && len(allergy.Reaction[0].Manifestation) > 0 {
			reaction = allergy.Reaction[0].Manifestation[0].Text
		}

		// Direct match: allergen name matches drug name
		if strings.Contains(newDrugNameLower, allergenNameLower) ||
			strings.Contains(allergenNameLower, newDrugNameLower) ||
			(allergenCode != "" && allergenCode == newRxNormCode) {
			conflicts = append(conflicts, AllergyConflict{
				Allergen:      MedicationInfo{RxNormCode: allergenCode, Name: allergenName},
				NewMedication: MedicationInfo{RxNormCode: newRxNormCode, Name: newDrugName},
				Severity:      severity,
				Mechanism:     "direct",
				Reaction:      reaction,
			})
			continue
		}

		// Cross-reactivity check: same drug class
		if allergenCode != "" {
			allergenClass, _ := ac.drugRepo.GetDrugClass(ctx, allergenCode)
			if allergenClass == "" {
				// Try RxNorm API
				classes, _ := ac.rxnorm.GetDrugClass(ctx, allergenCode)
				if len(classes) > 0 {
					allergenClass = classes[0]
					_ = ac.drugRepo.StoreDrugClass(ctx, allergenCode, allergenName, allergenClass)
				}
			}

			if allergenClass != "" {
				newDrugClass, _ := ac.drugRepo.GetDrugClass(ctx, newRxNormCode)
				if newDrugClass == "" {
					classes, _ := ac.rxnorm.GetDrugClass(ctx, newRxNormCode)
					if len(classes) > 0 {
						newDrugClass = classes[0]
						_ = ac.drugRepo.StoreDrugClass(ctx, newRxNormCode, newDrugName, newDrugClass)
					}
				}

				if newDrugClass != "" && strings.EqualFold(allergenClass, newDrugClass) {
					conflicts = append(conflicts, AllergyConflict{
						Allergen:      MedicationInfo{RxNormCode: allergenCode, Name: allergenName},
						NewMedication: MedicationInfo{RxNormCode: newRxNormCode, Name: newDrugName},
						Severity:      severity,
						Mechanism:     "cross-reactivity",
						DrugClass:     allergenClass,
						Reaction:      reaction,
					})
				}
			}
		}
	}

	return conflicts, nil
}
