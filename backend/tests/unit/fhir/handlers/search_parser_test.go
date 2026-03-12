package handlers_test

// Search Parser Tests
//
// These tests verify that FHIR search parameter parsers correctly extract
// query parameters, build SQL conditions with parameterized arguments, and
// handle pagination. Parameterized queries prevent SQL injection — these
// tests document that the parsers produce correct, safe SQL conditions.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/fhir/handlers"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func callParser(url string, parser handlers.SearchParamParser) (conditions []string, args []interface{}, argIdx, count, offset int) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	c.Request = req
	return parser(c)
}

// --- Pagination ---

func TestPagination_DefaultValues(t *testing.T) {
	conds, args, _, count, offset := callParser("/fhir/R4/Observation", handlers.ObservationSearchParser)
	assert.Empty(t, conds, "no query params = no conditions")
	assert.Empty(t, args)
	assert.Equal(t, 20, count, "default count is 20")
	assert.Equal(t, 0, offset, "default offset is 0")
}

func TestPagination_CustomValues(t *testing.T) {
	_, _, _, count, offset := callParser("/fhir/R4/Observation?_count=50&_offset=10", handlers.ObservationSearchParser)
	assert.Equal(t, 50, count)
	assert.Equal(t, 10, offset)
}

func TestPagination_MaxCountCapped(t *testing.T) {
	_, _, _, count, _ := callParser("/fhir/R4/Observation?_count=999", handlers.ObservationSearchParser)
	assert.Equal(t, 100, count, "count must be capped at 100")
}

func TestPagination_InvalidCountIgnored(t *testing.T) {
	_, _, _, count, _ := callParser("/fhir/R4/Observation?_count=abc", handlers.ObservationSearchParser)
	assert.Equal(t, 20, count, "invalid _count falls back to default")
}

func TestPagination_NegativeCountIgnored(t *testing.T) {
	_, _, _, count, _ := callParser("/fhir/R4/Observation?_count=-5", handlers.ObservationSearchParser)
	assert.Equal(t, 20, count, "negative _count falls back to default")
}

func TestPagination_NegativeOffsetIgnored(t *testing.T) {
	_, _, _, _, offset := callParser("/fhir/R4/Observation?_offset=-1", handlers.ObservationSearchParser)
	assert.Equal(t, 0, offset, "negative _offset falls back to 0")
}

// --- Observation Search Parser ---

func TestObservationParser_PatientParam(t *testing.T) {
	conds, args, _, _, _ := callParser("/fhir/R4/Observation?patient=Patient/123", handlers.ObservationSearchParser)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "patient_ref")
	require.Len(t, args, 1)
	assert.Equal(t, "Patient/123", args[0])
}

func TestObservationParser_CodeParam(t *testing.T) {
	conds, args, _, _, _ := callParser("/fhir/R4/Observation?code=85354-9", handlers.ObservationSearchParser)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "code")
	require.Len(t, args, 1)
	assert.Equal(t, "85354-9", args[0])
}

func TestObservationParser_MultipleParams(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Observation?patient=Patient/123&code=85354-9&status=final",
		handlers.ObservationSearchParser,
	)
	require.Len(t, conds, 3)
	require.Len(t, args, 3)
}

func TestObservationParser_DatePrefix(t *testing.T) {
	tests := []struct {
		query    string
		operator string
	}{
		{"date=gt2024-01-01", ">"},
		{"date=lt2024-12-31", "<"},
		{"date=ge2024-06-01", ">="},
		{"date=le2024-06-30", "<="},
		{"date=eq2024-03-15", "="},
		{"date=2024-03-15", "="},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			conds, args, _, _, _ := callParser("/fhir/R4/Observation?"+tt.query, handlers.ObservationSearchParser)
			require.Len(t, conds, 1)
			assert.Contains(t, conds[0], tt.operator)
			require.Len(t, args, 1)
		})
	}
}

func TestObservationParser_ParameterizedQueryPreventsInjection(t *testing.T) {
	// Even with malicious input, the parser produces parameterized queries.
	// Use url.Values to safely encode the malicious string.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	maliciousPatient := "Patient/123'; DROP TABLE fhir_resources; --"
	req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Observation", nil)
	q := req.URL.Query()
	q.Set("patient", maliciousPatient)
	req.URL.RawQuery = q.Encode()
	c.Request = req

	conds, args, _, _, _ := handlers.ObservationSearchParser(c)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "$1", "condition uses parameterized placeholder")
	require.Len(t, args, 1)
	assert.Equal(t, maliciousPatient, args[0],
		"malicious input is passed as a parameter, not interpolated into SQL")
}

// --- MedicationRequest Search Parser ---

func TestMedRequestParser_PatientAndStatus(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/MedicationRequest?patient=Patient/456&status=active",
		handlers.MedicationRequestSearchParser,
	)
	require.Len(t, conds, 2)
	require.Len(t, args, 2)
	assert.Equal(t, "Patient/456", args[0])
	assert.Equal(t, "active", args[1])
}

func TestMedRequestParser_IntentParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/MedicationRequest?intent=order",
		handlers.MedicationRequestSearchParser,
	)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "intent")
	assert.Equal(t, "order", args[0])
}

// --- Condition Search Parser ---

func TestConditionParser_ClinicalStatusParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Condition?clinical-status=active",
		handlers.ConditionSearchParser,
	)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "clinicalStatus")
	assert.Equal(t, "active", args[0])
}

func TestConditionParser_OnsetDateWithPrefix(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Condition?onset-date=ge2023-01-01",
		handlers.ConditionSearchParser,
	)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], ">=")
	assert.Equal(t, "2023-01-01", args[0])
}

// --- Practitioner Search Parser ---

func TestPractitionerParser_NameParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Practitioner?name=Sharma",
		handlers.PractitionerSearchParser,
	)
	require.Len(t, conds, 1)
	require.Len(t, args, 1)
	assert.Equal(t, "%Sharma%", args[0], "name search should use LIKE pattern")
}

func TestPractitionerParser_GenderParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Practitioner?gender=female",
		handlers.PractitionerSearchParser,
	)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "gender")
	assert.Equal(t, "female", args[0])
}

func TestPractitionerParser_ActiveParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Practitioner?active=true",
		handlers.PractitionerSearchParser,
	)
	require.Len(t, conds, 1)
	require.Len(t, args, 1)
	assert.Equal(t, true, args[0])
}

// --- Encounter Search Parser ---

func TestEncounterParser_ClassAndStatus(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Encounter?class=AMB&status=finished",
		handlers.EncounterSearchParser,
	)
	require.Len(t, conds, 2)
	require.Len(t, args, 2)
}

// --- DiagnosticReport Search Parser ---

func TestDiagnosticReportParser_PatientAndDate(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/DiagnosticReport?patient=Patient/789&date=lt2024-12-31",
		handlers.DiagnosticReportSearchParser,
	)
	require.Len(t, conds, 2)
	require.Len(t, args, 2)
	assert.Equal(t, "Patient/789", args[0])
	assert.Equal(t, "2024-12-31", args[1])
}

// --- AllergyIntolerance Search Parser ---

func TestAllergyParser_ClinicalStatusAndCriticality(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/AllergyIntolerance?clinical-status=active&criticality=high",
		handlers.AllergyIntoleranceSearchParser,
	)
	require.Len(t, conds, 2)
	assert.Equal(t, "active", args[0])
	assert.Equal(t, "high", args[1])
}

// --- Immunization Search Parser ---

func TestImmunizationParser_VaccineCodeAndDate(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Immunization?vaccine-code=08&date=ge2024-01-01",
		handlers.ImmunizationSearchParser,
	)
	require.Len(t, conds, 2)
	assert.Equal(t, "08", args[0])
	assert.Equal(t, "2024-01-01", args[1])
}

// --- No params = no conditions ---

func TestAllParsers_NoParams_NoConds(t *testing.T) {
	parsers := map[string]handlers.SearchParamParser{
		"Observation":        handlers.ObservationSearchParser,
		"MedicationRequest":  handlers.MedicationRequestSearchParser,
		"Condition":          handlers.ConditionSearchParser,
		"Encounter":          handlers.EncounterSearchParser,
		"Practitioner":       handlers.PractitionerSearchParser,
		"DiagnosticReport":   handlers.DiagnosticReportSearchParser,
		"AllergyIntolerance": handlers.AllergyIntoleranceSearchParser,
		"Immunization":       handlers.ImmunizationSearchParser,
	}

	for name, parser := range parsers {
		t.Run(name, func(t *testing.T) {
			conds, args, _, _, _ := callParser("/fhir/R4/"+name, parser)
			assert.Empty(t, conds, "%s parser with no params should produce no conditions", name)
			assert.Empty(t, args, "%s parser with no params should produce no args", name)
		})
	}
}

// --- Organization Search Parser ---

func TestOrganizationParser_NameParam(t *testing.T) {
	conds, args, _, _, _ := callParser(
		"/fhir/R4/Organization?name=Apollo",
		handlers.OrganizationSearchParser,
	)
	require.Len(t, conds, 1)
	assert.Contains(t, conds[0], "name")
	assert.Equal(t, "%Apollo%", args[0])
}
