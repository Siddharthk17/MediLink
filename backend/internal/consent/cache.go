package consent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const consentCacheTTL = 5 * time.Minute

type ConsentCache struct {
	redis *redis.Client
}

func NewConsentCache(redisClient *redis.Client) *ConsentCache {
	return &ConsentCache{redis: redisClient}
}

// Get checks the cache for a consent decision.
// Returns (granted, scope, found).
func (cc *ConsentCache) Get(ctx context.Context, providerID, patientFHIRID string) (bool, []string, bool) {
	key := fmt.Sprintf("consent:%s:%s", providerID, patientFHIRID)
	val, err := cc.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil, false // cache miss
	}
	if err != nil {
		return false, nil, false // treat errors as cache miss
	}

	if val == "denied" {
		return false, nil, true
	}

	if strings.HasPrefix(val, "granted:") {
		scopeJSON := strings.TrimPrefix(val, "granted:")
		var scope []string
		if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
			return false, nil, false
		}
		return true, scope, true
	}

	return false, nil, false
}

// Set stores a consent decision in cache.
func (cc *ConsentCache) Set(ctx context.Context, providerID, patientFHIRID string, granted bool, scope []string) {
	key := fmt.Sprintf("consent:%s:%s", providerID, patientFHIRID)
	var val string
	if !granted {
		val = "denied"
	} else {
		scopeJSON, _ := json.Marshal(scope)
		val = "granted:" + string(scopeJSON)
	}
	cc.redis.Set(ctx, key, val, consentCacheTTL)
}

// Invalidate removes a consent entry from cache immediately.
func (cc *ConsentCache) Invalidate(ctx context.Context, providerID, patientFHIRID string) {
	key := fmt.Sprintf("consent:%s:%s", providerID, patientFHIRID)
	cc.redis.Del(ctx, key)
}
