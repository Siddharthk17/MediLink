package llm

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/config"
)

// NewLLMExtractor returns a Gemini LLM extractor based on config.
func NewLLMExtractor(cfg *config.Config, logger zerolog.Logger) (LLMExtractor, error) {
	if cfg.Gemini.APIKey != "" {
		logger.Info().
			Str("provider", "gemini").
			Str("model", cfg.Gemini.Model).
			Msg("using Gemini as LLM extractor")
		return NewGeminiExtractor(cfg.Gemini.APIKey, logger), nil
	}

	return nil, fmt.Errorf("no LLM provider available: set GEMINI_API_KEY")
}
