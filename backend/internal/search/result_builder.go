package search

// SearchHit represents a single search result with optional highlights.
type SearchHit struct {
	ResourceType string
	ResourceID   string
	Score        float64
	Source       map[string]interface{}
	Highlights   map[string][]string
}

// BuildSearchBundle converts search hits into a FHIR Bundle searchset response.
func BuildSearchBundle(results []SearchHit, total int, query string) map[string]interface{} {
	entries := make([]interface{}, 0, len(results))
	for _, hit := range results {
		entry := map[string]interface{}{
			"fullUrl":  hit.ResourceType + "/" + hit.ResourceID,
			"resource": hit.Source,
			"search": map[string]interface{}{
				"mode":  "match",
				"score": hit.Score,
			},
		}
		if len(hit.Highlights) > 0 {
			entry["search"].(map[string]interface{})["extension"] = []map[string]interface{}{
				{"url": "highlight", "valueString": hit.Highlights},
			}
		}
		entries = append(entries, entry)
	}

	return map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        total,
		"entry":        entries,
	}
}
