package validator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePatient_Valid(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Sharma", "given": ["Rahul"]}],
		"gender": "male",
		"birthDate": "1995-03-15"
	}`)

	err := v.ValidatePatient(data, "")
	assert.Nil(t, err)
}

func TestValidatePatient_NoName(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"gender": "male"
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
	assert.Contains(t, err.Error(), "name")
}

func TestValidatePatient_EmptyName(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"use": "official"}]
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_FutureDOB(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"birthDate": "2099-01-01"
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
	foundFutureDOB := false
	for _, issue := range err.OperationOutcome.Issue {
		if issue.Details != nil && issue.Details.Text == "Patient.birthDate: cannot be in the future" {
			foundFutureDOB = true
		}
	}
	assert.True(t, foundFutureDOB, "expected future birthDate error")
}

func TestValidatePatient_InvalidGender(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"gender": "invalid"
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
	foundGenderErr := false
	for _, issue := range err.OperationOutcome.Issue {
		if issue.Details != nil && issue.Details.Text == "Patient.gender: must be male, female, other, or unknown" {
			foundGenderErr = true
		}
	}
	assert.True(t, foundGenderErr, "expected gender error")
}

func TestValidatePatient_InvalidID(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"id": "wrong-id",
		"name": [{"family": "Test"}]
	}`)

	err := v.ValidatePatient(data, "correct-id")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_IDMatchesURL(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"id": "same-id",
		"name": [{"family": "Test"}]
	}`)

	err := v.ValidatePatient(data, "same-id")
	assert.Nil(t, err)
}

func TestValidatePatient_InvalidBirthDateFormat(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"birthDate": "15-03-1995"
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_InvalidResourceType(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Observation",
		"name": [{"family": "Test"}]
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_IdentifierMissingSystem(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"identifier": [{"value": "123"}]
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_TelecomMissingValue(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"telecom": [{"system": "phone"}]
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_InvalidAddressUse(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Patient",
		"name": [{"family": "Test"}],
		"address": [{"use": "invalid"}]
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_ValidGenders(t *testing.T) {
	v := NewFHIRValidator()
	genders := []string{"male", "female", "other", "unknown"}

	for _, gender := range genders {
		t.Run(gender, func(t *testing.T) {
			data := json.RawMessage(`{
				"resourceType": "Patient",
				"name": [{"family": "Test"}],
				"gender": "` + gender + `"
			}`)
			err := v.ValidatePatient(data, "")
			assert.Nil(t, err)
		})
	}
}

func TestValidatePatient_InvalidJSON(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{invalid json`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.StatusCode)
}

func TestValidatePatient_MultipleErrors(t *testing.T) {
	v := NewFHIRValidator()
	data := json.RawMessage(`{
		"resourceType": "Observation",
		"gender": "invalid",
		"birthDate": "2099-01-01"
	}`)

	err := v.ValidatePatient(data, "")
	assert.NotNil(t, err)
	// Should have at least 3 errors: wrong resourceType, no name, invalid gender, future DOB
	assert.GreaterOrEqual(t, len(err.OperationOutcome.Issue), 3)
}

func TestValidatePatient_ValidAddressUse(t *testing.T) {
	v := NewFHIRValidator()
	uses := []string{"home", "work", "temp", "old", "billing"}

	for _, use := range uses {
		t.Run(use, func(t *testing.T) {
			data := json.RawMessage(`{
				"resourceType": "Patient",
				"name": [{"family": "Test"}],
				"address": [{"use": "` + use + `"}]
			}`)
			err := v.ValidatePatient(data, "")
			assert.Nil(t, err)
		})
	}
}
