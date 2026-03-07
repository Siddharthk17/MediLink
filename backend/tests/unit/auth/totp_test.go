package auth_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/pkg/crypto"

	"github.com/Siddharthk17/MediLink/internal/auth"
)

// 64 hex chars = 32-byte AES-256 key
const testAESHexKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func setupTOTPService(t *testing.T) auth.TOTPService {
	t.Helper()
	enc, err := crypto.NewAESEncryptor(testAESHexKey)
	require.NoError(t, err)
	return auth.NewTOTPService(enc)
}

func TestTOTPSetup_GeneratesSecret(t *testing.T) {
	svc := setupTOTPService(t)

	secret, _, err := svc.GenerateSecret("doctor@medilink.health")
	require.NoError(t, err)
	assert.NotEmpty(t, secret, "secret must be non-empty")
	assert.GreaterOrEqual(t, len(secret), 16, "secret should be at least 16 chars (base32)")
}

func TestTOTPSetup_QRCodeGenerated(t *testing.T) {
	svc := setupTOTPService(t)

	_, qrBase64, err := svc.GenerateSecret("doctor@medilink.health")
	require.NoError(t, err)

	t.Run("has data URI prefix", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(qrBase64, "data:image/png;base64,"))
	})

	t.Run("base64 decodes to valid PNG", func(t *testing.T) {
		b64Data := strings.TrimPrefix(qrBase64, "data:image/png;base64,")
		decoded, err := base64.StdEncoding.DecodeString(b64Data)
		require.NoError(t, err)
		assert.NotEmpty(t, decoded)

		// PNG magic bytes: 0x89 P N G
		require.GreaterOrEqual(t, len(decoded), 4)
		assert.Equal(t, byte(0x89), decoded[0])
		assert.Equal(t, byte('P'), decoded[1])
		assert.Equal(t, byte('N'), decoded[2])
		assert.Equal(t, byte('G'), decoded[3])
	})
}

func TestTOTPVerify_ValidCode(t *testing.T) {
	svc := setupTOTPService(t)

	secret, _, err := svc.GenerateSecret("user@medilink.health")
	require.NoError(t, err)

	// Generate current valid code using the OTP library
	code, err := totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:     0,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	assert.True(t, svc.ValidateCode(secret, code), "current TOTP code should be valid")
}

func TestTOTPVerify_ExpiredCode(t *testing.T) {
	svc := setupTOTPService(t)

	secret, _, err := svc.GenerateSecret("user@medilink.health")
	require.NoError(t, err)

	// Generate a code from 5 minutes ago — well outside the ±1 window
	code, err := totp.GenerateCodeCustom(secret, time.Now().Add(-5*time.Minute), totp.ValidateOpts{
		Period:    30,
		Skew:     0,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	assert.False(t, svc.ValidateCode(secret, code), "code from 5 min ago should be rejected")
}

func TestTOTPVerify_ClockSkew(t *testing.T) {
	svc := setupTOTPService(t)

	secret, _, err := svc.GenerateSecret("user@medilink.health")
	require.NoError(t, err)

	// Compute the previous period's midpoint to guarantee a different period
	now := time.Now()
	currentPeriod := now.Unix() / 30
	prevPeriodMid := time.Unix((currentPeriod-1)*30+15, 0)

	code, err := totp.GenerateCodeCustom(secret, prevPeriodMid, totp.ValidateOpts{
		Period:    30,
		Skew:     0,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	assert.True(t, svc.ValidateCode(secret, code),
		"code from the previous 30s window should be accepted (Skew=1)")
}

func TestTOTPVerify_BackupCode(t *testing.T) {
	svc := setupTOTPService(t)

	codes, hashes, err := svc.GenerateBackupCodes()
	require.NoError(t, err)
	require.Len(t, codes, 10)
	require.Len(t, hashes, 10)

	t.Run("valid backup code matches", func(t *testing.T) {
		idx, valid := svc.ValidateBackupCode(codes[0], hashes)
		assert.True(t, valid)
		assert.Equal(t, 0, idx)
	})

	t.Run("wrong code does not match", func(t *testing.T) {
		_, valid := svc.ValidateBackupCode("ZZZZZZZZ", hashes)
		assert.False(t, valid)
	})
}

func TestTOTPVerify_BackupCodeUsedTwice(t *testing.T) {
	svc := setupTOTPService(t)

	codes, hashes, err := svc.GenerateBackupCodes()
	require.NoError(t, err)

	// First use: succeeds
	idx, valid := svc.ValidateBackupCode(codes[0], hashes)
	require.True(t, valid)

	// Simulate consumption: remove the matched hash
	remaining := make([]string, 0, len(hashes)-1)
	remaining = append(remaining, hashes[:idx]...)
	remaining = append(remaining, hashes[idx+1:]...)

	// Second use: should fail because the hash was consumed
	_, valid = svc.ValidateBackupCode(codes[0], remaining)
	assert.False(t, valid, "backup code must not validate after its hash is consumed")
}
