package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const (
	jwtIssuer          = "medilink.health"
	jwtAudience        = "medilink.api"
	accessTokenExpiry  = 2 * time.Hour
	refreshTokenExpiry = 7 * 24 * time.Hour
)

// AccessTokenClaims represents the claims in a MediLink access token.
type AccessTokenClaims struct {
	jwt.RegisteredClaims
	UserID       string `json:"uid"`
	Role         string `json:"role"`
	OrgID        string `json:"oid"`
	TOTPVerified bool   `json:"mfa"`
	TokenType    string `json:"type"`
}

// RefreshTokenClaims represents the claims in a MediLink refresh token.
type RefreshTokenClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"uid"`
	TokenType string `json:"type"`
}

// JWTService handles JWT token operations.
type JWTService interface {
	GenerateAccessToken(userID, role, orgID string, totpVerified bool) (string, string, error)
	GenerateRefreshToken(userID string) (string, string, error)
	ValidateAccessToken(tokenString string) (*AccessTokenClaims, error)
	ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error)
	BlacklistToken(ctx context.Context, jti string, remaining time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

type jwtService struct {
	secret      []byte
	redisClient *redis.Client
}

// NewJWTService creates a new JWT service.
func NewJWTService(secret string, redisClient *redis.Client) JWTService {
	return &jwtService{
		secret:      []byte(secret),
		redisClient: redisClient,
	}
}

func (s *jwtService) generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate jti: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}

func (s *jwtService) GenerateAccessToken(userID, role, orgID string, totpVerified bool) (string, string, error) {
	jti, err := s.generateJTI()
	if err != nil {
		return "", "", err
	}

	now := time.Now()
	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{jwtAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
		UserID:       userID,
		Role:         role,
		OrgID:        orgID,
		TOTPVerified: totpVerified,
		TokenType:    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, jti, nil
}

func (s *jwtService) GenerateRefreshToken(userID string) (string, string, error) {
	jti, err := s.generateJTI()
	if err != nil {
		return "", "", err
	}

	now := time.Now()
	claims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{jwtAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
		UserID:    userID,
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, jti, nil
}

func (s *jwtService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// CRITICAL: Enforce HS256 only — reject algorithm confusion attacks
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return s.secret, nil
	},
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("token is not an access token")
	}

	return claims, nil
}

func (s *jwtService) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	claims := &RefreshTokenClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return s.secret, nil
	},
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("refresh token is not valid")
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	return claims, nil
}

func (s *jwtService) BlacklistToken(ctx context.Context, jti string, remaining time.Duration) error {
	key := fmt.Sprintf("jwt:blacklist:%s", jti)
	return s.redisClient.Set(ctx, key, "1", remaining).Err()
}

func (s *jwtService) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("jwt:blacklist:%s", jti)
	val, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}
