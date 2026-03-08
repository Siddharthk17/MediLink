package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/config"
)

// NewLLMExtractor returns the appropriate LLM implementation based on config.
// Gemini is PRIMARY — always tried first. Ollama is fallback.
func NewLLMExtractor(cfg *config.Config, logger zerolog.Logger) (LLMExtractor, error) {
	if cfg.Gemini.APIKey != "" {
		logger.Info().
			Str("provider", "gemini").
			Str("model", cfg.Gemini.Model).
			Msg("using Gemini as primary LLM extractor")
		return NewGeminiExtractor(cfg.Gemini.APIKey, logger), nil
	}

	if cfg.Ollama.BaseURL != "" {
		extractor := NewOllamaExtractor(cfg.Ollama.BaseURL, cfg.Ollama.Model, logger)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if extractor.Health(ctx) {
			logger.Info().
				Str("provider", "ollama").
				Str("model", cfg.Ollama.Model).
				Msg("using Ollama as fallback LLM extractor")
			return extractor, nil
		}
		logger.Warn().Str("url", cfg.Ollama.BaseURL).Msg("Ollama configured but unreachable")
	}

	return nil, fmt.Errorf(
		"no LLM provider available: set GEMINI_API_KEY or a reachable OLLAMA_BASE_URL")
}
