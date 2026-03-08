// Package main is the worker entrypoint for MediLink.
// Processes document OCR, email notifications, and Elasticsearch re-indexing tasks.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/anonymization"
	"github.com/Siddharthk17/MediLink/internal/config"
	"github.com/Siddharthk17/MediLink/internal/documents"
	"github.com/Siddharthk17/MediLink/internal/documents/llm"
	"github.com/Siddharthk17/MediLink/internal/documents/loinc"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	"github.com/Siddharthk17/MediLink/internal/tasks"
	"github.com/Siddharthk17/MediLink/pkg/database"
	"github.com/Siddharthk17/MediLink/pkg/search"
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

	// Initialize push service
	prefRepo := notifications.NewPostgresPrefsRepository(db.SQLX, log.Logger)
	var pushSvc notifications.PushService
	fcm, fcmErr := notifications.NewFCMPushService(cfg, prefRepo, log.Logger)
	if fcmErr != nil {
		log.Warn().Err(fcmErr).Msg("Firebase unavailable — using noop push service")
		pushSvc = &notifications.NoopPushService{}
	} else {
		pushSvc = fcm
	}

	// Notification task worker
	notifWorker := notifications.NewNotificationWorker(pushSvc, emailSvc, log.Logger)

	// Initialize Elasticsearch client (for reindex task)
	esAddresses := strings.Split(os.Getenv("ELASTICSEARCH_URL"), ",")
	if len(esAddresses) == 0 || esAddresses[0] == "" {
		esAddresses = []string{"http://localhost:9200"}
	}
	var esClient search.SearchClient
	es, esErr := search.NewESClient(esAddresses, log.Logger)
	if esErr != nil {
		log.Warn().Err(esErr).Msg("Elasticsearch unavailable — reindex tasks will fail")
	} else {
		esClient = es
	}

	// Initialize Redis client (for consent cache invalidation)
	redisOpt, _ := redis.ParseURL(cfg.Redis.URL)
	redisClient := redis.NewClient(redisOpt)

	// Anonymization export processor
	exportRepo := anonymization.NewPostgresExportRepository(db.SQLX)
	exporter := anonymization.NewExporter(storageClient, log.Logger)
	exportProcessor := anonymization.NewExportProcessor(db.SQLX, exportRepo, exporter, log.Logger)

	// Initialize document processor
	docJobRepo := documents.NewDocumentJobRepository(db.SQLX, log.Logger)
	ocrEngine := documents.NewTesseractOCR()
	loincMapper := loinc.NewLOINCMapper(db.SQLX, log.Logger)
	docProcessor := documents.NewDocumentProcessor(
		docJobRepo, storageClient, ocrEngine, llmExtractor, loincMapper,
		db.SQLX, emailSvc, log.Logger,
	)

	// Periodic task handlers
	tokenCleanup := tasks.NewTokenCleanupTask(db.SQLX, log.Logger)
	esReindex := tasks.NewESReindexTask(db.SQLX, esClient, log.Logger)
	jobExpiry := tasks.NewJobExpiryTask(db.SQLX, log.Logger)
	consentExpiry := tasks.NewConsentExpiryTask(db.SQLX, redisClient, log.Logger)
	statsSnapshot := tasks.NewStatsSnapshotTask(db.SQLX, log.Logger)

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
	// Document processing
	mux.HandleFunc(documents.TaskProcessDocument, docProcessor.ProcessDocument)
	// Notification tasks
	mux.HandleFunc(notifications.TaskSendPush, notifWorker.ProcessPushTask)
	mux.HandleFunc(notifications.TaskSendEmail, notifWorker.ProcessEmailTask)
	// Anonymization export
	mux.HandleFunc(anonymization.TaskAnonymizedExport, exportProcessor.Process)
	// Periodic tasks
	mux.HandleFunc(tasks.TaskCleanupExpiredTokens, tokenCleanup.Process)
	mux.HandleFunc(tasks.TaskESReindexMissed, esReindex.Process)
	mux.HandleFunc(tasks.TaskExpireStaleJobs, jobExpiry.Process)
	mux.HandleFunc(tasks.TaskRevokeExpiredConsents, consentExpiry.Process)
	mux.HandleFunc(tasks.TaskDailyStatsSnapshot, statsSnapshot.Process)

	// Setup scheduler for periodic tasks
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr},
		&asynq.SchedulerOpts{
			Logger: &zeroAsynqLogger{log.Logger},
		},
	)
	tasks.RegisterPeriodicTasks(scheduler)

	var wg sync.WaitGroup

	// Start Asynq task server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("Asynq worker started — processing tasks")
		if err := srv.Run(mux); err != nil {
			log.Error().Err(err).Msg("asynq server failed")
		}
	}()

	// Start scheduler
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("Asynq scheduler started — periodic tasks registered")
		if err := scheduler.Run(); err != nil {
			log.Error().Err(err).Msg("asynq scheduler failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down worker...")
	srv.Shutdown()
	scheduler.Shutdown()
	wg.Wait()
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

