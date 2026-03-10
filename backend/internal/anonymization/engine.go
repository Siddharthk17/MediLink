package anonymization

import (
	"encoding/json"
	"strings"
)

// TaskAnonymizedExport is the Asynq task type for anonymized research exports.
const TaskAnonymizedExport = "research:export"

// deepCopy creates an independent deep copy of a map via JSON round-trip.
// MUST be called before every anonymize* function to ensure the source data
// is never modified.
func deepCopy(m map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(m)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	return out
}

// generalizeToYear truncates a date or datetime field to the four-digit year.
// "1995-03-15" → "1995", "1995-03-15T10:00:00Z" → "1995".
func generalizeToYear(m map[string]interface{}, field string) {
	v, ok := m[field].(string)
	if !ok || len(v) < 4 {
		return
	}
	m[field] = v[:4]
}

// generalizeToMonthYear truncates a datetime to "YYYY-MM".
// "2024-01-15T10:00:00+05:30" → "2024-01".
func generalizeToMonthYear(m map[string]interface{}, field string) {
	v, ok := m[field].(string)
	if !ok || len(v) < 7 {
		return
	}
	m[field] = v[:7]
}

// pseudonymizeID replaces m["id"] with a deterministic pseudonym.
func pseudonymizeID(m map[string]interface{}, salt string) {
	if id, ok := m["id"].(string); ok {
		m["id"] = Pseudonymize(id, salt)
	}
}

// pseudonymizeReference pseudonymises a FHIR reference value inside m[field].
// Handles both the simple string form and the {reference: "..."} object form.
func pseudonymizeReference(m map[string]interface{}, field, salt string) {
	switch v := m[field].(type) {
	case string:
		m[field] = PseudonymizeRef(v, salt)
	case map[string]interface{}:
		if ref, ok := v["reference"].(string); ok {
			v["reference"] = PseudonymizeRef(ref, salt)
		}
	}
}

// generalizePeriod generalises period.start and period.end to month/year.
func generalizePeriod(m map[string]interface{}) {
	period, ok := m["period"].(map[string]interface{})
	if !ok {
		return
	}
	generalizeToMonthYear(period, "start")
	generalizeToMonthYear(period, "end")
}

// removeFields deletes a set of top-level keys from a map.
func removeFields(m map[string]interface{}, fields ...string) {
	for _, f := range fields {
		delete(m, f)
	}
}

// removeMeta strips server metadata that should not leak into exports.
func removeMeta(m map[string]interface{}) {
	delete(m, "meta")
}

// Per-resource anonymizers

// AnonymizePatient removes direct identifiers and generalises birth date.
func AnonymizePatient(p map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(p)
	pseudonymizeID(out, salt)
	removeMeta(out)
	removeFields(out,
		"name", "telecom", "address", "photo",
		"contact", "identifier", "text",
	)
	generalizeToYear(out, "birthDate")
	// Keep gender — needed for k-anonymity grouping.
	return out
}

// AnonymizeEncounter removes participant identifiers and generalises period.
func AnonymizeEncounter(e map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(e)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "subject", salt)
	removeFields(out, "reasonReference", "text")

	// Strip participant.individual references.
	if participants, ok := out["participant"].([]interface{}); ok {
		for _, p := range participants {
			if pm, ok := p.(map[string]interface{}); ok {
				delete(pm, "individual")
			}
		}
	}

	generalizePeriod(out)
	return out
}

// AnonymizeCondition keeps clinical codes but removes recorder/asserter.
func AnonymizeCondition(c map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(c)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "subject", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "recorder", "asserter", "note", "text")
	generalizeToMonthYear(out, "onsetDateTime")
	generalizeToMonthYear(out, "recordedDate")
	generalizeToMonthYear(out, "abatementDateTime")
	return out
}

// AnonymizeMedicationRequest keeps the medication but removes requester/recorder.
func AnonymizeMedicationRequest(m map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(m)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "subject", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "requester", "recorder", "note", "text")
	generalizeToMonthYear(out, "authoredOn")
	return out
}

// AnonymizeObservation keeps codes and values; removes performer/note.
func AnonymizeObservation(o map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(o)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "subject", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "performer", "note", "text")
	generalizeToMonthYear(out, "effectiveDateTime")
	generalizeToMonthYear(out, "issued")
	return out
}

// AnonymizeDiagnosticReport removes performer, presentedForm (NEVER export PDF), conclusion.
func AnonymizeDiagnosticReport(d map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(d)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "subject", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "performer", "presentedForm", "conclusion", "conclusionCode", "text")
	generalizeToMonthYear(out, "effectiveDateTime")
	generalizeToMonthYear(out, "issued")

	// Pseudonymize result references.
	if results, ok := out["result"].([]interface{}); ok {
		for _, r := range results {
			if rm, ok := r.(map[string]interface{}); ok {
				if ref, ok := rm["reference"].(string); ok {
					rm["reference"] = PseudonymizeRef(ref, salt)
				}
			}
		}
	}
	return out
}

// AnonymizeAllergyIntolerance keeps code, criticality, reaction.
func AnonymizeAllergyIntolerance(a map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(a)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "patient", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "recorder", "asserter", "note", "text")
	generalizeToMonthYear(out, "onsetDateTime")
	generalizeToMonthYear(out, "recordedDate")
	return out
}

// AnonymizeImmunization keeps vaccineCode; removes performer/lotNumber.
func AnonymizeImmunization(i map[string]interface{}, salt string) map[string]interface{} {
	out := deepCopy(i)
	pseudonymizeID(out, salt)
	removeMeta(out)
	pseudonymizeReference(out, "patient", salt)
	pseudonymizeReference(out, "encounter", salt)
	removeFields(out, "performer", "lotNumber", "fundingSource", "education", "note", "text")
	generalizeToMonthYear(out, "occurrenceDateTime")
	return out
}

// AnonymizeResource dispatches to the appropriate per-type anonymizer.
// Unknown resource types are returned with only ID pseudonymised and
// direct-identifier fields stripped.
func AnonymizeResource(resourceType string, data map[string]interface{}, salt string) map[string]interface{} {
	switch resourceType {
	case "Patient":
		return AnonymizePatient(data, salt)
	case "Encounter":
		return AnonymizeEncounter(data, salt)
	case "Condition":
		return AnonymizeCondition(data, salt)
	case "MedicationRequest":
		return AnonymizeMedicationRequest(data, salt)
	case "Observation":
		return AnonymizeObservation(data, salt)
	case "DiagnosticReport":
		return AnonymizeDiagnosticReport(data, salt)
	case "AllergyIntolerance":
		return AnonymizeAllergyIntolerance(data, salt)
	case "Immunization":
		return AnonymizeImmunization(data, salt)
	default:
		out := deepCopy(data)
		pseudonymizeID(out, salt)
		removeMeta(out)
		removeFields(out, "name", "telecom", "address", "identifier", "text")
		return out
	}
}

// ExtractBirthYear pulls the (possibly generalised) birth year from Patient data.
func ExtractBirthYear(data map[string]interface{}) string {
	bd, ok := data["birthDate"].(string)
	if !ok || len(bd) < 4 {
		return ""
	}
	return bd[:4]
}

// ExtractGender pulls the gender string from Patient data.
func ExtractGender(data map[string]interface{}) string {
	g, _ := data["gender"].(string)
	return g
}

// ExtractPrimaryConditionCode pulls the first ICD-10 / coding code from Condition.code.
func ExtractPrimaryConditionCode(data map[string]interface{}) string {
	code, ok := data["code"].(map[string]interface{})
	if !ok {
		return ""
	}
	codings, ok := code["coding"].([]interface{})
	if !ok || len(codings) == 0 {
		return ""
	}
	first, ok := codings[0].(map[string]interface{})
	if !ok {
		return ""
	}
	c, _ := first["code"].(string)
	if c != "" {
		return c
	}
	// Fall back to the top-level text in the CodeableConcept.
	text, _ := code["text"].(string)
	return strings.ToLower(text)
}
