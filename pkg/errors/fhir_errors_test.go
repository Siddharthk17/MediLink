package fhirerrors

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("test error", "Patient.name")
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "OperationOutcome", err.OperationOutcome.ResourceType)
	assert.Len(t, err.OperationOutcome.Issue, 1)
	assert.Equal(t, SeverityError, err.OperationOutcome.Issue[0].Severity)
	assert.Equal(t, CodeInvalid, err.OperationOutcome.Issue[0].Code)
	assert.Equal(t, "test error", err.OperationOutcome.Issue[0].Details.Text)
	assert.Equal(t, []string{"Patient.name"}, err.OperationOutcome.Issue[0].Expression)
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("Patient", "123")
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Contains(t, err.Error(), "Patient")
	assert.Contains(t, err.Error(), "123")
}

func TestNewProcessingError(t *testing.T) {
	err := NewProcessingError("internal failure")
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.Equal(t, SeverityFatal, err.OperationOutcome.Issue[0].Severity)
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("resource already exists")
	assert.Equal(t, http.StatusConflict, err.StatusCode)
	assert.Equal(t, CodeBusinessRule, err.OperationOutcome.Issue[0].Code)
}

func TestNewForbiddenError(t *testing.T) {
	err := NewForbiddenError("access denied")
	assert.Equal(t, http.StatusForbidden, err.StatusCode)
	assert.Equal(t, CodeForbidden, err.OperationOutcome.Issue[0].Code)
}

func TestFHIRError_JSON(t *testing.T) {
	err := NewValidationError("test")
	jsonBytes := err.JSON()

	var outcome OperationOutcome
	unmarshalErr := json.Unmarshal(jsonBytes, &outcome)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, "OperationOutcome", outcome.ResourceType)
	assert.Len(t, outcome.Issue, 1)
}

func TestFHIRError_Error(t *testing.T) {
	err := NewValidationError("my error message")
	assert.Equal(t, "my error message", err.Error())
}

func TestNewMultiIssueError(t *testing.T) {
	issues := []Issue{
		{Severity: SeverityError, Code: CodeInvalid, Details: &IssueDetail{Text: "error 1"}},
		{Severity: SeverityWarning, Code: CodeBusinessRule, Details: &IssueDetail{Text: "warning 1"}},
	}
	err := NewMultiIssueError(http.StatusBadRequest, issues)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Len(t, err.OperationOutcome.Issue, 2)
}
