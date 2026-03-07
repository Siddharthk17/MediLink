-- Timeline query optimization
CREATE INDEX idx_fhir_timeline
    ON fhir_resources (patient_ref, created_at DESC)
    WHERE deleted_at IS NULL;

-- Observation LOINC code lookup for lab trends
CREATE INDEX idx_observation_loinc
    ON fhir_resources ((data->'code'->'coding'->0->>'code'))
    WHERE resource_type = 'Observation' AND deleted_at IS NULL;

-- Encounter date for timeline ordering
CREATE INDEX idx_encounter_period_start
    ON fhir_resources (((data->'period'->>'start')::timestamptz) DESC)
    WHERE resource_type = 'Encounter' AND deleted_at IS NULL;

-- MedicationRequest status for drug interaction checker
CREATE INDEX idx_medication_request_status
    ON fhir_resources ((data->>'status'))
    WHERE resource_type = 'MedicationRequest' AND deleted_at IS NULL;

-- AllergyIntolerance for drug checker
CREATE INDEX idx_allergy_patient_status
    ON fhir_resources (patient_ref, (data->'clinicalStatus'->'coding'->0->>'code'))
    WHERE resource_type = 'AllergyIntolerance' AND deleted_at IS NULL;
