// Package crypto provides AES-256-GCM encryption for PII data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Encryptor defines the interface for encryption operations.
type Encryptor interface {
	// Encrypt encrypts plaintext bytes and returns ciphertext (nonce prepended).
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt decrypts ciphertext bytes (extracts nonce, decrypts remainder).
	Decrypt(ciphertext []byte) ([]byte, error)
	// EncryptString encrypts a string and returns ciphertext bytes.
	EncryptString(s string) ([]byte, error)
	// DecryptString decrypts ciphertext bytes and returns the original string.
	DecryptString(b []byte) (string, error)
}

// AESEncryptor implements Encryptor using AES-256-GCM.
type AESEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor creates a new AES-256-GCM encryptor from a hex-encoded key.
// The key must be exactly 64 hex characters (32 bytes).
func NewAESEncryptor(hexKey string) (*AESEncryptor, error) {
	if len(hexKey) != 64 {
		return nil, fmt.Errorf("encryption key must be exactly 64 hex characters (32 bytes), got %d", len(hexKey))
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex in encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &AESEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns nonce+ciphertext.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt extracts the nonce from the first 12 bytes and decrypts the remainder.
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns ciphertext bytes.
func (e *AESEncryptor) EncryptString(s string) ([]byte, error) {
	return e.Encrypt([]byte(s))
}

// DecryptString decrypts ciphertext bytes and returns the original string.
func (e *AESEncryptor) DecryptString(b []byte) (string, error) {
	plaintext, err := e.Decrypt(b)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
