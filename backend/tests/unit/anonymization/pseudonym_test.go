package anonymization_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Siddharthk17/MediLink/internal/anonymization"
)

func TestPseudonymize_Deterministic(t *testing.T) {
	a := anonymization.Pseudonymize("patient-abc-123", "salt-xyz")
	b := anonymization.Pseudonymize("patient-abc-123", "salt-xyz")

	assert.Equal(t, a, b, "same input and salt must produce identical pseudonyms")
	assert.NotEqual(t, "patient-abc-123", a, "pseudonym must differ from the original ID")
}

func TestPseudonymize_DifferentSalt(t *testing.T) {
	a := anonymization.Pseudonymize("patient-abc-123", "salt-one")
	b := anonymization.Pseudonymize("patient-abc-123", "salt-two")

	assert.NotEqual(t, a, b, "different salts must produce different pseudonyms")
}

func TestPseudonymizeRef_TypedReference(t *testing.T) {
	result := anonymization.PseudonymizeRef("Patient/abc-123", "salt")

	assert.True(t, strings.HasPrefix(result, "Patient/"),
		"resource type prefix must be preserved")
	assert.NotEqual(t, "Patient/abc-123", result,
		"ID portion must be pseudonymized")

	expected := "Patient/" + anonymization.Pseudonymize("abc-123", "salt")
	assert.Equal(t, expected, result,
		"pseudonymized ref must equal Type/ + Pseudonymize(id, salt)")
}

func TestPseudonymizeRef_UntypedReference(t *testing.T) {
	result := anonymization.PseudonymizeRef("abc-123", "salt")
	assert.Equal(t, "abc-123", result,
		"untyped reference (no slash) must be returned unchanged")
}

func TestPseudonymize_DifferentUUID_DifferentOutput(t *testing.T) {
	a := anonymization.Pseudonymize("uuid-one", "same-salt")
	b := anonymization.Pseudonymize("uuid-two", "same-salt")
	assert.NotEqual(t, a, b, "different UUIDs with same salt must produce different pseudonyms")
}
