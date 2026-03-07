package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Siddharthk17/MediLink/internal/auth"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-medilink-middleware-tests"

func setupTestRouter(jwtSvc auth.JWTService, handlers ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", append([]gin.HandlerFunc{auth.AuthMiddleware(jwtSvc)}, handlers...)...)
	return r
}

func newTestJWTService(t *testing.T) (auth.JWTService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	return auth.NewJWTService(testSecret, rdb), mr
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtSvc, _ := newTestJWTService(t)
	userID := uuid.New().String()

	tokenString, _, err := jwtSvc.GenerateAccessToken(userID, "patient", "org-1", false)
	require.NoError(t, err)

	var capturedActorID uuid.UUID
	var capturedRole string

	r := setupTestRouter(jwtSvc, func(c *gin.Context) {
		capturedActorID = auth.GetActorID(c)
		capturedRole = auth.GetActorRole(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, capturedActorID.String())
	assert.Equal(t, "patient", capturedRole)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	jwtSvc, _ := newTestJWTService(t)
	r := setupTestRouter(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	jwtSvc, _ := newTestJWTService(t)

	userID := uuid.New().String()
	pastTime := time.Now().Add(-3 * time.Hour)

	claims := auth.AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "medilink.health",
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"medilink.api"},
			ExpiresAt: jwt.NewNumericDate(pastTime.Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(pastTime),
			IssuedAt:  jwt.NewNumericDate(pastTime),
			ID:        "expired-jti-001",
		},
		UserID:       userID,
		Role:         "patient",
		OrgID:        "",
		TOTPVerified: false,
		TokenType:    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	r := setupTestRouter(jwtSvc)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}

func TestAuthMiddleware_PhysicianNoMFA(t *testing.T) {
	jwtSvc, _ := newTestJWTService(t)

	t.Run("AuthMiddleware blocks physician without MFA on non-allowed path", func(t *testing.T) {
		tokenString, _, err := jwtSvc.GenerateAccessToken(uuid.New().String(), "physician", "org-1", false)
		require.NoError(t, err)

		r := setupTestRouter(jwtSvc, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "MFA verification required")
	})

	t.Run("RequireMFA middleware blocks physician without TOTP verified", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/fhir/R4/Patient/:id", func(c *gin.Context) {
			c.Set(auth.ActorRoleKey, "physician")
			c.Set(auth.ActorTOTPVerifiedKey, false)
			c.Next()
		}, auth.RequireMFA(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "MFA verification required")
	})

	t.Run("RequireMFA allows patient without TOTP", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/fhir/R4/Patient/:id", func(c *gin.Context) {
			c.Set(auth.ActorRoleKey, "patient")
			c.Set(auth.ActorTOTPVerifiedKey, false)
			c.Next()
		}, auth.RequireMFA(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/fhir/R4/Patient/123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
