package clinical

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// DrugCheckerImpl implements the DrugChecker interface.
type DrugCheckerImpl struct {
	repo    DrugCheckerRepository
	fda     *OpenFDAClient
	rxnorm  *RxNormClient
	allergy *AllergyChecker
	db      *sqlx.DB
	redis   *redis.Client
	logger  zerolog.Logger
}

// NewDrugChecker creates a new DrugCheckerImpl.
func NewDrugChecker(
	repo DrugCheckerRepository,
	fda *OpenFDAClient,
	rxnorm *RxNormClient,
	allergy *AllergyChecker,
	db *sqlx.DB,
	redisClient *redis.Client,
	logger zerolog.Logger,
) *DrugCheckerImpl {
	return &DrugCheckerImpl{
		repo:    repo,
		fda:     fda,
		rxnorm:  rxnorm,
		allergy: allergy,
		db:      db,
		redis:   redisClient,
		logger:  logger,
	}
}

// CheckInteractions is the main entry point for drug interaction checking.
func (dc *DrugCheckerImpl) CheckInteractions(ctx context.Context, newRxNormCode, patientFHIRID string) (*DrugCheckResult, error) {
	// Get drug name
	newDrugName, _ := dc.rxnorm.GetDrugName(ctx, newRxNormCode)

	result := &DrugCheckResult{
		NewMedication:   MedicationInfo{RxNormCode: newRxNormCode, Name: newDrugName},
		Interactions:    []DrugInteraction{},
		AllergyConflicts: []AllergyConflict{},
		HighestSeverity: SeverityNone,
		CheckComplete:   true,
	}

	// Step 1: Load patient's active MedicationRequests
	var activeMeds []struct {
		Data json.RawMessage `db:"data"`
	}
	err := dc.db.SelectContext(ctx, &activeMeds,
		`SELECT data FROM fhir_resources
		 WHERE resource_type = 'MedicationRequest'
		   AND patient_ref = $1
		   AND status = 'active'
		   AND deleted_at IS NULL`,
		"Patient/"+patientFHIRID)
	if err != nil {
		dc.logger.Error().Err(err).Msg("failed to load active medications")
		result.CheckComplete = false
		result.CheckError = "failed to load active medications"
		return result, nil
	}

	// Step 2: Check each active medication for interactions
	for _, med := range activeMeds {
		rxCode := extractRxNormFromMedRequest(med.Data)
		if rxCode == "" || rxCode == newRxNormCode {
			continue
		}

		interaction, err := dc.CheckSinglePair(ctx, newRxNormCode, rxCode)
		if err != nil {
			dc.logger.Error().Err(err).Str("drugA", newRxNormCode).Str("drugB", rxCode).Msg("interaction check failed")
			result.CheckComplete = false
			result.CheckError = fmt.Sprintf("interaction check failed for %s: %s", rxCode, err.Error())
			continue
		}

		if interaction != nil && interaction.Severity != SeverityNone {
			result.Interactions = append(result.Interactions, *interaction)
			if SeverityRank(interaction.Severity) > SeverityRank(result.HighestSeverity) {
				result.HighestSeverity = interaction.Severity
			}
			if interaction.Severity == SeverityContraindicated {
				result.HasContraindication = true
			}
		}
	}

	// Step 3: Check allergies
	allergyConflicts, err := dc.allergy.CheckAllergies(ctx, newRxNormCode, newDrugName, patientFHIRID)
	if err != nil {
		dc.logger.Error().Err(err).Msg("allergy check failed")
		result.CheckComplete = false
		result.CheckError = "allergy check failed: " + err.Error()
	} else {
		result.AllergyConflicts = allergyConflicts
		for _, ac := range allergyConflicts {
			if SeverityRank(ac.Severity) > SeverityRank(result.HighestSeverity) {
				result.HighestSeverity = ac.Severity
			}
			if ac.Severity == SeverityContraindicated {
				result.HasContraindication = true
			}
		}
	}

	return result, nil
}

// CheckSinglePair checks one drug pair using cache-aside pattern.
func (dc *DrugCheckerImpl) CheckSinglePair(ctx context.Context, rxNormA, rxNormB string) (*DrugInteraction, error) {
	// Step 1: Check PostgreSQL cache
	cached, err := dc.repo.GetCachedInteraction(ctx, rxNormA, rxNormB)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		dc.logger.Debug().Str("drugA", rxNormA).Str("drugB", rxNormB).Msg("interaction cache hit")
		return cached, nil
	}

	// Step 2: Cache miss — call OpenFDA
	dc.logger.Debug().Str("drugA", rxNormA).Str("drugB", rxNormB).Msg("interaction cache miss, calling OpenFDA")

	nameA, _ := dc.rxnorm.GetDrugName(ctx, rxNormA)
	nameB, _ := dc.rxnorm.GetDrugName(ctx, rxNormB)

	labels, err := dc.fda.GetDrugInteractions(ctx, nameA)
	if err != nil {
		// OpenFDA failure — return unknown severity, don't block
		return &DrugInteraction{
			DrugA:    MedicationInfo{RxNormCode: rxNormA, Name: nameA},
			DrugB:    MedicationInfo{RxNormCode: rxNormB, Name: nameB},
			Severity: SeverityUnknown,
			Description: "Drug interaction check unavailable — OpenFDA API unreachable",
			Source:   "openfda",
			Cached:   false,
		}, nil
	}

	// Step 3: Search labels for the other drug
	description, severity := FindInteractionInLabels(labels, nameB)

	if severity == SeverityNone && description == "" {
		// Try reverse lookup (search drug B's labels for drug A)
		labelsB, errB := dc.fda.GetDrugInteractions(ctx, nameB)
		if errB == nil {
			description, severity = FindInteractionInLabels(labelsB, nameA)
		}
	}

	interaction := &DrugInteraction{
		DrugA:       MedicationInfo{RxNormCode: rxNormA, Name: nameA},
		DrugB:       MedicationInfo{RxNormCode: rxNormB, Name: nameB},
		Severity:    severity,
		Description: description,
		Source:      "openfda",
		Cached:      false,
	}

	// Step 4: Cache the result permanently
	if err := dc.repo.CacheInteraction(ctx, interaction); err != nil {
		dc.logger.Error().Err(err).Msg("failed to cache interaction")
	}

	return interaction, nil
}

// AcknowledgeInteraction records a physician's acknowledgment.
func (dc *DrugCheckerImpl) AcknowledgeInteraction(ctx context.Context, req AcknowledgmentRequest) (*Acknowledgment, error) {
	if len(req.PhysicianReason) < 20 {
		return nil, fmt.Errorf("acknowledgment reason must be at least 20 characters")
	}

	ackID := uuid.New()
	expiresAt := time.Now().Add(30 * time.Minute)

	// Store acknowledgment in Redis with 30-minute TTL
	ackKey := fmt.Sprintf("drug:ack:%s:%s:%s", req.PhysicianID, req.PatientFHIRID, req.NewRxNormCode)
	ackData := map[string]interface{}{
		"id":               ackID.String(),
		"physician_id":     req.PhysicianID,
		"patient_fhir_id":  req.PatientFHIRID,
		"new_rxnorm_code":  req.NewRxNormCode,
		"conflicting_codes": req.ConflictingCodes,
		"reason":           req.PhysicianReason,
		"expires_at":       expiresAt.Format(time.RFC3339),
	}

	data, _ := json.Marshal(ackData)
	dc.redis.Set(ctx, ackKey, string(data), 30*time.Minute)

	dc.logger.Info().
		Str("physician", req.PhysicianID).
		Str("patient", req.PatientFHIRID).
		Str("medication", req.NewRxNormCode).
		Msg("interaction acknowledged")

	return &Acknowledgment{
		ID:        ackID,
		ExpiresAt: expiresAt,
	}, nil
}

// HasValidAcknowledgment checks if a valid (non-expired) acknowledgment exists.
func (dc *DrugCheckerImpl) HasValidAcknowledgment(ctx context.Context, physicianID, patientFHIRID, rxNormCode string) bool {
	ackKey := fmt.Sprintf("drug:ack:%s:%s:%s", physicianID, patientFHIRID, rxNormCode)
	result, err := dc.redis.Get(ctx, ackKey).Result()
	return err == nil && result != ""
}

// extractRxNormFromMedRequest extracts the RxNorm code from a MedicationRequest FHIR resource.
func extractRxNormFromMedRequest(data json.RawMessage) string {
	var med struct {
		MedicationCodeableConcept struct {
			Coding []struct {
				System string `json:"system"`
				Code   string `json:"code"`
			} `json:"coding"`
		} `json:"medicationCodeableConcept"`
	}
	if err := json.Unmarshal(data, &med); err != nil {
		return ""
	}
	for _, coding := range med.MedicationCodeableConcept.Coding {
		if coding.Code != "" {
			return coding.Code
		}
	}
	return ""
}
