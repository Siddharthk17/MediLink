package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// OTPService generates and hashes one-time passwords.
type OTPService interface {
	GenerateOTP() (code string, hash string, err error)
	HashOTP(code string) string
}

type otpService struct{}

// NewOTPService creates a new OTP service.
func NewOTPService() OTPService {
	return &otpService{}
}

// GenerateOTP generates a cryptographically secure 6-digit OTP.
func (s *otpService) GenerateOTP() (string, string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Convert to 6-digit code (100000-999999)
	n := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	code := fmt.Sprintf("%06d", (n%900000)+100000)

	hash := s.HashOTP(code)
	return code, hash, nil
}

// HashOTP returns the SHA-256 hex hash of an OTP code.
func (s *otpService) HashOTP(code string) string {
	h := sha256.Sum256([]byte(code))
	return fmt.Sprintf("%x", h)
}
