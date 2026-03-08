package anonymization

import (
	"strconv"
	"time"
)

// AgeBand generalises a four-digit birth year into a 10-year age band.
// Unknown or future years return "unknown".
func AgeBand(birthYear string) string {
	year, err := strconv.Atoi(birthYear)
	if err != nil {
		return "unknown"
	}
	age := time.Now().Year() - year
	switch {
	case age < 0:
		return "unknown"
	case age < 18:
		return "0-17"
	case age < 30:
		return "18-29"
	case age < 40:
		return "30-39"
	case age < 50:
		return "40-49"
	case age < 60:
		return "50-59"
	case age < 70:
		return "60-69"
	default:
		return "70+"
	}
}

// AnonymizedRecord holds one anonymized FHIR resource together with the
// quasi-identifiers needed for k-anonymity grouping.
type AnonymizedRecord struct {
	ResourceType         string
	ResourceID           string
	Data                 map[string]interface{}
	AgeBand              string
	Gender               string
	PrimaryConditionCode string
}

// KAnonStats summarises the effect of k-anonymity suppression on a dataset.
type KAnonStats struct {
	TotalRecords      int `json:"totalRecords"`
	KeptRecords       int `json:"keptRecords"`
	SuppressedRecords int `json:"suppressedRecords"`
	SuppressedGroups  int `json:"suppressedGroups"`
	SmallestGroupKept int `json:"smallestGroupKept"`
	KValue            int `json:"kValue"`
}

// ApplyKAnonymity removes equivalence classes with fewer than k members.
func ApplyKAnonymity(records []AnonymizedRecord, k int) ([]AnonymizedRecord, KAnonStats) {
	type groupKey struct{ Age, Gender, Condition string }
	groups := make(map[groupKey][]int)
	for i, r := range records {
		key := groupKey{r.AgeBand, r.Gender, r.PrimaryConditionCode}
		groups[key] = append(groups[key], i)
	}

	keep := make(map[int]bool)
	stats := KAnonStats{TotalRecords: len(records), KValue: k}
	smallest := int(^uint(0) >> 1) // max int

	for _, idxs := range groups {
		if len(idxs) >= k {
			for _, i := range idxs {
				keep[i] = true
			}
			if len(idxs) < smallest {
				smallest = len(idxs)
			}
		} else {
			stats.SuppressedGroups++
			stats.SuppressedRecords += len(idxs)
		}
	}

	kept := make([]AnonymizedRecord, 0, len(keep))
	for i, r := range records {
		if keep[i] {
			kept = append(kept, r)
		}
	}
	stats.KeptRecords = len(kept)
	if smallest < int(^uint(0)>>1) {
		stats.SmallestGroupKept = smallest
	}
	return kept, stats
}
