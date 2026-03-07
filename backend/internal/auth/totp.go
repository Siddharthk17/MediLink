package auth

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/Siddharthk17/MediLink/pkg/crypto"
	"golang.org/x/crypto/bcrypt"
)

// TOTPService handles TOTP MFA operations.
type TOTPService interface {
	GenerateSecret(email string) (secret string, qrCodeBase64 string, err error)
	ValidateCode(secret string, code string) bool
	GenerateBackupCodes() (codes []string, hashes []string, err error)
	ValidateBackupCode(code string, hashes []string) (matched int, valid bool)
}

type totpService struct {
	encryptor crypto.Encryptor
}

// NewTOTPService creates a new TOTP service.
func NewTOTPService(encryptor crypto.Encryptor) TOTPService {
	return &totpService{encryptor: encryptor}
}

func (s *totpService) GenerateSecret(email string) (string, string, error) {
	// Generate 20 random bytes for the TOTP secret
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Create otpauth:// URI
	key, err := otp.NewKeyFromURL(
		fmt.Sprintf("otpauth://totp/MediLink:%s?secret=%s&issuer=MediLink&algorithm=SHA1&digits=6&period=30", email, secret),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create TOTP key: %w", err)
	}

	// Generate QR code as PNG
	png, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	qrBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	return secret, qrBase64, nil
}

func (s *totpService) ValidateCode(secret string, code string) bool {
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:     1, // ±1 window = 30 seconds tolerance
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return valid
}

func (s *totpService) GenerateBackupCodes() ([]string, []string, error) {
	codes := make([]string, 10)
	hashes := make([]string, 10)

	for i := 0; i < 10; i++ {
		// Generate 8-char hex code using crypto/rand
		b := make([]byte, 5) // 5 bytes = 10 hex chars, take 8
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = fmt.Sprintf("%X", b)[:8]

		hash, err := bcrypt.GenerateFromPassword([]byte(codes[i]), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		hashes[i] = string(hash)
	}

	return codes, hashes, nil
}

func (s *totpService) ValidateBackupCode(code string, hashes []string) (int, bool) {
	for i, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil {
			return i, true
		}
	}
	return -1, false
}
