package clinical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RxNormClient queries the NLM RxNorm API.
type RxNormClient struct {
	baseURL    string
	httpClient *http.Client
	cache      *redis.Client
	logger     zerolog.Logger
}

// NewRxNormClient creates a new RxNorm API client.
func NewRxNormClient(cache *redis.Client, logger zerolog.Logger) *RxNormClient {
	return &RxNormClient{
		baseURL:    "https://rxnav.nlm.nih.gov/REST",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      cache,
		logger:     logger,
	}
}

// GetDrugName returns the official drug name for an RxNorm code. Cached in Redis for 24 hours.
func (c *RxNormClient) GetDrugName(ctx context.Context, rxNormCode string) (string, error) {
	cacheKey := "rxnorm:name:" + rxNormCode

	// Check Redis cache
	cached, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		return cached, nil
	}

	url := fmt.Sprintf("%s/rxcui/%s/properties.json", c.baseURL, rxNormCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return rxNormCode, fmt.Errorf("rxnorm request build: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn().Err(err).Str("rxnorm", rxNormCode).Msg("RxNorm API unreachable")
		return rxNormCode, nil // return code as name on failure
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rxNormCode, nil
	}

	var result struct {
		Properties struct {
			Name string `json:"name"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return rxNormCode, nil
	}

	name := result.Properties.Name
	if name == "" {
		name = rxNormCode
	}

	// Cache for 24 hours
	c.cache.Set(ctx, cacheKey, name, 24*time.Hour)
	return name, nil
}

// GetDrugClass returns the drug class(es) for an RxNorm code.
func (c *RxNormClient) GetDrugClass(ctx context.Context, rxNormCode string) ([]string, error) {
	cacheKey := "rxnorm:class:" + rxNormCode

	cached, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var classes []string
		if err := json.Unmarshal([]byte(cached), &classes); err == nil {
			return classes, nil
		}
	}

	url := fmt.Sprintf("%s/rxclass/class/byRxcui.json?rxcui=%s", c.baseURL, rxNormCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("rxnorm class request build: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn().Err(err).Str("rxnorm", rxNormCode).Msg("RxNorm class API unreachable")
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var result struct {
		RxclassDrugInfoList struct {
			RxclassDrugInfo []struct {
				RxclassMinConceptItem struct {
					ClassName string `json:"className"`
				} `json:"rxclassMinConceptItem"`
			} `json:"rxclassDrugInfo"`
		} `json:"rxclassDrugInfoList"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil
	}

	classes := make([]string, 0)
	seen := make(map[string]bool)
	for _, info := range result.RxclassDrugInfoList.RxclassDrugInfo {
		className := info.RxclassMinConceptItem.ClassName
		if className != "" && !seen[className] {
			classes = append(classes, className)
			seen[className] = true
		}
	}

	// Cache for 24 hours
	if data, err := json.Marshal(classes); err == nil {
		c.cache.Set(ctx, cacheKey, string(data), 24*time.Hour)
	}

	return classes, nil
}

// NormalizeDrugName converts a free-text drug name to an RxNorm code.
func (c *RxNormClient) NormalizeDrugName(ctx context.Context, drugName string) (string, error) {
	cacheKey := "rxnorm:normalize:" + drugName

	cached, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		return cached, nil
	}

	url := fmt.Sprintf("%s/rxcui.json?name=%s&search=2", c.baseURL, drugName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("rxnorm normalize request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("rxnorm normalize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rxnorm normalize status: %d", resp.StatusCode)
	}

	var result struct {
		IDGroup struct {
			RxnormID []string `json:"rxnormId"`
		} `json:"idGroup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("rxnorm normalize decode: %w", err)
	}

	if len(result.IDGroup.RxnormID) == 0 {
		return "", fmt.Errorf("no RxNorm code found for: %s", drugName)
	}

	code := result.IDGroup.RxnormID[0]
	c.cache.Set(ctx, cacheKey, code, 24*time.Hour)
	return code, nil
}
