package search

import (
	"time"
)

// SearchFilters holds role-based and temporal filters for ES queries.
type SearchFilters struct {
	ActorRole            string
	PatientFHIRID        string     // for patient role — always their own ID
	PatientRef           string     // from ?patient= param
	ConsentedPatientRefs []string   // for physician role — all consented patient refs
	DateFrom             *time.Time
	DateTo               *time.Time
}

var indexByType = map[string]string{
	"Patient":            "fhir-patients",
	"Practitioner":       "fhir-practitioners",
	"Organization":       "fhir-organizations",
	"Encounter":          "fhir-encounters",
	"Condition":          "fhir-conditions",
	"MedicationRequest":  "fhir-medication-requests",
	"Observation":        "fhir-observations",
	"DiagnosticReport":   "fhir-diagnostic-reports",
	"AllergyIntolerance": "fhir-allergy-intolerances",
	"Immunization":       "fhir-immunizations",
}

// BuildQuery constructs an ES multi-index query with role-based filters,
// fuzzy matching, highlighting, and pagination.
func BuildQuery(q string, count, offset int, f SearchFilters) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query": q,
							"fields": []string{
								"family^3", "given^2",
								"codeDisplay^2", "medicationDisplay^2",
								"type", "status",
							},
							"type":                 "best_fields",
							"fuzziness":            "AUTO",
							"minimum_should_match": "75%",
						},
					},
				},
				"filter": buildFilters(f),
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"family":            map[string]interface{}{},
				"given":             map[string]interface{}{},
				"codeDisplay":       map[string]interface{}{},
				"medicationDisplay": map[string]interface{}{},
			},
			"pre_tags":  []string{"<em>"},
			"post_tags": []string{"</em>"},
		},
		"size": count,
		"from": offset,
		"sort": []interface{}{
			map[string]interface{}{"_score": "desc"},
			map[string]interface{}{"lastUpdated": "desc"},
		},
		"_source": true,
	}
}

// IndicesForTypes maps FHIR resource type names to ES index names.
// An empty slice returns all known indices.
func IndicesForTypes(types []string) []string {
	if len(types) == 0 {
		all := make([]string, 0, len(indexByType))
		for _, v := range indexByType {
			all = append(all, v)
		}
		return all
	}
	result := make([]string, 0, len(types))
	for _, t := range types {
		if idx, ok := indexByType[t]; ok {
			result = append(result, idx)
		}
	}
	return result
}

func buildFilters(f SearchFilters) []interface{} {
	var filters []interface{}

	switch f.ActorRole {
	case "patient":
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"patientRef": "Patient/" + f.PatientFHIRID},
		})
	case "physician":
		if f.PatientRef != "" {
			filters = append(filters, map[string]interface{}{
				"term": map[string]interface{}{"patientRef": f.PatientRef},
			})
		} else if len(f.ConsentedPatientRefs) > 0 {
			filters = append(filters, map[string]interface{}{
				"terms": map[string]interface{}{"patientRef": f.ConsentedPatientRefs},
			})
		}
	case "admin":
		if f.PatientRef != "" {
			filters = append(filters, map[string]interface{}{
				"term": map[string]interface{}{"patientRef": f.PatientRef},
			})
		}
	}

	if f.DateFrom != nil || f.DateTo != nil {
		r := map[string]interface{}{}
		if f.DateFrom != nil {
			r["gte"] = f.DateFrom.Format(time.RFC3339)
		}
		if f.DateTo != nil {
			r["lte"] = f.DateTo.Format(time.RFC3339)
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{"date": r},
		})
	}

	return filters
}
