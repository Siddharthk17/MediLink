package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Siddharthk17/MediLink/internal/middleware"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		id := middleware.GetRequestID(c)
		assert.NotEmpty(t, id)
		c.String(200, id)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestIDMiddleware_UsesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, middleware.GetRequestID(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	router.ServeHTTP(w, req)

	assert.Equal(t, "my-custom-id", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "my-custom-id", w.Body.String())
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORSMiddleware())
	router.OPTIONS("/test", func(c *gin.Context) {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.SecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-store, no-cache, must-revalidate, private", w.Header().Get("Cache-Control"))
}
