package search

import (
	"context"
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// NoopSearchClient tests – verify the interface contract
// ---------------------------------------------------------------------------

func TestNoopSearchClient_IndexResource(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	err := client.IndexResource(context.Background(), "Patient", "p-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_DeleteResource(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	err := client.DeleteResource(context.Background(), "Patient", "p-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_SearchByPatient(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	results, err := client.SearchByPatient(context.Background(), "Patient/p-1", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_SearchGlobal(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	results, err := client.SearchGlobal(context.Background(), "fever", "", 10, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_SearchObservationsByCode(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	results, err := client.SearchObservationsByCode(context.Background(), "Patient/p-1", "8480-6")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_EnsureIndices(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	err := client.EnsureIndices(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_Health(t *testing.T) {
	var client SearchClient = &NoopSearchClient{}
	if !client.Health(context.Background()) {
		t.Fatal("expected Health to return true")
	}
}

// ---------------------------------------------------------------------------
// ESClient.indexName tests – verify all 10 resource types and fallback
// ---------------------------------------------------------------------------

func TestESClient_IndexNames(t *testing.T) {
	client := &ESClient{
		indices: map[string]string{
			"Patient":            IndexPatient,
			"Practitioner":       IndexPractitioner,
			"Organization":       IndexOrganization,
			"Encounter":          IndexEncounter,
			"Condition":          IndexCondition,
			"MedicationRequest":  IndexMedicationRequest,
			"Observation":        IndexObservation,
			"DiagnosticReport":   IndexDiagnosticReport,
			"AllergyIntolerance": IndexAllergyIntolerance,
			"Immunization":       IndexImmunization,
		},
	}

	cases := []struct {
		resourceType string
		expected     string
	}{
		{"Patient", IndexPatient},
		{"Practitioner", IndexPractitioner},
		{"Organization", IndexOrganization},
		{"Encounter", IndexEncounter},
		{"Condition", IndexCondition},
		{"MedicationRequest", IndexMedicationRequest},
		{"Observation", IndexObservation},
		{"DiagnosticReport", IndexDiagnosticReport},
		{"AllergyIntolerance", IndexAllergyIntolerance},
		{"Immunization", IndexImmunization},
	}

	for _, tc := range cases {
		t.Run(tc.resourceType, func(t *testing.T) {
			got := client.indexName(tc.resourceType)
			if got != tc.expected {
				t.Errorf("indexName(%q) = %q, want %q", tc.resourceType, got, tc.expected)
			}
		})
	}

	// Unknown resource type falls back to "fhir-{lowercase}s"
	t.Run("UnknownFallback", func(t *testing.T) {
		got := client.indexName("Procedure")
		expected := "fhir-procedures"
		if got != expected {
			t.Errorf("indexName(%q) = %q, want %q", "Procedure", got, expected)
		}
	})
}

// ---------------------------------------------------------------------------
// ESClient.getMappingForResource tests – verify mapping structure
// ---------------------------------------------------------------------------

func TestESClient_GetMappingForResource(t *testing.T) {
	client := &ESClient{}

	// Common fields expected in every mapping.
	commonFields := []string{"resourceType", "id", "patientRef", "status", "date", "lastUpdated"}

	t.Run("Patient", func(t *testing.T) {
		m := client.getMappingForResource("Patient")
		props := extractProperties(t, m)

		for _, f := range commonFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Patient mapping missing common field %q", f)
			}
		}
		patientFields := []string{"family", "given", "birthDate", "gender", "identifier"}
		for _, f := range patientFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Patient mapping missing field %q", f)
			}
		}
	})

	t.Run("Observation", func(t *testing.T) {
		m := client.getMappingForResource("Observation")
		props := extractProperties(t, m)

		for _, f := range commonFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Observation mapping missing common field %q", f)
			}
		}
		obsFields := []string{"code", "codeDisplay", "valueQuantity", "unit", "effectiveDate", "encounterRef", "category"}
		for _, f := range obsFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Observation mapping missing field %q", f)
			}
		}
	})

	t.Run("Encounter", func(t *testing.T) {
		m := client.getMappingForResource("Encounter")
		props := extractProperties(t, m)

		for _, f := range commonFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Encounter mapping missing common field %q", f)
			}
		}
		encFields := []string{"class", "type", "start", "end", "providerRef"}
		for _, f := range encFields {
			if _, ok := props[f]; !ok {
				t.Errorf("Encounter mapping missing field %q", f)
			}
		}
	})

	t.Run("MappingIsValidJSON", func(t *testing.T) {
		for _, rt := range []string{"Patient", "Observation", "Encounter", "Condition"} {
			m := client.getMappingForResource(rt)
			if _, err := json.Marshal(m); err != nil {
				t.Errorf("mapping for %s is not serializable to JSON: %v", rt, err)
			}
		}
	})
}

// extractProperties is a test helper that drills into the mapping structure
// and returns the "properties" map, failing the test if the structure is unexpected.
func extractProperties(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()

	mappings, ok := m["mappings"]
	if !ok {
		t.Fatal("mapping missing top-level 'mappings' key")
	}
	mappingsMap, ok := mappings.(map[string]interface{})
	if !ok {
		t.Fatal("'mappings' is not a map")
	}
	props, ok := mappingsMap["properties"]
	if !ok {
		t.Fatal("mapping missing 'properties' key")
	}
	propsMap, ok := props.(map[string]interface{})
	if !ok {
		t.Fatal("'properties' is not a map")
	}
	return propsMap
}
