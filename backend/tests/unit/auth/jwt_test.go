package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/auth"
)

const testJWTSecret = "test-super-secret-key-for-jwt-256bit!"

func setupJWTService(t *testing.T) (auth.JWTService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })

	return auth.NewJWTService(testJWTSecret, client), mr
}

func TestGenerateAccessToken_Physician(t *testing.T) {
	svc, _ := setupJWTService(t)

	tokenStr, jti, err := svc.GenerateAccessToken("doc-123", "physician", "org-456", true)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)
	assert.NotEmpty(t, jti)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	t.Run("has correct user ID", func(t *testing.T) {
		assert.Equal(t, "doc-123", claims.UserID)
	})
	t.Run("has correct role", func(t *testing.T) {
		assert.Equal(t, "physician", claims.Role)
	})
	t.Run("has correct org_id", func(t *testing.T) {
		assert.Equal(t, "org-456", claims.OrgID)
	})
	t.Run("has totp_verified true", func(t *testing.T) {
		assert.True(t, claims.TOTPVerified)
	})
	t.Run("token type is access", func(t *testing.T) {
		assert.Equal(t, "access", claims.TokenType)
	})
	t.Run("jti matches", func(t *testing.T) {
		assert.Equal(t, jti, claims.ID)
	})
}

func TestGenerateAccessToken_Patient(t *testing.T) {
	svc, _ := setupJWTService(t)

	tokenStr, _, err := svc.GenerateAccessToken("patient-789", "patient", "org-456", true)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	t.Run("role is patient", func(t *testing.T) {
		assert.Equal(t, "patient", claims.Role)
	})
	t.Run("totp_verified is true", func(t *testing.T) {
		assert.True(t, claims.TOTPVerified)
	})
	t.Run("subject matches user ID", func(t *testing.T) {
		assert.Equal(t, "patient-789", claims.Subject)
	})
}

func TestValidateToken_Valid(t *testing.T) {
	svc, _ := setupJWTService(t)

	tokenStr, jti, err := svc.GenerateAccessToken("user-1", "admin", "org-1", false)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	t.Run("user ID parsed", func(t *testing.T) {
		assert.Equal(t, "user-1", claims.UserID)
	})
	t.Run("subject matches", func(t *testing.T) {
		assert.Equal(t, "user-1", claims.Subject)
	})
	t.Run("jti matches", func(t *testing.T) {
		assert.Equal(t, jti, claims.ID)
	})
	t.Run("issuer is medilink", func(t *testing.T) {
		assert.Equal(t, "medilink.health", claims.Issuer)
	})
	t.Run("audience contains api", func(t *testing.T) {
		assert.Contains(t, claims.Audience, "medilink.api")
	})
}

func TestValidateToken_Expired(t *testing.T) {
	svc, _ := setupJWTService(t)

	past := time.Now().Add(-3 * time.Hour)
	claims := auth.AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "medilink.health",
			Subject:   "user-expired",
			Audience:  jwt.ClaimStrings{"medilink.api"},
			ExpiresAt: jwt.NewNumericDate(past.Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(past),
			IssuedAt:  jwt.NewNumericDate(past),
			ID:        "expired-jti",
		},
		UserID:       "user-expired",
		Role:         "patient",
		OrgID:        "org-1",
		TOTPVerified: true,
		TokenType:    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token")
}

func TestValidateToken_WrongAlgorithm(t *testing.T) {
	svc, _ := setupJWTService(t)

	claims := jwt.MapClaims{
		"iss":  "medilink.health",
		"aud":  []string{"medilink.api"},
		"sub":  "user-attacker",
		"uid":  "user-attacker",
		"role": "admin",
		"type": "access",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"nbf":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err, "tokens with 'none' algorithm must be rejected")
}

func TestValidateToken_WrongIssuer(t *testing.T) {
	svc, _ := setupJWTService(t)

	now := time.Now()
	claims := auth.AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "evil.attacker.com",
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{"medilink.api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        "wrong-issuer-jti",
		},
		UserID:       "user-1",
		Role:         "admin",
		OrgID:        "org-1",
		TOTPVerified: true,
		TokenType:    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	_, err = svc.ValidateAccessToken(tokenStr)
	assert.Error(t, err, "token with wrong issuer must be rejected")
}

func TestValidateToken_Blacklisted(t *testing.T) {
	svc, _ := setupJWTService(t)
	ctx := context.Background()

	tokenStr, jti, err := svc.GenerateAccessToken("user-1", "physician", "org-1", true)
	require.NoError(t, err)

	// Blacklist the token
	err = svc.BlacklistToken(ctx, jti, time.Hour)
	require.NoError(t, err)

	t.Run("IsBlacklisted returns true", func(t *testing.T) {
		blacklisted, err := svc.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("non-blacklisted token returns false", func(t *testing.T) {
		blacklisted, err := svc.IsBlacklisted(ctx, "random-jti")
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("structural validation still passes", func(t *testing.T) {
		claims, err := svc.ValidateAccessToken(tokenStr)
		require.NoError(t, err)
		assert.NotNil(t, claims)
	})
}

func TestRefreshToken_Rotation(t *testing.T) {
	svc, _ := setupJWTService(t)
	ctx := context.Background()

	// Generate initial refresh token
	oldTokenStr, oldJTI, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	oldClaims, err := svc.ValidateRefreshToken(oldTokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-1", oldClaims.UserID)
	assert.Equal(t, "refresh", oldClaims.TokenType)

	// Rotate: blacklist old, issue new
	err = svc.BlacklistToken(ctx, oldJTI, 7*24*time.Hour)
	require.NoError(t, err)

	newTokenStr, newJTI, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	t.Run("old token is blacklisted", func(t *testing.T) {
		blacklisted, err := svc.IsBlacklisted(ctx, oldJTI)
		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("new token is not blacklisted", func(t *testing.T) {
		blacklisted, err := svc.IsBlacklisted(ctx, newJTI)
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("new token is valid", func(t *testing.T) {
		newClaims, err := svc.ValidateRefreshToken(newTokenStr)
		require.NoError(t, err)
		assert.Equal(t, "user-1", newClaims.UserID)
	})

	t.Run("JTIs differ", func(t *testing.T) {
		assert.NotEqual(t, oldJTI, newJTI)
	})
}

func TestRefreshToken_Suspended(t *testing.T) {
	svc, _ := setupJWTService(t)

	tokenStr, _, err := svc.GenerateRefreshToken("user-suspended")
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(tokenStr)
	require.NoError(t, err)

	t.Run("user ID matches", func(t *testing.T) {
		assert.Equal(t, "user-suspended", claims.UserID)
	})
	t.Run("token type is refresh", func(t *testing.T) {
		assert.Equal(t, "refresh", claims.TokenType)
	})
	t.Run("issuer is correct", func(t *testing.T) {
		assert.Equal(t, "medilink.health", claims.Issuer)
	})
}
