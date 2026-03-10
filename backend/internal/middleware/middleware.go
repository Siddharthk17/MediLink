// Package middleware provides HTTP middleware for the MediLink API.
package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RequestIDKey is the context key for the request ID.
const RequestIDKey = "request_id"

// RequestIDMiddleware generates a unique request ID for each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RequestLoggingMiddleware logs each HTTP request using zerolog.
func RequestLoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get(RequestIDKey)

		event := logger.Info()
		if status >= 400 {
			event = logger.Warn()
		}
		if status >= 500 {
			event = logger.Error()
		}

		rid, _ := requestID.(string)

		event.
			Str("request_id", rid).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP request")
	}
}

// allowedOrigins is the set of origins permitted for CORS.
// Populated once via InitCORSOrigins from the CORS_ALLOWED_ORIGINS env var.
var allowedOrigins = map[string]bool{
	"http://localhost:3000": true,
	"http://localhost:3001": true,
}

// InitCORSOrigins replaces the default origin allowlist.
// Pass a comma-separated string of origins (e.g. "https://app.medilink.health,http://localhost:3000").
func InitCORSOrigins(origins string) {
	if origins == "" {
		return
	}
	m := make(map[string]bool)
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			m[o] = true
		}
	}
	allowedOrigins = m
}

// CORSMiddleware handles Cross-Origin Resource Sharing.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Location, ETag")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Vary", "Origin")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers to all responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Content-Type", "application/fhir+json; charset=utf-8")
		c.Next()
	}
}

// GetRequestID extracts the request ID from the Gin context.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		return id.(string)
	}
	return ""
}

// GetClientIP extracts the client IP from the Gin context.
func GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetUserAgent extracts the user agent from the request.
func GetUserAgent(c *gin.Context) string {
	return c.Request.UserAgent()
}

