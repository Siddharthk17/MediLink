package anonymization_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/anonymization"
)

// fhirPatient returns a realistic FHIR Patient resource with PII fields populated.
func fhirPatient() map[string]interface{} {
	return map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-abc-123",
		"meta":         map[string]interface{}{"versionId": "1"},
		"name": []interface{}{
			map[string]interface{}{
				"family": "Smith",
				"given":  []interface{}{"John", "Michael"},
			},
		},
		"telecom": []interface{}{
			map[string]interface{}{"system": "phone", "value": "+1-555-0100"},
			map[string]interface{}{"system": "email", "value": "john@example.com"},
		},
		"identifier": []interface{}{
			map[string]interface{}{
				"system": "http://hospital.org/mrn",
				"value":  "MRN-12345",
			},
		},
		"gender":    "male",
		"birthDate": "1995-03-15",
		"address": []interface{}{
			map[string]interface{}{"city": "Springfield", "state": "IL"},
		},
	}
}

func TestAnonymizePatient_RemovesName(t *testing.T) {
	result := anonymization.AnonymizePatient(fhirPatient(), "test-salt")
	_, hasName := result["name"]
	assert.False(t, hasName, "name must be removed from anonymized patient")
}

func TestAnonymizePatient_RemovesTelecom(t *testing.T) {
	result := anonymization.AnonymizePatient(fhirPatient(), "test-salt")
	_, hasTelecom := result["telecom"]
	assert.False(t, hasTelecom, "telecom must be removed from anonymized patient")
}

func TestAnonymizePatient_RemovesIdentifiers(t *testing.T) {
	result := anonymization.AnonymizePatient(fhirPatient(), "test-salt")
	_, hasIdentifier := result["identifier"]
	assert.False(t, hasIdentifier, "identifier must be removed from anonymized patient")
}

func TestAnonymizePatient_GeneralizesBirthDate(t *testing.T) {
	result := anonymization.AnonymizePatient(fhirPatient(), "test-salt")
	assert.Equal(t, "1995", result["birthDate"], "birthDate must be generalized to year only")
}

func TestAnonymizePatient_KeepsGender(t *testing.T) {
	result := anonymization.AnonymizePatient(fhirPatient(), "test-salt")
	assert.Equal(t, "male", result["gender"], "gender must be preserved for k-anonymity grouping")
}

func TestAnonymizeEncounter_RemovesParticipant(t *testing.T) {
	encounter := map[string]interface{}{
		"resourceType": "Encounter",
		"id":           "enc-001",
		"status":       "finished",
		"class":        map[string]interface{}{"code": "AMB"},
		"subject":      map[string]interface{}{"reference": "Patient/patient-abc-123"},
		"participant": []interface{}{
			map[string]interface{}{
				"type": []interface{}{
					map[string]interface{}{"text": "primary performer"},
				},
				"individual": map[string]interface{}{
					"reference": "Practitioner/dr-smith-456",
					"display":   "Dr. Smith",
				},
			},
		},
		"period": map[string]interface{}{
			"start": "2024-01-15T10:00:00+05:30",
			"end":   "2024-01-15T11:00:00+05:30",
		},
	}

	result := anonymization.AnonymizeEncounter(encounter, "test-salt")

	participants, ok := result["participant"].([]interface{})
	require.True(t, ok, "participant array should still exist")
	require.Len(t, participants, 1)

	p, ok := participants[0].(map[string]interface{})
	require.True(t, ok)
	_, hasIndividual := p["individual"]
	assert.False(t, hasIndividual, "participant.individual must be removed")
}

func TestAnonymizeEncounter_GeneralizesPeriod(t *testing.T) {
	encounter := map[string]interface{}{
		"resourceType": "Encounter",
		"id":           "enc-002",
		"status":       "finished",
		"subject":      map[string]interface{}{"reference": "Patient/p1"},
		"period": map[string]interface{}{
			"start": "2024-01-15T10:00:00+05:30",
			"end":   "2024-01-15T11:30:00+05:30",
		},
	}

	result := anonymization.AnonymizeEncounter(encounter, "test-salt")

	period, ok := result["period"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2024-01", period["start"], "period.start must be generalized to YYYY-MM")
	assert.Equal(t, "2024-01", period["end"], "period.end must be generalized to YYYY-MM")
}

func TestAnonymizeObservation_KeepsLOINC(t *testing.T) {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"id":           "obs-001",
		"status":       "final",
		"code": map[string]interface{}{
			"coding": []interface{}{
				map[string]interface{}{
					"system":  "http://loinc.org",
					"code":    "29463-7",
					"display": "Body Weight",
				},
			},
		},
		"subject":           map[string]interface{}{"reference": "Patient/p1"},
		"performer":         []interface{}{map[string]interface{}{"reference": "Practitioner/dr1"}},
		"effectiveDateTime": "2024-01-15T10:00:00Z",
		"valueQuantity": map[string]interface{}{
			"value":  float64(70),
			"unit":   "kg",
			"system": "http://unitsofmeasure.org",
			"code":   "kg",
		},
	}

	result := anonymization.AnonymizeObservation(obs, "test-salt")

	code, ok := result["code"].(map[string]interface{})
	require.True(t, ok, "code must be preserved")
	codings := code["coding"].([]interface{})
	first := codings[0].(map[string]interface{})
	assert.Equal(t, "http://loinc.org", first["system"])
	assert.Equal(t, "29463-7", first["code"])
	assert.Equal(t, "Body Weight", first["display"])

	_, hasPerformer := result["performer"]
	assert.False(t, hasPerformer, "performer must be removed")
}

func TestAnonymizeObservation_GeneralizesDate(t *testing.T) {
	obs := map[string]interface{}{
		"resourceType":      "Observation",
		"id":                "obs-002",
		"status":            "final",
		"code":              map[string]interface{}{"text": "BP"},
		"subject":           map[string]interface{}{"reference": "Patient/p1"},
		"effectiveDateTime": "2024-06-20T14:30:00Z",
	}

	result := anonymization.AnonymizeObservation(obs, "test-salt")
	assert.Equal(t, "2024-06", result["effectiveDateTime"],
		"effectiveDateTime must be generalized to YYYY-MM")
}

func TestAnonymizeDiagnosticReport_RemovesPDF(t *testing.T) {
	report := map[string]interface{}{
		"resourceType": "DiagnosticReport",
		"id":           "dr-001",
		"status":       "final",
		"code":         map[string]interface{}{"text": "CBC"},
		"subject":      map[string]interface{}{"reference": "Patient/p1"},
		"presentedForm": []interface{}{
			map[string]interface{}{
				"contentType": "application/pdf",
				"data":        "JVBERi0xLjQK...",
			},
		},
		"conclusion": "Normal CBC results",
	}

	result := anonymization.AnonymizeDiagnosticReport(report, "test-salt")

	_, hasPDF := result["presentedForm"]
	assert.False(t, hasPDF, "presentedForm (PDF) must never be exported")
	_, hasConclusion := result["conclusion"]
	assert.False(t, hasConclusion, "conclusion must be removed")
}

func TestAnonymizeMedRequest_KeepsRxNorm(t *testing.T) {
	medReq := map[string]interface{}{
		"resourceType": "MedicationRequest",
		"id":           "mr-001",
		"status":       "active",
		"intent":       "order",
		"medicationCodeableConcept": map[string]interface{}{
			"coding": []interface{}{
				map[string]interface{}{
					"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
					"code":    "861004",
					"display": "Metformin 500 MG Oral Tablet",
				},
			},
		},
		"subject":   map[string]interface{}{"reference": "Patient/p1"},
		"requester": map[string]interface{}{"reference": "Practitioner/dr1"},
	}

	result := anonymization.AnonymizeMedicationRequest(medReq, "test-salt")

	med, ok := result["medicationCodeableConcept"].(map[string]interface{})
	require.True(t, ok, "medicationCodeableConcept must be preserved")
	codings := med["coding"].([]interface{})
	first := codings[0].(map[string]interface{})
	assert.Equal(t, "http://www.nlm.nih.gov/research/umls/rxnorm", first["system"])
	assert.Equal(t, "861004", first["code"])

	_, hasRequester := result["requester"]
	assert.False(t, hasRequester, "requester must be removed")
}

func TestAnonymizeDoesNotModifyOriginal(t *testing.T) {
	original := fhirPatient()

	_ = anonymization.AnonymizePatient(original, "test-salt")

	assert.Equal(t, "patient-abc-123", original["id"], "original id must not change")
	assert.Equal(t, "1995-03-15", original["birthDate"], "original birthDate must not change")
	assert.NotNil(t, original["name"], "original name must not be removed")
	assert.NotNil(t, original["telecom"], "original telecom must not be removed")
	assert.NotNil(t, original["identifier"], "original identifier must not be removed")
	assert.NotNil(t, original["address"], "original address must not be removed")
}
