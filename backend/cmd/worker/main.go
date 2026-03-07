// Package main is the worker entrypoint for MediLink.
// This is a scaffold for Week 1 — real worker tasks will be added in later weeks.
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	log.Info().
		Str("env", cfg.App.Environment).
		Msg("MediLink worker starting (scaffold — no tasks registered yet)")

	// Week 1: Worker scaffold only.
	// Asynq task processing will be added in Week 2+.
	select {}
}
