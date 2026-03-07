// Package integration_test provides integration tests that require a real
// PostgreSQL database, Redis instance, and Elasticsearch cluster.
package integration_test

import (
	"fmt"
	"os"
	"testing"
)

// TestMain controls integration test lifecycle.
// If DATABASE_URL is not set, all integration tests are skipped so that
// unit tests always run in CI without a real database.
func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("DATABASE_URL not set — skipping integration tests")
		os.Exit(0)
	}

	// TODO: run migrations up before tests, down after tests
	os.Exit(m.Run())
}
