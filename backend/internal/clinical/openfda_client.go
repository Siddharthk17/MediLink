package clinical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// OpenFDAClient queries the OpenFDA drug label API.
type OpenFDAClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// DrugLabelResult represents the OpenFDA API response.
type DrugLabelResult struct {
	Results []struct {
		OpenFDAInfo struct {
			RxCUI       []string `json:"rxcui"`
			GenericName []string `json:"generic_name"`
			BrandName   []string `json:"brand_name"`
		} `json:"openfda"`
		DrugInteractions  []string `json:"drug_interactions"`
		Warnings          []string `json:"warnings"`
		Contraindications []string `json:"contraindications"`
	} `json:"results"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewOpenFDAClient creates a new OpenFDA API client.
func NewOpenFDAClient(baseURL string, logger zerolog.Logger) *OpenFDAClient {
	if baseURL == "" {
		baseURL = "https://api.fda.gov"
	}
	return &OpenFDAClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// extractGenericName strips dosage, strength, and form from an RxNorm drug name.
// "amoxicillin 250 MG Oral Capsule" → "amoxicillin"
// "Metformin Hydrochloride 500 MG" → "Metformin Hydrochloride"
func extractGenericName(fullName string) string {
	parts := strings.Fields(fullName)
	var cleaned []string
	for _, p := range parts {
		// Stop at the first numeric token (dosage strength like "250", "500")
		if len(p) > 0 && p[0] >= '0' && p[0] <= '9' {
			break
		}
		// Skip common dosage-form words
		lower := strings.ToLower(p)
		if lower == "mg" || lower == "ml" || lower == "mcg" || lower == "oral" ||
			lower == "tablet" || lower == "capsule" || lower == "solution" ||
			lower == "injection" || lower == "cream" || lower == "ointment" ||
			lower == "drops" || lower == "suspension" || lower == "extended" ||
			lower == "release" || lower == "delayed" || lower == "chewable" {
			continue
		}
		cleaned = append(cleaned, p)
	}
	if len(cleaned) == 0 {
		return fullName
	}
	return strings.Join(cleaned, " ")
}

// GetDrugInteractions fetches interaction data for a drug from OpenFDA.
func (c *OpenFDAClient) GetDrugInteractions(ctx context.Context, drugName string) (*DrugLabelResult, error) {
	cleanName := extractGenericName(drugName)
	searchURL := fmt.Sprintf("%s/drug/label.json?search=drug_interactions:%s&limit=5",
		c.baseURL, url.QueryEscape(cleanName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openfda request build: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn().Err(err).Str("drug", drugName).Msg("OpenFDA API unreachable")
		return nil, fmt.Errorf("openfda request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		c.logger.Warn().Str("drug", drugName).Msg("OpenFDA rate limited")
		return nil, fmt.Errorf("openfda rate limited")
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn().Int("status", resp.StatusCode).Str("drug", cleanName).Msg("OpenFDA non-200 response")
		return &DrugLabelResult{}, nil
	}

	var result DrugLabelResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openfda decode: %w", err)
	}

	if result.Error != nil {
		c.logger.Warn().Str("code", result.Error.Code).Str("msg", result.Error.Message).Msg("OpenFDA returned error")
		return &DrugLabelResult{}, nil
	}

	return &result, nil
}

// FindInteractionInLabels searches OpenFDA label results for mentions of a second drug.
func FindInteractionInLabels(labels *DrugLabelResult, otherDrugName string) (string, InteractionSeverity) {
	if labels == nil || len(labels.Results) == 0 {
		return "", SeverityNone
	}

	otherLower := strings.ToLower(otherDrugName)

	// Check contraindications first (most severe)
	for _, result := range labels.Results {
		for _, ci := range result.Contraindications {
			if strings.Contains(strings.ToLower(ci), otherLower) {
				return ci, SeverityContraindicated
			}
		}
	}

	// Check drug_interactions field
	for _, result := range labels.Results {
		for _, di := range result.DrugInteractions {
			if strings.Contains(strings.ToLower(di), otherLower) {
				severity := ParseInteractionSeverity(di)
				return di, severity
			}
		}
	}

	// Check warnings
	for _, result := range labels.Results {
		for _, w := range result.Warnings {
			if strings.Contains(strings.ToLower(w), otherLower) {
				severity := ParseInteractionSeverity(w)
				return w, severity
			}
		}
	}

	return "", SeverityNone
}

// ParseInteractionSeverity classifies free-text interaction description into severity levels.
func ParseInteractionSeverity(description string) InteractionSeverity {
	lower := strings.ToLower(description)

	contraindicatedTerms := []string{
		"contraindicated", "must not", "do not use", "avoid concomitant",
		"should not be used together", "never administer",
	}
	for _, term := range contraindicatedTerms {
		if strings.Contains(lower, term) {
			return SeverityContraindicated
		}
	}

	majorTerms := []string{
		"serious", "life-threatening", "potentially fatal", "significant risk",
		"severe", "dangerous", "death", "fatal",
	}
	for _, term := range majorTerms {
		if strings.Contains(lower, term) {
			return SeverityMajor
		}
	}

	moderateTerms := []string{
		"may increase", "monitor", "caution", "use with caution",
		"increased risk", "may decrease", "adjustment", "dose adjustment",
	}
	for _, term := range moderateTerms {
		if strings.Contains(lower, term) {
			return SeverityModerate
		}
	}

	minorTerms := []string{
		"mild", "unlikely", "minimally", "minor",
	}
	for _, term := range minorTerms {
		if strings.Contains(lower, term) {
			return SeverityMinor
		}
	}

	// If we found text mentioning the drug but can't classify, default to moderate
	if len(description) > 0 {
		return SeverityModerate
	}

	return SeverityNone
}
