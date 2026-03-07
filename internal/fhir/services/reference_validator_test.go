package services

import (
	"context"
	"strings"
	"testing"
)

func TestReferenceValidator_InvalidFormat(t *testing.T) {
	rv := NewReferenceValidator(nil) // nil db is fine since format check fails first

	t.Run("empty ref returns nil", func(t *testing.T) {
		err := rv.ValidateReference(context.Background(), "", "Patient")
		if err != nil {
			t.Fatalf("expected nil error for empty ref, got %v", err)
		}
	})

	t.Run("no slash returns error", func(t *testing.T) {
		err := rv.ValidateReference(context.Background(), "PatientABC", "Patient")
		if err == nil {
			t.Fatal("expected error for ref without slash")
		}
		if !strings.Contains(err.Error(), "Invalid reference format") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("wrong type returns error", func(t *testing.T) {
		err := rv.ValidateReference(context.Background(), "Organization/123", "Patient")
		if err == nil {
			t.Fatal("expected error for type mismatch")
		}
		if !strings.Contains(err.Error(), "type mismatch") {
			t.Fatalf("expected 'type mismatch' in error, got: %v", err)
		}
	})

	t.Run("empty id returns error", func(t *testing.T) {
		err := rv.ValidateReference(context.Background(), "Patient/", "Patient")
		if err == nil {
			t.Fatal("expected error for empty id")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Fatalf("expected 'empty' in error, got: %v", err)
		}
	})
}

func TestReferenceValidator_OptionalEmpty(t *testing.T) {
	rv := NewReferenceValidator(nil)
	err := rv.ValidateOptionalReference(context.Background(), "", "Patient")
	if err != nil {
		t.Fatalf("expected nil for empty optional reference, got %v", err)
	}
}
