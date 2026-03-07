package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESEncryptor_ValidKey(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestNewAESEncryptor_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"too short", "0123456789abcdef"},
		{"too long", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewAESEncryptor(tt.key)
			assert.Error(t, err)
			assert.Nil(t, enc)
		})
	}
}

func TestNewAESEncryptor_InvalidHex(t *testing.T) {
	key := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	enc, err := NewAESEncryptor(key)
	assert.Error(t, err)
	assert.Nil(t, enc)
}

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "Hello, World!"},
		{"empty string", ""},
		{"unicode", "こんにちは世界"},
		{"long text", "This is a longer text that tests encryption of more data than a single block size would normally handle in AES encryption."},
		{"special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.EncryptString(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, ciphertext)

			// Ciphertext should differ from plaintext
			assert.NotEqual(t, []byte(tt.plaintext), ciphertext)

			decrypted, err := enc.DecryptString(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestAESEncryptor_DifferentCiphertexts(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := "same input"
	ct1, err := enc.EncryptString(plaintext)
	require.NoError(t, err)
	ct2, err := enc.EncryptString(plaintext)
	require.NoError(t, err)

	// Due to random nonce, same plaintext should produce different ciphertexts
	assert.NotEqual(t, ct1, ct2)
}

func TestAESEncryptor_DecryptTooShort(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)

	_, err = enc.Decrypt([]byte("short"))
	assert.Error(t, err)
}

func TestAESEncryptor_DecryptCorrupted(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)

	ciphertext, err := enc.EncryptString("test data")
	require.NoError(t, err)

	// Corrupt the ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = enc.DecryptString(ciphertext)
	assert.Error(t, err)
}

func TestAESEncryptor_EncryptDecryptBytes(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	require.NoError(t, err)

	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	ciphertext, err := enc.Encrypt(data)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}
