// Package validator provides FHIR R4 resource validation.
package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// PatientData represents the FHIR Patient resource structure for validation.
type PatientData struct {
	ResourceType       string                   `json:"resourceType"`
	ID                 string                   `json:"id,omitempty"`
	Meta               map[string]interface{}   `json:"meta,omitempty"`
	Identifier         []IdentifierData         `json:"identifier,omitempty"`
	Active             *bool                    `json:"active,omitempty"`
	Name               []HumanNameData          `json:"name,omitempty"`
	Telecom            []ContactPointData       `json:"telecom,omitempty"`
	Gender             string                   `json:"gender,omitempty"`
	BirthDate          string                   `json:"birthDate,omitempty"`
	DeceasedBoolean    *bool                    `json:"deceasedBoolean,omitempty"`
	DeceasedDateTime   string                   `json:"deceasedDateTime,omitempty"`
	Address            []AddressData            `json:"address,omitempty"`
	Communication      []map[string]interface{} `json:"communication,omitempty"`
	GeneralPractitioner []map[string]interface{} `json:"generalPractitioner,omitempty"`
}

// IdentifierData represents a FHIR Identifier.
type IdentifierData struct {
	Use    string `json:"use,omitempty"`
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
}

// HumanNameData represents a FHIR HumanName.
type HumanNameData struct {
	Use    string   `json:"use,omitempty"`
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// ContactPointData represents a FHIR ContactPoint.
type ContactPointData struct {
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
	Use    string `json:"use,omitempty"`
}

// AddressData represents a FHIR Address.
type AddressData struct {
	Use        string   `json:"use,omitempty"`
	Type       string   `json:"type,omitempty"`
	Line       []string `json:"line,omitempty"`
	City       string   `json:"city,omitempty"`
	State      string   `json:"state,omitempty"`
	PostalCode string   `json:"postalCode,omitempty"`
	Country    string   `json:"country,omitempty"`
}

// FHIRValidator validates FHIR resources.
type FHIRValidator struct{}

// NewFHIRValidator creates a new FHIR validator.
func NewFHIRValidator() *FHIRValidator {
	return &FHIRValidator{}
}

// ValidatePatient validates a Patient resource from raw JSON.
// urlID is the ID from the URL parameter (empty string for create operations).
func (v *FHIRValidator) ValidatePatient(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var patient PatientData
	if err := json.Unmarshal(data, &patient); err != nil {
		return fhirerrors.NewValidationError(
			fmt.Sprintf("Invalid JSON: %s", err.Error()),
		)
	}

	var issues []fhirerrors.Issue

	// resourceType must be "Patient"
	if patient.ResourceType != "Patient" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Patient'"},
			Expression: []string{"Patient.resourceType"},
		})
	}

	// ID mismatch check: if id in body and URL ID are both present, they must match
	if urlID != "" && patient.ID != "" && patient.ID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Patient.id"},
		})
	}

	// At least one name element required
	if len(patient.Name) == 0 {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Patient.name: at least one name is required"},
			Expression: []string{"Patient.name"},
		})
	}

	// Each name must have family or given
	for i, name := range patient.Name {
		if name.Family == "" && len(name.Given) == 0 {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Patient.name[%d]: family or given name is required", i)},
				Expression: []string{fmt.Sprintf("Patient.name[%d]", i)},
			})
		}
	}

	// birthDate validation
	if patient.BirthDate != "" {
		birthDate, err := time.Parse("2006-01-02", patient.BirthDate)
		if err != nil {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Patient.birthDate: must be in YYYY-MM-DD format"},
				Expression: []string{"Patient.birthDate"},
			})
		} else if birthDate.After(time.Now()) {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Patient.birthDate: cannot be in the future"},
				Expression: []string{"Patient.birthDate"},
			})
		}
	}

	// gender validation
	validGenders := map[string]bool{"male": true, "female": true, "other": true, "unknown": true}
	if patient.Gender != "" && !validGenders[patient.Gender] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Patient.gender: must be male, female, other, or unknown"},
			Expression: []string{"Patient.gender"},
		})
	}

	// identifier validation: system and value both required
	for i, ident := range patient.Identifier {
		if ident.System == "" || ident.Value == "" {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Patient.identifier[%d]: system and value are both required", i)},
				Expression: []string{fmt.Sprintf("Patient.identifier[%d]", i)},
			})
		}
	}

	// telecom validation: system and value both required
	for i, tel := range patient.Telecom {
		if tel.System == "" || tel.Value == "" {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Patient.telecom[%d]: system and value are both required", i)},
				Expression: []string{fmt.Sprintf("Patient.telecom[%d]", i)},
			})
		}
	}

	// address use validation
	validAddressUse := map[string]bool{"home": true, "work": true, "temp": true, "old": true, "billing": true}
	for i, addr := range patient.Address {
		if addr.Use != "" && !validAddressUse[strings.ToLower(addr.Use)] {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Patient.address[%d].use: must be home, work, temp, old, or billing", i)},
				Expression: []string{fmt.Sprintf("Patient.address[%d].use", i)},
			})
		}
	}

	// deceasedDateTime validation
	if patient.DeceasedBoolean != nil && *patient.DeceasedBoolean && patient.DeceasedDateTime != "" {
		dt, err := time.Parse(time.RFC3339, patient.DeceasedDateTime)
		if err == nil && dt.After(time.Now()) {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Patient.deceasedDateTime: cannot be in the future"},
				Expression: []string{"Patient.deceasedDateTime"},
			})
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}

	return nil
}
