package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Siddharthk17/MediLink/internal/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRateLimitRouter(t *testing.T, redisClient *redis.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("actor_role", "patient") // patient has 100/min limit
		c.Next()
	})
	r.Use(middleware.RateLimitMiddleware(redisClient))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	return rdb, mr
}

func doRequest(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit_UnderLimit(t *testing.T) {
	rdb, _ := newTestRedis(t)
	r := setupRateLimitRouter(t, rdb)

	// Single request should pass
	w := doRequest(r)
	assert.Equal(t, http.StatusOK, w.Code)

	// A few more requests should all pass
	for i := 0; i < 9; i++ {
		w = doRequest(r)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimit_AtLimit(t *testing.T) {
	rdb, _ := newTestRedis(t)
	r := setupRateLimitRouter(t, rdb)

	patientLimit := middleware.RateLimits["patient"].Requests // 100

	// Send exactly the limit number of requests — all should pass
	for i := 0; i < patientLimit; i++ {
		w := doRequest(r)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
	}
}

func TestRateLimit_OverLimit(t *testing.T) {
	rdb, _ := newTestRedis(t)
	r := setupRateLimitRouter(t, rdb)

	patientLimit := middleware.RateLimits["patient"].Requests // 100

	// Exhaust the limit
	for i := 0; i < patientLimit; i++ {
		w := doRequest(r)
		require.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
	}

	// Next request should be rate limited
	w := doRequest(r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
}

func TestRateLimit_Headers(t *testing.T) {
	rdb, _ := newTestRedis(t)
	r := setupRateLimitRouter(t, rdb)

	patientLimit := middleware.RateLimits["patient"].Requests // 100

	t.Run("headers present on first request", func(t *testing.T) {
		w := doRequest(r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"), "X-RateLimit-Limit header should be present")
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"), "X-RateLimit-Remaining header should be present")
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"), "X-RateLimit-Reset header should be present")

		assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "99", w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("headers present on rate-limited response", func(t *testing.T) {
		// Exhaust remaining requests (already used 1 above)
		for i := 1; i < patientLimit; i++ {
			doRequest(r)
		}

		w := doRequest(r)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"), "X-RateLimit-Limit header should be present on 429")
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"), "X-RateLimit-Remaining header should be present on 429")
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"), "X-RateLimit-Reset header should be present on 429")
		assert.NotEmpty(t, w.Header().Get("Retry-After"), "Retry-After header should be present on 429")

		assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	})
}
