// Package anonymization provides deterministic pseudonymization,
// k-anonymity suppression, and research data export capabilities.
package anonymization

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Pseudonymize produces a deterministic UUID-shaped pseudonym from a real UUID
// and a per-export salt. The same (realUUID, salt) pair always yields the same
// output, but different salts produce unlinkable pseudonyms.
func Pseudonymize(realUUID, exportSalt string) string {
	h := sha256.New()
	h.Write([]byte(realUUID + ":" + exportSalt))
	b := h.Sum(nil)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// PseudonymizeRef pseudonymizes a FHIR reference such as "Patient/abc-123".
// Non-reference strings are returned unchanged.
func PseudonymizeRef(ref, exportSalt string) string {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return ref
	}
	return parts[0] + "/" + Pseudonymize(parts[1], exportSalt)
}

// GenerateExportSalt returns a cryptographically random 32-byte hex string
// used as the per-export pseudonymization salt.
func GenerateExportSalt() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
