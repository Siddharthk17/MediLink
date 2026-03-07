package fhirerrors_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

func TestNewValidationError(t *testing.T) {
	err := fhirerrors.NewValidationError("test error", "Patient.name")
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "OperationOutcome", err.OperationOutcome.ResourceType)
	assert.Len(t, err.OperationOutcome.Issue, 1)
	assert.Equal(t, fhirerrors.SeverityError, err.OperationOutcome.Issue[0].Severity)
	assert.Equal(t, fhirerrors.CodeInvalid, err.OperationOutcome.Issue[0].Code)
	assert.Equal(t, "test error", err.OperationOutcome.Issue[0].Details.Text)
	assert.Equal(t, []string{"Patient.name"}, err.OperationOutcome.Issue[0].Expression)
}

func TestNewNotFoundError(t *testing.T) {
	err := fhirerrors.NewNotFoundError("Patient", "123")
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Contains(t, err.Error(), "Patient")
	assert.Contains(t, err.Error(), "123")
}

func TestNewProcessingError(t *testing.T) {
	err := fhirerrors.NewProcessingError("internal failure")
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.Equal(t, fhirerrors.SeverityFatal, err.OperationOutcome.Issue[0].Severity)
}

func TestNewConflictError(t *testing.T) {
	err := fhirerrors.NewConflictError("resource already exists")
	assert.Equal(t, http.StatusConflict, err.StatusCode)
	assert.Equal(t, fhirerrors.CodeBusinessRule, err.OperationOutcome.Issue[0].Code)
}

func TestNewForbiddenError(t *testing.T) {
	err := fhirerrors.NewForbiddenError("access denied")
	assert.Equal(t, http.StatusForbidden, err.StatusCode)
	assert.Equal(t, fhirerrors.CodeForbidden, err.OperationOutcome.Issue[0].Code)
}

func TestFHIRError_JSON(t *testing.T) {
	err := fhirerrors.NewValidationError("test")
	jsonBytes := err.JSON()

	var outcome fhirerrors.OperationOutcome
	unmarshalErr := json.Unmarshal(jsonBytes, &outcome)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, "OperationOutcome", outcome.ResourceType)
	assert.Len(t, outcome.Issue, 1)
}

func TestFHIRError_Error(t *testing.T) {
	err := fhirerrors.NewValidationError("my error message")
	assert.Equal(t, "my error message", err.Error())
}

func TestNewMultiIssueError(t *testing.T) {
	issues := []fhirerrors.Issue{
		{Severity: fhirerrors.SeverityError, Code: fhirerrors.CodeInvalid, Details: &fhirerrors.IssueDetail{Text: "error 1"}},
		{Severity: fhirerrors.SeverityWarning, Code: fhirerrors.CodeBusinessRule, Details: &fhirerrors.IssueDetail{Text: "warning 1"}},
	}
	err := fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Len(t, err.OperationOutcome.Issue, 2)
}
