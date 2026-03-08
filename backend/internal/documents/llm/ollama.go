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

// OllamaExtractor implements LLMExtractor using Ollama.
type OllamaExtractor struct {
	baseURL    string
	model      string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewOllamaExtractor creates a new OllamaExtractor.
func NewOllamaExtractor(baseURL, model string, logger zerolog.Logger) *OllamaExtractor {
	if model == "" {
		model = "llama3.1"
	}
	return &OllamaExtractor{
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		logger:     logger,
	}
}

type ollamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	System  string         `json:"system"`
	Stream  bool           `json:"stream"`
	Format  string         `json:"format"`
	Options map[string]any `json:"options"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// ExtractLabResults extracts structured lab results from OCR text using Ollama.
func (o *OllamaExtractor) ExtractLabResults(ctx context.Context, ocrText, docType string) (*ExtractionResult, error) {
	prompt := BuildExtractionPrompt(ocrText)

	reqBody := ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		System: ExtractionSystemPrompt,
		Stream: false,
		Format: "json",
		Options: map[string]any{
			"temperature": 0.0,
			"num_predict": 2048,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama marshal: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama status: %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(ollamaResp.Response), &result); err != nil {
		return nil, fmt.Errorf("ollama JSON parse: %w", err)
	}

	return &result, nil
}

// ProviderName returns the provider name.
func (o *OllamaExtractor) ProviderName() string { return "ollama" }

// Health checks if Ollama is reachable.
func (o *OllamaExtractor) Health(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
