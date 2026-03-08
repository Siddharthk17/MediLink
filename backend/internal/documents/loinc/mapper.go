// Package loinc provides LOINC code lookup for lab test names.
package loinc

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// LOINCResult represents a LOINC code mapping result.
type LOINCResult struct {
	Code       string
	Display    string
	Unit       string
	NormalLow  float64
	NormalHigh float64
}

// LOINCMapper defines the interface for LOINC code lookup.
type LOINCMapper interface {
	Lookup(ctx context.Context, testName string) (*LOINCResult, bool)
	BulkLookup(ctx context.Context, testNames []string) map[string]*LOINCResult
}

// PostgresLOINCMapper implements LOINCMapper using PostgreSQL.
type PostgresLOINCMapper struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewLOINCMapper creates a new PostgresLOINCMapper.
func NewLOINCMapper(db *sqlx.DB, logger zerolog.Logger) *PostgresLOINCMapper {
	return &PostgresLOINCMapper{db: db, logger: logger}
}

// Lookup finds a LOINC code for a test name using exact then fuzzy match.
func (m *PostgresLOINCMapper) Lookup(ctx context.Context, testName string) (*LOINCResult, bool) {
	lower := strings.ToLower(strings.TrimSpace(testName))
	if lower == "" {
		return nil, false
	}

	// Strategy 1: Exact match on test_name_lower
	var row struct {
		LOINCCode    string  `db:"loinc_code"`
		LOINCDisplay string  `db:"loinc_display"`
		Unit         *string `db:"unit"`
		NormalLow    *float64 `db:"normal_low"`
		NormalHigh   *float64 `db:"normal_high"`
	}

	err := m.db.GetContext(ctx, &row,
		`SELECT loinc_code, loinc_display, unit, normal_low, normal_high
		 FROM loinc_mappings WHERE test_name_lower = $1 LIMIT 1`, lower)
	if err == nil {
		result := &LOINCResult{
			Code:    row.LOINCCode,
			Display: row.LOINCDisplay,
		}
		if row.Unit != nil {
			result.Unit = *row.Unit
		}
		if row.NormalLow != nil {
			result.NormalLow = *row.NormalLow
		}
		if row.NormalHigh != nil {
			result.NormalHigh = *row.NormalHigh
		}
		return result, true
	}

	// Strategy 2: Full-text search using GIN index
	tsQuery := strings.Join(strings.Fields(lower), " & ")
	err = m.db.GetContext(ctx, &row,
		`SELECT loinc_code, loinc_display, unit, normal_low, normal_high
		 FROM loinc_mappings, to_tsquery('english', $1) query
		 WHERE to_tsvector('english', test_name) @@ query
		 ORDER BY ts_rank(to_tsvector('english', test_name), query) DESC
		 LIMIT 1`, tsQuery)
	if err == nil {
		result := &LOINCResult{
			Code:    row.LOINCCode,
			Display: row.LOINCDisplay,
		}
		if row.Unit != nil {
			result.Unit = *row.Unit
		}
		if row.NormalLow != nil {
			result.NormalLow = *row.NormalLow
		}
		if row.NormalHigh != nil {
			result.NormalHigh = *row.NormalHigh
		}
		return result, true
	}

	return nil, false
}

// BulkLookup looks up multiple test names at once.
func (m *PostgresLOINCMapper) BulkLookup(ctx context.Context, testNames []string) map[string]*LOINCResult {
	results := make(map[string]*LOINCResult, len(testNames))
	for _, name := range testNames {
		if result, found := m.Lookup(ctx, name); found {
			results[name] = result
		}
	}
	return results
}
