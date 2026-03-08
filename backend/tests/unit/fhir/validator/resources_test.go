package validator_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Siddharthk17/MediLink/internal/fhir/validator"
)

// Practitioner

func TestValidatePractitioner_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Practitioner","name":[{"family":"Mehta","given":["Priya"]}]}`)

	err := v.ValidatePractitioner(data, "")
	assert.Nil(t, err)
}

func TestValidatePractitioner_MissingName(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Practitioner"}`)

	err := v.ValidatePractitioner(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePractitioner_MissingFamily(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Practitioner","name":[{"given":["Priya"]}]}`)

	err := v.ValidatePractitioner(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// Organization

func TestValidateOrganization_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Organization","name":"Apollo Hospital","type":[{"coding":[{"code":"hosp"}]}]}`)

	err := v.ValidateOrganization(data, "")
	assert.Nil(t, err)
}

func TestValidateOrganization_MissingName(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Organization"}`)

	err := v.ValidateOrganization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateOrganization_EmptyName(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Organization","name":""}`)

	err := v.ValidateOrganization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateOrganization_InvalidType(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Organization","name":"Test Org","type":[{"coding":[{"code":"invalid-type"}]}]}`)

	err := v.ValidateOrganization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// Encounter

func TestValidateEncounter_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Encounter","status":"planned","class":{"code":"AMB"},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateEncounter(data, "")
	assert.Nil(t, err)
}

func TestValidateEncounter_MissingStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Encounter","class":{"code":"AMB"},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateEncounter(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateEncounter_InvalidStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Encounter","status":"bogus","class":{"code":"AMB"},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateEncounter(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateEncounter_MissingSubject(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Encounter","status":"planned","class":{"code":"AMB"}}`)

	err := v.ValidateEncounter(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateEncounter_MissingClass(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Encounter","status":"planned","subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateEncounter(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateEncounter_PeriodEndBeforeStart(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType":"Encounter",
		"status":"planned",
		"class":{"code":"AMB"},
		"subject":{"reference":"Patient/test-id"},
		"period":{"start":"2024-06-15T10:00:00+05:30","end":"2024-06-14T10:00:00+05:30"}
	}`)

	err := v.ValidateEncounter(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// Condition

func TestValidateCondition_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Condition","clinicalStatus":{"coding":[{"code":"active"}]},"code":{"coding":[{"system":"http://hl7.org/fhir/sid/icd-10","code":"E11.9"}]},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateCondition(data, "")
	assert.Nil(t, err)
}

func TestValidateCondition_MissingCode(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Condition","clinicalStatus":{"coding":[{"code":"active"}]},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateCondition(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateCondition_InvalidICD10(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Condition","clinicalStatus":{"coding":[{"code":"active"}]},"code":{"coding":[{"system":"http://hl7.org/fhir/sid/icd-10","code":"INVALID"}]},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateCondition(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateCondition_ActiveWithAbatement(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Condition","clinicalStatus":{"coding":[{"code":"active"}]},"code":{"coding":[{"system":"http://hl7.org/fhir/sid/icd-10","code":"E11.9"}]},"subject":{"reference":"Patient/test-id"},"abatementDateTime":"2024-01-10T00:00:00+05:30"}`)

	err := v.ValidateCondition(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// MedicationRequest

func TestValidateMedicationRequest_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"MedicationRequest","status":"active","intent":"order","medicationCodeableConcept":{"coding":[{"code":"860975"}]},"subject":{"reference":"Patient/test-id"},"authoredOn":"2024-01-15T10:00:00+05:30"}`)

	err := v.ValidateMedicationRequest(data, "")
	assert.Nil(t, err)
}

func TestValidateMedicationRequest_MissingStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"MedicationRequest","intent":"order","medicationCodeableConcept":{"coding":[{"code":"860975"}]},"subject":{"reference":"Patient/test-id"},"authoredOn":"2024-01-15T10:00:00+05:30"}`)

	err := v.ValidateMedicationRequest(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateMedicationRequest_MissingMedication(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"MedicationRequest","status":"active","intent":"order","subject":{"reference":"Patient/test-id"},"authoredOn":"2024-01-15T10:00:00+05:30"}`)

	err := v.ValidateMedicationRequest(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateMedicationRequest_MissingIntent(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"MedicationRequest","status":"active","medicationCodeableConcept":{"coding":[{"code":"860975"}]},"subject":{"reference":"Patient/test-id"},"authoredOn":"2024-01-15T10:00:00+05:30"}`)

	err := v.ValidateMedicationRequest(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// Observation

func TestValidateObservation_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueQuantity":{"value":7.2,"unit":"%"}}`)

	err := v.ValidateObservation(data, "")
	assert.Nil(t, err)
}

func TestValidateObservation_MissingStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueQuantity":{"value":7.2,"unit":"%"}}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_MissingCode(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueString":"normal"}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_MissingEffective(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"valueQuantity":{"value":7.2,"unit":"%"}}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_InvalidInterpretation(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueQuantity":{"value":7.2,"unit":"%"},"interpretation":[{"coding":[{"code":"INVALID"}]}]}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_ValidWithDataAbsentReason(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","dataAbsentReason":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/data-absent-reason","code":"unknown"}]}}`)

	err := v.ValidateObservation(data, "")
	assert.Nil(t, err)
}

func TestValidateObservation_MissingValueAndDataAbsentReason(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30"}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_BothValueAndDataAbsentReason(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueQuantity":{"value":7.2,"unit":"%"},"dataAbsentReason":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/data-absent-reason","code":"unknown"}]}}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateObservation_MultipleValues(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Observation","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"4548-4"}]},"subject":{"reference":"Patient/test-id"},"effectiveDateTime":"2024-01-15T08:00:00+05:30","valueQuantity":{"value":7.2,"unit":"%"},"valueString":"normal"}`)

	err := v.ValidateObservation(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// DiagnosticReport

func TestValidateDiagnosticReport_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"DiagnosticReport","status":"final","code":{"coding":[{"code":"58410-2"}]},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateDiagnosticReport(data, "")
	assert.Nil(t, err)
}

func TestValidateDiagnosticReport_MissingStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"DiagnosticReport","code":{"coding":[{"code":"58410-2"}]},"subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateDiagnosticReport(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateDiagnosticReport_MissingCode(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"DiagnosticReport","status":"final","subject":{"reference":"Patient/test-id"}}`)

	err := v.ValidateDiagnosticReport(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateDiagnosticReport_InvalidResultRef(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"DiagnosticReport","status":"final","code":{"coding":[{"code":"58410-2"}]},"subject":{"reference":"Patient/test-id"},"result":[{"reference":"InvalidRef/123"}]}`)

	err := v.ValidateDiagnosticReport(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// AllergyIntolerance

func TestValidateAllergyIntolerance_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"AllergyIntolerance","clinicalStatus":{"coding":[{"code":"active"}]},"verificationStatus":{"coding":[{"code":"confirmed"}]},"code":{"coding":[{"code":"7980"}]},"patient":{"reference":"Patient/test-id"}}`)

	err := v.ValidateAllergyIntolerance(data, "")
	assert.Nil(t, err)
}

func TestValidateAllergyIntolerance_MissingPatient(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"AllergyIntolerance","clinicalStatus":{"coding":[{"code":"active"}]},"verificationStatus":{"coding":[{"code":"confirmed"}]},"code":{"coding":[{"code":"7980"}]}}`)

	err := v.ValidateAllergyIntolerance(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateAllergyIntolerance_ActiveWithRefuted(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"AllergyIntolerance","clinicalStatus":{"coding":[{"code":"active"}]},"verificationStatus":{"coding":[{"code":"refuted"}]},"code":{"coding":[{"code":"7980"}]},"patient":{"reference":"Patient/test-id"}}`)

	err := v.ValidateAllergyIntolerance(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateAllergyIntolerance_InvalidCategory(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"AllergyIntolerance","clinicalStatus":{"coding":[{"code":"active"}]},"verificationStatus":{"coding":[{"code":"confirmed"}]},"code":{"coding":[{"code":"7980"}]},"patient":{"reference":"Patient/test-id"},"category":["invalid-cat"]}`)

	err := v.ValidateAllergyIntolerance(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

// Immunization

func TestValidateImmunization_Valid(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Immunization","status":"completed","vaccineCode":{"coding":[{"code":"208"}]},"patient":{"reference":"Patient/test-id"},"occurrenceDateTime":"2021-04-15T10:00:00+05:30"}`)

	err := v.ValidateImmunization(data, "")
	assert.Nil(t, err)
}

func TestValidateImmunization_MissingStatus(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Immunization","vaccineCode":{"coding":[{"code":"208"}]},"patient":{"reference":"Patient/test-id"},"occurrenceDateTime":"2021-04-15T10:00:00+05:30"}`)

	err := v.ValidateImmunization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateImmunization_MissingVaccineCode(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Immunization","status":"completed","patient":{"reference":"Patient/test-id"},"occurrenceDateTime":"2021-04-15T10:00:00+05:30"}`)

	err := v.ValidateImmunization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateImmunization_FutureOccurrence(t *testing.T) {
	v := validator.NewFHIRValidator()
	futureDate := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	data := json.RawMessage(fmt.Sprintf(`{"resourceType":"Immunization","status":"completed","vaccineCode":{"coding":[{"code":"208"}]},"patient":{"reference":"Patient/test-id"},"occurrenceDateTime":"%s"}`, futureDate))

	err := v.ValidateImmunization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidateImmunization_MissingOccurrenceWhenCompleted(t *testing.T) {
	v := validator.NewFHIRValidator()
	data := json.RawMessage(`{"resourceType":"Immunization","status":"completed","vaccineCode":{"coding":[{"code":"208"}]},"patient":{"reference":"Patient/test-id"}}`)

	err := v.ValidateImmunization(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}
