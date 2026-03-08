package llm

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/config"
)

// NewLLMExtractor returns the appropriate LLM implementation based on config.
func NewLLMExtractor(cfg *config.Config, logger zerolog.Logger) (LLMExtractor, error) {
	if cfg.Ollama.BaseURL != "" {
		ollama := NewOllamaExtractor(cfg.Ollama.BaseURL, cfg.Ollama.Model, logger)
		if ollama.Health(context.Background()) {
			logger.Info().Msg("using Ollama for LLM extraction")
			return ollama, nil
		}
		logger.Warn().Msg("Ollama unreachable, falling back to Gemini")
	}

	if cfg.Gemini.APIKey != "" {
		logger.Info().Msg("using Gemini for LLM extraction")
		return NewGeminiExtractor(cfg.Gemini.APIKey, logger), nil
	}

	return nil, fmt.Errorf("no LLM provider available: set OLLAMA_BASE_URL or GEMINI_API_KEY")
}
