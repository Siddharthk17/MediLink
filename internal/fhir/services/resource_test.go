package services

import (
	"encoding/json"
	"testing"

	"github.com/Siddharthk17/medilink/internal/fhir/repository"
)

func TestExtractSubjectRef(t *testing.T) {
	t.Run("valid subject", func(t *testing.T) {
		data := json.RawMessage(`{"subject":{"reference":"Patient/123"}}`)
		ref := ExtractSubjectRef(data)
		if ref == nil || *ref != "Patient/123" {
			t.Fatalf("expected Patient/123, got %v", ref)
		}
	})

	t.Run("missing subject", func(t *testing.T) {
		data := json.RawMessage(`{"resourceType":"Observation"}`)
		ref := ExtractSubjectRef(data)
		if ref != nil {
			t.Fatalf("expected nil, got %v", *ref)
		}
	})
}

func TestExtractPatientFieldRef(t *testing.T) {
	t.Run("valid patient", func(t *testing.T) {
		data := json.RawMessage(`{"patient":{"reference":"Patient/456"}}`)
		ref := ExtractPatientFieldRef(data)
		if ref == nil || *ref != "Patient/456" {
			t.Fatalf("expected Patient/456, got %v", ref)
		}
	})

	t.Run("missing patient", func(t *testing.T) {
		data := json.RawMessage(`{"resourceType":"Immunization"}`)
		ref := ExtractPatientFieldRef(data)
		if ref != nil {
			t.Fatalf("expected nil, got %v", *ref)
		}
	})
}

func TestExtractStatusField(t *testing.T) {
	t.Run("has status", func(t *testing.T) {
		data := json.RawMessage(`{"status":"active"}`)
		status := ExtractStatusField(data)
		if status != "active" {
			t.Fatalf("expected 'active', got '%s'", status)
		}
	})

	t.Run("no status", func(t *testing.T) {
		data := json.RawMessage(`{"resourceType":"Patient"}`)
		status := ExtractStatusField(data)
		if status != "" {
			t.Fatalf("expected empty string, got '%s'", status)
		}
	})
}

func TestGenericSearchBundle(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		bundle := GenericSearchBundle(nil, 0, "Observation", 20, 0, "http://localhost")
		if bundle["total"] != 0 {
			t.Fatalf("expected total 0, got %v", bundle["total"])
		}
		if bundle["type"] != "searchset" {
			t.Fatalf("expected type 'searchset', got %v", bundle["type"])
		}
		if bundle["resourceType"] != "Bundle" {
			t.Fatalf("expected resourceType 'Bundle', got %v", bundle["resourceType"])
		}
		entries := bundle["entry"].([]map[string]interface{})
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("with results", func(t *testing.T) {
		resources := []*repository.FHIRResource{
			{ResourceID: "obs-1", Data: json.RawMessage(`{"resourceType":"Observation","id":"obs-1"}`)},
		}
		bundle := GenericSearchBundle(resources, 1, "Observation", 20, 0, "http://localhost")
		if bundle["total"] != 1 {
			t.Fatalf("expected total 1, got %v", bundle["total"])
		}
		entries := bundle["entry"].([]map[string]interface{})
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("pagination next link", func(t *testing.T) {
		bundle := GenericSearchBundle(nil, 50, "Encounter", 20, 0, "http://localhost")
		links := bundle["link"].([]map[string]string)
		hasNext := false
		for _, l := range links {
			if l["relation"] == "next" {
				hasNext = true
			}
		}
		if !hasNext {
			t.Fatal("expected next link for paginated results")
		}
	})

	t.Run("pagination previous link", func(t *testing.T) {
		bundle := GenericSearchBundle(nil, 50, "Encounter", 20, 20, "http://localhost")
		links := bundle["link"].([]map[string]string)
		hasPrev := false
		for _, l := range links {
			if l["relation"] == "previous" {
				hasPrev = true
			}
		}
		if !hasPrev {
			t.Fatal("expected previous link when offset > 0")
		}
	})

	t.Run("no next link on last page", func(t *testing.T) {
		bundle := GenericSearchBundle(nil, 10, "Patient", 20, 0, "http://localhost")
		links := bundle["link"].([]map[string]string)
		for _, l := range links {
			if l["relation"] == "next" {
				t.Fatal("should not have next link when all results fit in one page")
			}
		}
	})
}
