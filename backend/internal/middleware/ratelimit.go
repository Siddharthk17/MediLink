package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	Key      string
}

var RateLimits = map[string]RateLimitConfig{
	"patient":       {Requests: 100, Window: time.Minute, Key: "rl:patient"},
	"physician":     {Requests: 500, Window: time.Minute, Key: "rl:physician"},
	"admin":         {Requests: 1000, Window: time.Minute, Key: "rl:admin"},
	"search":        {Requests: 20, Window: time.Minute, Key: "rl:search"},
	"auth_login":    {Requests: 5, Window: time.Minute, Key: "rl:auth"},
	"auth_login_ip": {Requests: 20, Window: time.Minute, Key: "rl:auth_ip"},
	"break_glass":   {Requests: 3, Window: 24 * time.Hour, Key: "rl:breakglass"},
	"totp_verify":   {Requests: 5, Window: 10 * time.Minute, Key: "rl:totp"},
}

// RateLimitMiddleware enforces per-role rate limiting using Redis sorted sets.
func RateLimitMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("actor_role")
		if role == "" {
			role = "patient" // default for unauthenticated
		}

		config, ok := RateLimits[role]
		if !ok {
			config = RateLimits["patient"]
		}

		identifier := c.GetString("actor_id")
		if identifier == "" {
			identifier = c.ClientIP()
		}

		allowed, remaining, resetAt := checkRateLimit(c, redisClient, config, identifier)

		// Always set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if !allowed {
			retryAfter := resetAt - time.Now().Unix()
			if retryAfter < 0 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "throttled",
					"diagnostics": "Rate limit exceeded. Try again later.",
				}},
			})
			return
		}

		c.Next()
	}
}

// AuthRateLimitMiddleware is specifically for auth endpoints with stricter limits.
func AuthRateLimitMiddleware(redisClient *redis.Client, limitType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		config, ok := RateLimits[limitType]
		if !ok {
			c.Next()
			return
		}

		identifier := c.ClientIP()

		allowed, remaining, resetAt := checkRateLimit(c, redisClient, config, identifier)

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

		if !allowed {
			retryAfter := resetAt - time.Now().Unix()
			if retryAfter < 0 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"resourceType": "OperationOutcome",
				"issue": []gin.H{{
					"severity":    "error",
					"code":        "throttled",
					"diagnostics": "Too many attempts. Try again later.",
				}},
			})
			return
		}

		c.Next()
	}
}

func checkRateLimit(c *gin.Context, redisClient *redis.Client, config RateLimitConfig, identifier string) (allowed bool, remaining int, resetAt int64) {
	ctx := c.Request.Context()
	key := fmt.Sprintf("%s:%s", config.Key, identifier)
	now := time.Now()
	windowStart := now.Add(-config.Window)
	resetAt = now.Add(config.Window).Unix()

	// Use Redis pipeline for atomicity
	pipe := redisClient.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", float64(windowStart.UnixNano())))

	// Count current entries
	countCmd := pipe.ZCard(ctx, key)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		// On Redis error, allow the request (fail open)
		return true, config.Requests, resetAt
	}

	count := int(countCmd.Val())

	if count >= config.Requests {
		return false, 0, resetAt
	}

	// Add this request
	nowNano := float64(now.UnixNano())
	pipe2 := redisClient.Pipeline()
	pipe2.ZAdd(ctx, key, redis.Z{Score: nowNano, Member: nowNano})
	pipe2.Expire(ctx, key, config.Window)
	_, _ = pipe2.Exec(ctx)

	remaining = config.Requests - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, resetAt
}
