package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// GeminiExtractor implements LLMExtractor using Google Gemini.
type GeminiExtractor struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewGeminiExtractor creates a new GeminiExtractor.
func NewGeminiExtractor(apiKey string, logger zerolog.Logger) *GeminiExtractor {
	return &GeminiExtractor{
		apiKey:     apiKey,
		model:      "gemini-1.5-flash",
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		logger:     logger,
	}
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGenConfig  `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// ExtractLabResults extracts structured lab results using Gemini.
func (g *GeminiExtractor) ExtractLabResults(ctx context.Context, ocrText, docType string) (*ExtractionResult, error) {
	prompt := ExtractionSystemPrompt + "\n\n" + BuildExtractionPrompt(ocrText)

	reqBody := geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: prompt}},
		}},
		GenerationConfig: geminiGenConfig{
			Temperature:      0.0,
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini marshal: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini status: %d", resp.StatusCode)
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("gemini decode: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini empty response")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text

	var result ExtractionResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("gemini JSON parse: %w", err)
	}

	return &result, nil
}

// ProviderName returns the provider name.
func (g *GeminiExtractor) ProviderName() string { return "gemini" }
