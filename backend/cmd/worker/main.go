// Package main is the worker entrypoint for MediLink.
// Processes document OCR, email notifications, and Elasticsearch re-indexing tasks.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/config"
	"github.com/Siddharthk17/MediLink/internal/documents"
	"github.com/Siddharthk17/MediLink/internal/documents/llm"
	"github.com/Siddharthk17/MediLink/internal/documents/loinc"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	"github.com/Siddharthk17/MediLink/pkg/database"
	"github.com/Siddharthk17/MediLink/pkg/storage"
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
		Msg("MediLink worker starting")

	// Connect to PostgreSQL
	db, err := database.NewPostgresConnections(
		cfg.Database.DSN,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to PostgreSQL")
	}
	defer db.Close()

	// Initialize MinIO storage
	var storageClient storage.StorageClient
	mc, err := storage.NewMinIOClient(
		cfg.Storage.Endpoint, cfg.Storage.AccessKey, cfg.Storage.SecretKey,
		cfg.Storage.Bucket, cfg.Storage.UseSSL, log.Logger,
	)
	if err != nil {
		log.Warn().Err(err).Msg("MinIO unavailable — using noop storage")
		storageClient = &storage.NoopStorageClient{}
	} else {
		storageClient = mc
	}

	// Initialize LLM extractor
	var llmExtractor llm.LLMExtractor
	llmExtractor, err = llm.NewLLMExtractor(cfg, log.Logger)
	if err != nil {
		log.Warn().Err(err).Msg("no LLM provider available — document processing will fail")
	}

	// Initialize email service
	var emailSvc notifications.EmailService
	if cfg.Resend.APIKey != "" {
		emailSvc = notifications.NewResendEmailService(cfg.Resend.APIKey, cfg.Resend.FromEmail, log.Logger)
	} else {
		emailSvc = &notifications.NoopEmailService{}
	}

	// Initialize document processor
	docJobRepo := documents.NewDocumentJobRepository(db.SQLX, log.Logger)
	ocrEngine := documents.NewTesseractOCR()
	loincMapper := loinc.NewLOINCMapper(db.SQLX, log.Logger)
	docProcessor := documents.NewDocumentProcessor(
		docJobRepo, storageClient, ocrEngine, llmExtractor, loincMapper,
		db.SQLX, emailSvc, log.Logger,
	)

	// Parse Redis URL for Asynq
	redisAddr := cfg.Redis.URL
	if len(redisAddr) > 8 && redisAddr[:8] == "redis://" {
		redisAddr = redisAddr[8:]
	}

	// Start Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"documents":     10,
				"notifications":  5,
				"elasticsearch":  3,
				"default":        1,
			},
			Logger: &zeroAsynqLogger{log.Logger},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(documents.TaskProcessDocument, docProcessor.ProcessDocument)

	// Start server in a goroutine
	go func() {
		log.Info().Msg("Asynq worker started — processing tasks")
		if err := srv.Run(mux); err != nil {
			log.Fatal().Err(err).Msg("asynq server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down worker...")
	srv.Shutdown()
	log.Info().Msg("worker exited")
}

// zeroAsynqLogger adapts zerolog for Asynq's logger interface.
type zeroAsynqLogger struct {
	logger zerolog.Logger
}

func (l *zeroAsynqLogger) Debug(args ...interface{}) { l.logger.Debug().Msgf("%v", args) }
func (l *zeroAsynqLogger) Info(args ...interface{})  { l.logger.Info().Msgf("%v", args) }
func (l *zeroAsynqLogger) Warn(args ...interface{})  { l.logger.Warn().Msgf("%v", args) }
func (l *zeroAsynqLogger) Error(args ...interface{}) { l.logger.Error().Msgf("%v", args) }
func (l *zeroAsynqLogger) Fatal(args ...interface{}) { l.logger.Fatal().Msgf("%v", args) }

