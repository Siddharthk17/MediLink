// Package llm provides LLM-based extraction of structured lab results from OCR text.
package llm

import (
	"context"
)

// ExtractionResult contains structured data extracted from a lab report.
type ExtractionResult struct {
	ReportDate     string       `json:"reportDate"`
	LabName        string       `json:"labName"`
	PatientName    string       `json:"patientName"`
	OrderingDoctor string       `json:"orderingDoctor"`
	ReportType     string       `json:"reportType"`
	Results        []TestResult `json:"results"`
}

// TestResult represents a single lab test result extracted from OCR text.
type TestResult struct {
	TestName       string  `json:"testName"`
	Value          float64 `json:"value"`
	ValueString    string  `json:"valueString"`
	IsNumeric      bool    `json:"isNumeric"`
	Unit           string  `json:"unit"`
	ReferenceRange string  `json:"referenceRange"`
	RefRangeLow    float64 `json:"refRangeLow"`
	RefRangeHigh   float64 `json:"refRangeHigh"`
	IsAbnormal     bool    `json:"isAbnormal"`
	AbnormalFlag   string  `json:"abnormalFlag"`
	LOINCCode      string  `json:"loincCode"`
	LOINCDisplay   string  `json:"loincDisplay"`
}

// LLMExtractor defines the interface for LLM-based lab result extraction.
type LLMExtractor interface {
	ExtractLabResults(ctx context.Context, ocrText string, docType string) (*ExtractionResult, error)
	ProviderName() string
}
