package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/auth"
)

func TestValidatePasswordStrength_Valid(t *testing.T) {
	violations := auth.ValidatePasswordStrength("Str0ng!Pass#99", "testuser")
	assert.Empty(t, violations, "a strong password should produce no violations")
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	violations := auth.ValidatePasswordStrength("Sh0r!t", "user")
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations, "Password must be at least 8 characters")
}

func TestValidatePasswordStrength_NoUpper(t *testing.T) {
	violations := auth.ValidatePasswordStrength("lowercase1!", "user")
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations, "Password must contain at least one uppercase letter")
}

func TestValidatePasswordStrength_NoSpecial(t *testing.T) {
	violations := auth.ValidatePasswordStrength("NoSpecial1abc", "user")
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations, "Password must contain at least one special character")
}

func TestValidatePasswordStrength_CommonPassword(t *testing.T) {
	violations := auth.ValidatePasswordStrength("Password1!", "user")
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations, "Password is too common")
}

func TestBcryptHash_Verify(t *testing.T) {
	password := "MySecure!Pass99"

	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	t.Run("correct password verifies", func(t *testing.T) {
		assert.True(t, auth.VerifyPassword(hash, password))
	})

	t.Run("wrong password fails", func(t *testing.T) {
		assert.False(t, auth.VerifyPassword(hash, "WrongPassword1!"))
	})
}

func TestBcryptHash_DifferentEachTime(t *testing.T) {
	password := "SameInput!Pass1"

	hash1, err := auth.HashPassword(password)
	require.NoError(t, err)

	hash2, err := auth.HashPassword(password)
	require.NoError(t, err)

	t.Run("hashes differ due to random salt", func(t *testing.T) {
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("both hashes verify the original password", func(t *testing.T) {
		assert.True(t, auth.VerifyPassword(hash1, password))
		assert.True(t, auth.VerifyPassword(hash2, password))
	})
}
