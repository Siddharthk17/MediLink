package search_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Siddharthk17/MediLink/pkg/search"
)

// search.NoopSearchClient tests – verify the interface contract

func TestNoopSearchClient_IndexResource(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	err := client.IndexResource(context.Background(), "Patient", "p-1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_DeleteResource(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	err := client.DeleteResource(context.Background(), "Patient", "p-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_SearchByPatient(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	results, err := client.SearchByPatient(context.Background(), "Patient/p-1", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_SearchGlobal(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	results, err := client.SearchGlobal(context.Background(), "fever", "", 10, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_SearchObservationsByCode(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	results, err := client.SearchObservationsByCode(context.Background(), "Patient/p-1", "8480-6")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestNoopSearchClient_EnsureIndices(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	err := client.EnsureIndices(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNoopSearchClient_Health(t *testing.T) {
	var client search.SearchClient = &search.NoopSearchClient{}
	if !client.Health(context.Background()) {
		t.Fatal("expected Health to return true")
	}
}

// search.ESClient index constant tests – verify all 10 resource type constants exist

func TestESClient_IndexConstants(t *testing.T) {
	// Verify all 10 FHIR resource index constants are defined
	constants := map[string]string{
		"Patient":            search.IndexPatient,
		"Practitioner":       search.IndexPractitioner,
		"Organization":       search.IndexOrganization,
		"Encounter":          search.IndexEncounter,
		"Condition":          search.IndexCondition,
		"MedicationRequest":  search.IndexMedicationRequest,
		"Observation":        search.IndexObservation,
		"DiagnosticReport":   search.IndexDiagnosticReport,
		"AllergyIntolerance": search.IndexAllergyIntolerance,
		"Immunization":       search.IndexImmunization,
	}

	for resourceType, indexName := range constants {
		t.Run(resourceType, func(t *testing.T) {
			if indexName == "" {
				t.Errorf("index constant for %q is empty", resourceType)
			}
			// All indices should follow the fhir- prefix pattern
			if len(indexName) < 5 {
				t.Errorf("index constant for %q is suspiciously short: %q", resourceType, indexName)
			}
		})
	}
}

// search.SearchClient interface compliance – verify both implementations

func TestESClient_ImplementsSearchClient(t *testing.T) {
	// Verify NoopSearchClient satisfies the SearchClient interface at compile time.
	var _ search.SearchClient = &search.NoopSearchClient{}

	// ESClient also satisfies it (cannot instantiate without ES, but the type assertion compiles).
	var _ search.SearchClient = (*search.ESClient)(nil)
}

// search.SearchResult struct tests – verify field access

func TestSearchResult_Fields(t *testing.T) {
	result := search.SearchResult{
		ResourceType: "Patient",
		ResourceID:   "p-123",
		Data:         map[string]interface{}{"resourceType": "Patient", "id": "p-123"},
	}

	if result.ResourceType != "Patient" {
		t.Errorf("expected ResourceType 'Patient', got %q", result.ResourceType)
	}
	if result.ResourceID != "p-123" {
		t.Errorf("expected ResourceID 'p-123', got %q", result.ResourceID)
	}
	if len(result.Data) == 0 {
		t.Error("expected non-empty Data")
	}
}

// Index naming convention tests

func TestIndexConstants_NamingConvention(t *testing.T) {
	indices := []struct {
		name  string
		value string
	}{
		{"IndexPatient", search.IndexPatient},
		{"IndexPractitioner", search.IndexPractitioner},
		{"IndexOrganization", search.IndexOrganization},
		{"IndexEncounter", search.IndexEncounter},
		{"IndexCondition", search.IndexCondition},
		{"IndexMedicationRequest", search.IndexMedicationRequest},
		{"IndexObservation", search.IndexObservation},
		{"IndexDiagnosticReport", search.IndexDiagnosticReport},
		{"IndexAllergyIntolerance", search.IndexAllergyIntolerance},
		{"IndexImmunization", search.IndexImmunization},
	}

	for _, idx := range indices {
		t.Run(idx.name, func(t *testing.T) {
			// All index names must start with "fhir-"
			if len(idx.value) < 5 || idx.value[:5] != "fhir-" {
				t.Errorf("index %s = %q, expected fhir- prefix", idx.name, idx.value)
			}
		})
	}
}

func TestNoopSearchClient_AllMethodsReturnNil(t *testing.T) {
	client := &search.NoopSearchClient{}
	ctx := context.Background()

	t.Run("IndexResource", func(t *testing.T) {
		err := client.IndexResource(ctx, "Patient", "p-1", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("DeleteResource", func(t *testing.T) {
		err := client.DeleteResource(ctx, "Patient", "p-1")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("SearchByPatient", func(t *testing.T) {
		res, err := client.SearchByPatient(ctx, "Patient/p-1", nil)
		if err != nil || res != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", res, err)
		}
	})
	t.Run("SearchGlobal", func(t *testing.T) {
		res, err := client.SearchGlobal(ctx, "test", "", 10, 0)
		if err != nil || res != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", res, err)
		}
	})
	t.Run("SearchObservationsByCode", func(t *testing.T) {
		res, err := client.SearchObservationsByCode(ctx, "Patient/p-1", "8480-6")
		if err != nil || res != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", res, err)
		}
	})
	t.Run("EnsureIndices", func(t *testing.T) {
		err := client.EnsureIndices(ctx)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("Health", func(t *testing.T) {
		if !client.Health(ctx) {
			t.Fatal("expected true")
		}
	})
}
