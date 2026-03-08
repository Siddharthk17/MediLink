package validator

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"time"

	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
)

// Helper functions

func extractString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func extractMap(m map[string]interface{}, key string) map[string]interface{} {
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return sub
}

func extractArray(m map[string]interface{}, key string) []interface{} {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	return arr
}

// extractCodingCode extracts coding[0].code from a CodeableConcept field.
func extractCodingCode(m map[string]interface{}, fieldName string) string {
	cc := extractMap(m, fieldName)
	if cc == nil {
		return ""
	}
	coding := extractArray(cc, "coding")
	if len(coding) == 0 {
		return ""
	}
	entry, ok := coding[0].(map[string]interface{})
	if !ok {
		return ""
	}
	return extractString(entry, "code")
}

// extractReference extracts the reference string from a Reference field.
func extractReference(m map[string]interface{}, fieldName string) string {
	ref := extractMap(m, fieldName)
	if ref == nil {
		return ""
	}
	return extractString(ref, "reference")
}

var patientRefPattern = regexp.MustCompile(`^Patient/[A-Za-z0-9\-\.]+$`)

func isValidPatientRef(ref string) bool {
	return patientRefPattern.MatchString(ref)
}

var icd10Pattern = regexp.MustCompile(`^[A-Za-z]\d{2}(\.\d{1,2})?$`)

// parseDateTime tries RFC3339 then date-only format.
func parseDateTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// 1. ValidatePractitioner

func (v *FHIRValidator) ValidatePractitioner(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Practitioner" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Practitioner'"},
			Expression: []string{"Practitioner.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Practitioner.id"},
		})
	}

	names := extractArray(m, "name")
	if len(names) == 0 {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Practitioner.name: at least one name is required"},
			Expression: []string{"Practitioner.name"},
		})
	}
	for i, n := range names {
		nm, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		if extractString(nm, "family") == "" {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Practitioner.name[%d].family is required", i)},
				Expression: []string{fmt.Sprintf("Practitioner.name[%d].family", i)},
			})
		}
	}

	qualifications := extractArray(m, "qualification")
	for i, q := range qualifications {
		qm, ok := q.(map[string]interface{})
		if !ok {
			continue
		}
		if extractMap(qm, "code") == nil {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Practitioner.qualification[%d].code is required", i)},
				Expression: []string{fmt.Sprintf("Practitioner.qualification[%d].code", i)},
			})
		}
	}

	validGenders := map[string]bool{"male": true, "female": true, "other": true, "unknown": true}
	gender := extractString(m, "gender")
	if gender != "" && !validGenders[gender] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Practitioner.gender: must be male, female, other, or unknown"},
			Expression: []string{"Practitioner.gender"},
		})
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 2. ValidateOrganization

func (v *FHIRValidator) ValidateOrganization(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Organization" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Organization'"},
			Expression: []string{"Organization.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Organization.id"},
		})
	}

	name := extractString(m, "name")
	if name == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Organization.name is required"},
			Expression: []string{"Organization.name"},
		})
	}

	validOrgTypes := map[string]bool{
		"prov": true, "hosp": true, "dept": true, "team": true, "govt": true,
		"ins": true, "edu": true, "reli": true, "crs": true, "bus": true,
	}
	typeArr := extractArray(m, "type")
	for i, t := range typeArr {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		coding := extractArray(tm, "coding")
		for j, c := range coding {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			code := extractString(cm, "code")
			if code != "" && !validOrgTypes[code] {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Organization.type[%d].coding[%d].code: invalid organization type '%s'", i, j, code)},
					Expression: []string{fmt.Sprintf("Organization.type[%d].coding[%d].code", i, j)},
				})
			}
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 3. ValidateEncounter

func (v *FHIRValidator) ValidateEncounter(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Encounter" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Encounter'"},
			Expression: []string{"Encounter.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Encounter.id"},
		})
	}

	validStatuses := map[string]bool{
		"planned": true, "arrived": true, "triaged": true, "in-progress": true,
		"onleave": true, "finished": true, "cancelled": true,
	}
	status := extractString(m, "status")
	if status == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.status is required"},
			Expression: []string{"Encounter.status"},
		})
	} else if !validStatuses[status] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Encounter.status: invalid value '%s'", status)},
			Expression: []string{"Encounter.status"},
		})
	}

	validClassCodes := map[string]bool{
		"AMB": true, "IMP": true, "EMER": true, "VR": true, "HH": true, "OBSENC": true,
	}
	classMap := extractMap(m, "class")
	if classMap == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.class is required"},
			Expression: []string{"Encounter.class"},
		})
	} else {
		classCode := extractString(classMap, "code")
		if classCode == "" {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Encounter.class.code is required"},
				Expression: []string{"Encounter.class.code"},
			})
		} else if !validClassCodes[classCode] {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Encounter.class.code: invalid value '%s'", classCode)},
				Expression: []string{"Encounter.class.code"},
			})
		}
	}

	subjectRef := extractReference(m, "subject")
	if subjectRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.subject is required"},
			Expression: []string{"Encounter.subject"},
		})
	} else if !isValidPatientRef(subjectRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.subject.reference must match 'Patient/{id}'"},
			Expression: []string{"Encounter.subject.reference"},
		})
	}

	period := extractMap(m, "period")
	needsStart := status == "in-progress" || status == "finished" || status == "onleave"
	if needsStart && (period == nil || extractString(period, "start") == "") {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.period.start is required when status is " + status},
			Expression: []string{"Encounter.period.start"},
		})
	}
	if status == "finished" && (period == nil || extractString(period, "end") == "") {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Encounter.period.end is required when status is finished"},
			Expression: []string{"Encounter.period.end"},
		})
	}
	if period != nil {
		startStr := extractString(period, "start")
		endStr := extractString(period, "end")
		if startStr != "" && endStr != "" {
			startT, errS := parseDateTime(startStr)
			endT, errE := parseDateTime(endStr)
			if errS == nil && errE == nil && !endT.After(startT) {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: "Encounter.period.end must be after period.start"},
					Expression: []string{"Encounter.period.end"},
				})
			}
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 4. ValidateCondition

func (v *FHIRValidator) ValidateCondition(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Condition" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Condition'"},
			Expression: []string{"Condition.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Condition.id"},
		})
	}

	validClinicalStatus := map[string]bool{
		"active": true, "recurrence": true, "relapse": true,
		"inactive": true, "remission": true, "resolved": true,
	}
	clinicalCode := extractCodingCode(m, "clinicalStatus")
	if clinicalCode == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Condition.clinicalStatus is required with a valid coding"},
			Expression: []string{"Condition.clinicalStatus"},
		})
	} else if !validClinicalStatus[clinicalCode] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Condition.clinicalStatus: invalid code '%s'", clinicalCode)},
			Expression: []string{"Condition.clinicalStatus"},
		})
	}

	validVerificationStatus := map[string]bool{
		"unconfirmed": true, "provisional": true, "differential": true,
		"confirmed": true, "refuted": true, "entered-in-error": true,
	}
	verificationCode := extractCodingCode(m, "verificationStatus")
	if verificationCode != "" && !validVerificationStatus[verificationCode] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Condition.verificationStatus: invalid code '%s'", verificationCode)},
			Expression: []string{"Condition.verificationStatus"},
		})
	}

	codeMap := extractMap(m, "code")
	if codeMap == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Condition.code is required"},
			Expression: []string{"Condition.code"},
		})
	} else {
		coding := extractArray(codeMap, "coding")
		if len(coding) == 0 {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Condition.code must have at least one coding entry"},
				Expression: []string{"Condition.code.coding"},
			})
		} else {
			firstCoding, ok := coding[0].(map[string]interface{})
			if ok {
				codeVal := extractString(firstCoding, "code")
				if codeVal != "" && !icd10Pattern.MatchString(codeVal) {
					issues = append(issues, fhirerrors.Issue{
						Severity:   fhirerrors.SeverityError,
						Code:       fhirerrors.CodeInvalid,
						Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Condition.code.coding[0].code: '%s' does not match ICD-10 pattern", codeVal)},
						Expression: []string{"Condition.code.coding[0].code"},
					})
				}
			}
		}
	}

	subjectRef := extractReference(m, "subject")
	if subjectRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Condition.subject is required"},
			Expression: []string{"Condition.subject"},
		})
	} else if !isValidPatientRef(subjectRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Condition.subject.reference must match 'Patient/{id}'"},
			Expression: []string{"Condition.subject.reference"},
		})
	}

	abatementDT := extractString(m, "abatementDateTime")
	if abatementDT != "" && clinicalCode == "active" && verificationCode != "entered-in-error" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Condition.abatementDateTime cannot be set when clinicalStatus is 'active'"},
			Expression: []string{"Condition.abatementDateTime"},
		})
	}

	onsetDT := extractString(m, "onsetDateTime")
	if onsetDT != "" && abatementDT != "" {
		onsetT, errO := parseDateTime(onsetDT)
		abateT, errA := parseDateTime(abatementDT)
		if errO == nil && errA == nil && !abateT.After(onsetT) {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Condition.abatementDateTime must be after onsetDateTime"},
				Expression: []string{"Condition.abatementDateTime"},
			})
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 5. ValidateMedicationRequest

func (v *FHIRValidator) ValidateMedicationRequest(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "MedicationRequest" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'MedicationRequest'"},
			Expression: []string{"MedicationRequest.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"MedicationRequest.id"},
		})
	}

	validStatuses := map[string]bool{
		"active": true, "on-hold": true, "cancelled": true, "completed": true,
		"entered-in-error": true, "stopped": true, "draft": true, "unknown": true,
	}
	status := extractString(m, "status")
	if status == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest.status is required"},
			Expression: []string{"MedicationRequest.status"},
		})
	} else if !validStatuses[status] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("MedicationRequest.status: invalid value '%s'", status)},
			Expression: []string{"MedicationRequest.status"},
		})
	}

	validIntents := map[string]bool{
		"proposal": true, "plan": true, "order": true, "original-order": true,
		"reflex-order": true, "filler-order": true, "instance-order": true, "option": true,
	}
	intent := extractString(m, "intent")
	if intent == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest.intent is required"},
			Expression: []string{"MedicationRequest.intent"},
		})
	} else if !validIntents[intent] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("MedicationRequest.intent: invalid value '%s'", intent)},
			Expression: []string{"MedicationRequest.intent"},
		})
	}

	_, hasConcept := m["medicationCodeableConcept"]
	_, hasRef := m["medicationReference"]
	if !hasConcept && !hasRef {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest: medicationCodeableConcept or medicationReference is required"},
			Expression: []string{"MedicationRequest.medication[x]"},
		})
	}

	subjectRef := extractReference(m, "subject")
	if subjectRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest.subject is required"},
			Expression: []string{"MedicationRequest.subject"},
		})
	} else if !isValidPatientRef(subjectRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest.subject.reference must match 'Patient/{id}'"},
			Expression: []string{"MedicationRequest.subject.reference"},
		})
	}

	if extractString(m, "authoredOn") == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "MedicationRequest.authoredOn is required"},
			Expression: []string{"MedicationRequest.authoredOn"},
		})
	}

	dosageInstructions := extractArray(m, "dosageInstruction")
	for i, di := range dosageInstructions {
		dim, ok := di.(map[string]interface{})
		if !ok {
			continue
		}
		doseAndRate := extractArray(dim, "doseAndRate")
		for j, dr := range doseAndRate {
			drm, ok := dr.(map[string]interface{})
			if !ok {
				continue
			}
			dq := extractMap(drm, "doseQuantity")
			if dq != nil {
				if val, ok := dq["value"]; ok {
					if num, ok := val.(float64); ok && num <= 0 {
						issues = append(issues, fhirerrors.Issue{
							Severity:   fhirerrors.SeverityError,
							Code:       fhirerrors.CodeInvalid,
							Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("MedicationRequest.dosageInstruction[%d].doseAndRate[%d].doseQuantity.value must be > 0", i, j)},
							Expression: []string{fmt.Sprintf("MedicationRequest.dosageInstruction[%d].doseAndRate[%d].doseQuantity.value", i, j)},
						})
					}
				}
			}
		}
		timing := extractMap(dim, "timing")
		if timing != nil {
			repeat := extractMap(timing, "repeat")
			if repeat != nil {
				if freq, ok := repeat["frequency"]; ok {
					if num, ok := freq.(float64); ok && num <= 0 {
						issues = append(issues, fhirerrors.Issue{
							Severity:   fhirerrors.SeverityError,
							Code:       fhirerrors.CodeInvalid,
							Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("MedicationRequest.dosageInstruction[%d].timing.repeat.frequency must be > 0", i)},
							Expression: []string{fmt.Sprintf("MedicationRequest.dosageInstruction[%d].timing.repeat.frequency", i)},
						})
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 6. ValidateObservation

func (v *FHIRValidator) ValidateObservation(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Observation" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Observation'"},
			Expression: []string{"Observation.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Observation.id"},
		})
	}

	validStatuses := map[string]bool{
		"registered": true, "preliminary": true, "final": true, "amended": true,
		"corrected": true, "cancelled": true, "entered-in-error": true, "unknown": true,
	}
	status := extractString(m, "status")
	if status == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation.status is required"},
			Expression: []string{"Observation.status"},
		})
	} else if !validStatuses[status] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Observation.status: invalid value '%s'", status)},
			Expression: []string{"Observation.status"},
		})
	}

	codeMap := extractMap(m, "code")
	if codeMap == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation.code is required"},
			Expression: []string{"Observation.code"},
		})
	} else if len(extractArray(codeMap, "coding")) == 0 {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation.code must have a coding array"},
			Expression: []string{"Observation.code.coding"},
		})
	}

	subjectRef := extractReference(m, "subject")
	if subjectRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation.subject is required"},
			Expression: []string{"Observation.subject"},
		})
	} else if !isValidPatientRef(subjectRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation.subject.reference must match 'Patient/{id}'"},
			Expression: []string{"Observation.subject.reference"},
		})
	}

	_, hasEffDT := m["effectiveDateTime"]
	_, hasEffPeriod := m["effectivePeriod"]
	if !hasEffDT && !hasEffPeriod {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation: effectiveDateTime or effectivePeriod is required"},
			Expression: []string{"Observation.effective[x]"},
		})
	}

	vq := extractMap(m, "valueQuantity")
	if vq != nil {
		if val, ok := vq["value"]; ok {
			if num, ok := val.(float64); ok {
				if math.IsInf(num, 0) || math.IsNaN(num) {
					issues = append(issues, fhirerrors.Issue{
						Severity:   fhirerrors.SeverityError,
						Code:       fhirerrors.CodeInvalid,
						Details:    &fhirerrors.IssueDetail{Text: "Observation.valueQuantity.value must be a finite number"},
						Expression: []string{"Observation.valueQuantity.value"},
					})
				}
			}
		}
	}

	refRanges := extractArray(m, "referenceRange")
	for i, rr := range refRanges {
		rrm, ok := rr.(map[string]interface{})
		if !ok {
			continue
		}
		low := extractMap(rrm, "low")
		high := extractMap(rrm, "high")
		if low != nil && high != nil {
			lowVal, lowOk := low["value"].(float64)
			highVal, highOk := high["value"].(float64)
			if lowOk && highOk && highVal <= lowVal {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Observation.referenceRange[%d]: high.value must be greater than low.value", i)},
					Expression: []string{fmt.Sprintf("Observation.referenceRange[%d]", i)},
				})
			}
		}
	}

	validInterpretation := map[string]bool{
		"L": true, "N": true, "H": true, "LL": true, "HH": true,
		"A": true, "AA": true, "U": true, "IND": true, "E": true,
	}
	interpretations := extractArray(m, "interpretation")
	for i, interp := range interpretations {
		im, ok := interp.(map[string]interface{})
		if !ok {
			continue
		}
		coding := extractArray(im, "coding")
		for j, c := range coding {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			code := extractString(cm, "code")
			if code != "" && !validInterpretation[code] {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Observation.interpretation[%d].coding[%d].code: invalid value '%s'", i, j, code)},
					Expression: []string{fmt.Sprintf("Observation.interpretation[%d].coding[%d].code", i, j)},
				})
			}
		}
	}

	// FHIR R4: Observation must have exactly one value[x] OR a dataAbsentReason, not both
	valueFields := []string{
		"valueQuantity", "valueCodeableConcept", "valueString", "valueBoolean",
		"valueInteger", "valueRange", "valueRatio", "valueSampledData",
		"valueTime", "valueDateTime", "valuePeriod",
	}
	valueCount := 0
	for _, field := range valueFields {
		if _, ok := m[field]; ok {
			valueCount++
		}
	}
	_, hasDataAbsentReason := m["dataAbsentReason"]

	if valueCount == 0 && !hasDataAbsentReason {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation: must have either a value[x] or a dataAbsentReason"},
			Expression: []string{"Observation.value[x]"},
		})
	} else if valueCount > 0 && hasDataAbsentReason {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation: cannot have both a value[x] and a dataAbsentReason"},
			Expression: []string{"Observation.value[x]"},
		})
	} else if valueCount > 1 {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Observation: must have at most one value[x] element"},
			Expression: []string{"Observation.value[x]"},
		})
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 7. ValidateDiagnosticReport

var observationRefPattern = regexp.MustCompile(`^Observation/[A-Za-z0-9\-\.]+$`)

func (v *FHIRValidator) ValidateDiagnosticReport(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "DiagnosticReport" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'DiagnosticReport'"},
			Expression: []string{"DiagnosticReport.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"DiagnosticReport.id"},
		})
	}

	validStatuses := map[string]bool{
		"registered": true, "partial": true, "preliminary": true, "final": true,
		"amended": true, "corrected": true, "appended": true, "cancelled": true,
		"entered-in-error": true, "unknown": true,
	}
	status := extractString(m, "status")
	if status == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "DiagnosticReport.status is required"},
			Expression: []string{"DiagnosticReport.status"},
		})
	} else if !validStatuses[status] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("DiagnosticReport.status: invalid value '%s'", status)},
			Expression: []string{"DiagnosticReport.status"},
		})
	}

	if extractMap(m, "code") == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "DiagnosticReport.code is required"},
			Expression: []string{"DiagnosticReport.code"},
		})
	}

	subjectRef := extractReference(m, "subject")
	if subjectRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "DiagnosticReport.subject is required"},
			Expression: []string{"DiagnosticReport.subject"},
		})
	} else if !isValidPatientRef(subjectRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "DiagnosticReport.subject.reference must match 'Patient/{id}'"},
			Expression: []string{"DiagnosticReport.subject.reference"},
		})
	}

	results := extractArray(m, "result")
	for i, r := range results {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		ref := extractString(rm, "reference")
		if ref != "" && !observationRefPattern.MatchString(ref) {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("DiagnosticReport.result[%d].reference must match 'Observation/{id}'", i)},
				Expression: []string{fmt.Sprintf("DiagnosticReport.result[%d].reference", i)},
			})
		}
	}

	presentedForms := extractArray(m, "presentedForm")
	for i, pf := range presentedForms {
		pfm, ok := pf.(map[string]interface{})
		if !ok {
			continue
		}
		if extractString(pfm, "contentType") == "" {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("DiagnosticReport.presentedForm[%d].contentType is required", i)},
				Expression: []string{fmt.Sprintf("DiagnosticReport.presentedForm[%d].contentType", i)},
			})
		}
		if sizeVal, ok := pfm["size"]; ok {
			if num, ok := sizeVal.(float64); ok && num <= 0 {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("DiagnosticReport.presentedForm[%d].size must be > 0", i)},
					Expression: []string{fmt.Sprintf("DiagnosticReport.presentedForm[%d].size", i)},
				})
			}
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 8. ValidateAllergyIntolerance

func (v *FHIRValidator) ValidateAllergyIntolerance(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "AllergyIntolerance" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'AllergyIntolerance'"},
			Expression: []string{"AllergyIntolerance.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"AllergyIntolerance.id"},
		})
	}

	clinicalCode := extractCodingCode(m, "clinicalStatus")
	if clinicalCode == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.clinicalStatus is required with a valid coding"},
			Expression: []string{"AllergyIntolerance.clinicalStatus"},
		})
	}

	verificationCode := extractCodingCode(m, "verificationStatus")
	if verificationCode == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.verificationStatus is required with a valid coding"},
			Expression: []string{"AllergyIntolerance.verificationStatus"},
		})
	}

	if extractMap(m, "code") == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.code is required"},
			Expression: []string{"AllergyIntolerance.code"},
		})
	}

	patientRef := extractReference(m, "patient")
	if patientRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.patient is required"},
			Expression: []string{"AllergyIntolerance.patient"},
		})
	} else if !isValidPatientRef(patientRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.patient.reference must match 'Patient/{id}'"},
			Expression: []string{"AllergyIntolerance.patient.reference"},
		})
	}

	validTypes := map[string]bool{"allergy": true, "intolerance": true}
	typeVal := extractString(m, "type")
	if typeVal != "" && !validTypes[typeVal] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.type: must be allergy or intolerance"},
			Expression: []string{"AllergyIntolerance.type"},
		})
	}

	validCategories := map[string]bool{"food": true, "medication": true, "environment": true, "biologic": true}
	categories := extractArray(m, "category")
	for i, cat := range categories {
		catStr, ok := cat.(string)
		if !ok {
			continue
		}
		if !validCategories[catStr] {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("AllergyIntolerance.category[%d]: invalid value '%s'", i, catStr)},
				Expression: []string{fmt.Sprintf("AllergyIntolerance.category[%d]", i)},
			})
		}
	}

	validCriticality := map[string]bool{"low": true, "high": true, "unable-to-assess": true}
	criticality := extractString(m, "criticality")
	if criticality != "" && !validCriticality[criticality] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.criticality: must be low, high, or unable-to-assess"},
			Expression: []string{"AllergyIntolerance.criticality"},
		})
	}

	validSeverity := map[string]bool{"mild": true, "moderate": true, "severe": true}
	reactions := extractArray(m, "reaction")
	for i, r := range reactions {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		sev := extractString(rm, "severity")
		if sev != "" && !validSeverity[sev] {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("AllergyIntolerance.reaction[%d].severity: must be mild, moderate, or severe", i)},
				Expression: []string{fmt.Sprintf("AllergyIntolerance.reaction[%d].severity", i)},
			})
		}
	}

	if clinicalCode == "active" && verificationCode == "refuted" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "AllergyIntolerance.clinicalStatus cannot be 'active' when verificationStatus is 'refuted'"},
			Expression: []string{"AllergyIntolerance.clinicalStatus"},
		})
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

// 9. ValidateImmunization

func (v *FHIRValidator) ValidateImmunization(data json.RawMessage, urlID string) *fhirerrors.FHIRError {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fhirerrors.NewValidationError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	var issues []fhirerrors.Issue

	if extractString(m, "resourceType") != "Immunization" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "resourceType must be 'Immunization'"},
			Expression: []string{"Immunization.resourceType"},
		})
	}

	bodyID := extractString(m, "id")
	if urlID != "" && bodyID != "" && bodyID != urlID {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Resource ID in body does not match URL parameter"},
			Expression: []string{"Immunization.id"},
		})
	}

	validStatuses := map[string]bool{"completed": true, "entered-in-error": true, "not-done": true}
	status := extractString(m, "status")
	if status == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.status is required"},
			Expression: []string{"Immunization.status"},
		})
	} else if !validStatuses[status] {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Immunization.status: invalid value '%s'", status)},
			Expression: []string{"Immunization.status"},
		})
	}

	vcMap := extractMap(m, "vaccineCode")
	if vcMap == nil {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.vaccineCode is required"},
			Expression: []string{"Immunization.vaccineCode"},
		})
	} else if len(extractArray(vcMap, "coding")) == 0 {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.vaccineCode must have a coding array"},
			Expression: []string{"Immunization.vaccineCode.coding"},
		})
	}

	patientRef := extractReference(m, "patient")
	if patientRef == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.patient is required"},
			Expression: []string{"Immunization.patient"},
		})
	} else if !isValidPatientRef(patientRef) {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.patient.reference must match 'Patient/{id}'"},
			Expression: []string{"Immunization.patient.reference"},
		})
	}

	occurrenceDT := extractString(m, "occurrenceDateTime")
	if status == "completed" && occurrenceDT == "" {
		issues = append(issues, fhirerrors.Issue{
			Severity:   fhirerrors.SeverityError,
			Code:       fhirerrors.CodeInvalid,
			Details:    &fhirerrors.IssueDetail{Text: "Immunization.occurrenceDateTime is required when status is 'completed'"},
			Expression: []string{"Immunization.occurrenceDateTime"},
		})
	}

	if status == "completed" && occurrenceDT != "" {
		t, err := parseDateTime(occurrenceDT)
		if err == nil && t.After(time.Now()) {
			issues = append(issues, fhirerrors.Issue{
				Severity:   fhirerrors.SeverityError,
				Code:       fhirerrors.CodeInvalid,
				Details:    &fhirerrors.IssueDetail{Text: "Immunization.occurrenceDateTime cannot be in the future when status is 'completed'"},
				Expression: []string{"Immunization.occurrenceDateTime"},
			})
		}
	}

	dq := extractMap(m, "doseQuantity")
	if dq != nil {
		if val, ok := dq["value"]; ok {
			if num, ok := val.(float64); ok && num <= 0 {
				issues = append(issues, fhirerrors.Issue{
					Severity:   fhirerrors.SeverityError,
					Code:       fhirerrors.CodeInvalid,
					Details:    &fhirerrors.IssueDetail{Text: "Immunization.doseQuantity.value must be > 0"},
					Expression: []string{"Immunization.doseQuantity.value"},
				})
			}
		}
	}

	protocols := extractArray(m, "protocolApplied")
	for i, p := range protocols {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if doseNum, ok := pm["doseNumberPositiveInt"]; ok {
			dn, ok := doseNum.(float64)
			if ok {
				if dn < 1 {
					issues = append(issues, fhirerrors.Issue{
						Severity:   fhirerrors.SeverityError,
						Code:       fhirerrors.CodeInvalid,
						Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Immunization.protocolApplied[%d].doseNumberPositiveInt must be >= 1", i)},
						Expression: []string{fmt.Sprintf("Immunization.protocolApplied[%d].doseNumberPositiveInt", i)},
					})
				}
				if seriesDoses, ok := pm["seriesDosesPositiveInt"]; ok {
					sd, ok := seriesDoses.(float64)
					if ok && dn > sd {
						issues = append(issues, fhirerrors.Issue{
							Severity:   fhirerrors.SeverityError,
							Code:       fhirerrors.CodeInvalid,
							Details:    &fhirerrors.IssueDetail{Text: fmt.Sprintf("Immunization.protocolApplied[%d].doseNumberPositiveInt must be <= seriesDosesPositiveInt", i)},
							Expression: []string{fmt.Sprintf("Immunization.protocolApplied[%d].doseNumberPositiveInt", i)},
						})
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return fhirerrors.NewMultiIssueError(http.StatusBadRequest, issues)
	}
	return nil
}

