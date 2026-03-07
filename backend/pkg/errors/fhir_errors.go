// Package fhirerrors provides FHIR R4 OperationOutcome error types.
package fhirerrors

import (
	"encoding/json"
	"net/http"
)

// Severity levels for OperationOutcome issues.
const (
	SeverityFatal       = "fatal"
	SeverityError       = "error"
	SeverityWarning     = "warning"
	SeverityInformation = "information"
)

// Issue codes for OperationOutcome.
const (
	CodeInvalid      = "invalid"
	CodeNotFound     = "not-found"
	CodeForbidden    = "forbidden"
	CodeTooMany      = "too-many"
	CodeBusinessRule = "business-rule"
	CodeProcessing   = "processing"
)

// OperationOutcome represents a FHIR R4 OperationOutcome resource.
type OperationOutcome struct {
	ResourceType string  `json:"resourceType"`
	Issue        []Issue `json:"issue"`
}

// Issue represents a single issue within an OperationOutcome.
type Issue struct {
	Severity   string       `json:"severity"`
	Code       string       `json:"code"`
	Details    *IssueDetail `json:"details,omitempty"`
	Expression []string     `json:"expression,omitempty"`
}

// IssueDetail holds the text description of an issue.
type IssueDetail struct {
	Text string `json:"text"`
}

// FHIRError wraps an OperationOutcome with an HTTP status code.
type FHIRError struct {
	StatusCode       int
	OperationOutcome OperationOutcome
}

// Error implements the error interface.
func (e *FHIRError) Error() string {
	if len(e.OperationOutcome.Issue) > 0 && e.OperationOutcome.Issue[0].Details != nil {
		return e.OperationOutcome.Issue[0].Details.Text
	}
	return "FHIR operation error"
}

// JSON returns the OperationOutcome as JSON bytes.
func (e *FHIRError) JSON() []byte {
	data, _ := json.Marshal(e.OperationOutcome)
	return data
}

// NewOperationOutcome creates a new OperationOutcome with a single issue.
func NewOperationOutcome(severity, code, message string, expressions ...string) OperationOutcome {
	issue := Issue{
		Severity: severity,
		Code:     code,
		Details:  &IssueDetail{Text: message},
	}
	if len(expressions) > 0 {
		issue.Expression = expressions
	}
	return OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue:        []Issue{issue},
	}
}

// NewValidationError creates a 400 Bad Request FHIR error.
func NewValidationError(message string, expressions ...string) *FHIRError {
	return &FHIRError{
		StatusCode:       http.StatusBadRequest,
		OperationOutcome: NewOperationOutcome(SeverityError, CodeInvalid, message, expressions...),
	}
}

// NewValidationErrorWithStatus creates a FHIR validation error with a custom status code.
func NewValidationErrorWithStatus(message string, statusCode int, expressions ...string) *FHIRError {
	return &FHIRError{
		StatusCode:       statusCode,
		OperationOutcome: NewOperationOutcome(SeverityError, CodeInvalid, message, expressions...),
	}
}

// NewNotFoundError creates a 404 Not Found FHIR error.
func NewNotFoundError(resourceType, resourceID string) *FHIRError {
	return &FHIRError{
		StatusCode:       http.StatusNotFound,
		OperationOutcome: NewOperationOutcome(SeverityError, CodeNotFound, resourceType+" with ID "+resourceID+" not found"),
	}
}

// NewForbiddenError creates a 403 Forbidden FHIR error.
func NewForbiddenError(message string) *FHIRError {
	return &FHIRError{
		StatusCode:       http.StatusForbidden,
		OperationOutcome: NewOperationOutcome(SeverityError, CodeForbidden, message),
	}
}

// NewProcessingError creates a 500 Internal Server Error FHIR error.
func NewProcessingError(message string) *FHIRError {
	return &FHIRError{
		StatusCode:       http.StatusInternalServerError,
		OperationOutcome: NewOperationOutcome(SeverityFatal, CodeProcessing, message),
	}
}

// NewConflictError creates a 409 Conflict FHIR error.
func NewConflictError(message string) *FHIRError {
	return &FHIRError{
		StatusCode:       http.StatusConflict,
		OperationOutcome: NewOperationOutcome(SeverityError, CodeBusinessRule, message),
	}
}

// NewMultiIssueError creates a FHIR error with multiple issues.
func NewMultiIssueError(statusCode int, issues []Issue) *FHIRError {
	return &FHIRError{
		StatusCode: statusCode,
		OperationOutcome: OperationOutcome{
			ResourceType: "OperationOutcome",
			Issue:        issues,
		},
	}
}

// ErrNotFound is a sentinel error for repository layer.
var ErrNotFound = NewNotFoundError("Resource", "unknown")
