// Package main is the API server entrypoint for MediLink.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/medilink/internal/audit"
	"github.com/Siddharthk17/medilink/internal/auth"
	"github.com/Siddharthk17/medilink/internal/config"
	"github.com/Siddharthk17/medilink/internal/fhir/handlers"
	"github.com/Siddharthk17/medilink/internal/fhir/repository"
	"github.com/Siddharthk17/medilink/internal/fhir/services"
	"github.com/Siddharthk17/medilink/internal/fhir/validator"
	"github.com/Siddharthk17/medilink/internal/middleware"
	"github.com/Siddharthk17/medilink/pkg/cache"
	"github.com/Siddharthk17/medilink/pkg/crypto"
	"github.com/Siddharthk17/medilink/pkg/database"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
	"github.com/Siddharthk17/medilink/pkg/search"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup zerolog
	setupLogger(cfg.App.LogLevel)

	// Validate encryption key at startup
	_, err = crypto.NewAESEncryptor(cfg.Encryption.Key)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid encryption key — refusing to start")
	}
	log.Info().Msg("encryption key validated")

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

	// Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}
	defer redisClient.Close()

	// Initialize Elasticsearch
	esAddresses := strings.Split(os.Getenv("ELASTICSEARCH_URL"), ",")
	if len(esAddresses) == 0 || esAddresses[0] == "" {
		esAddresses = []string{"http://localhost:9200"}
	}

	var esClient search.SearchClient
	realES, esErr := search.NewESClient(esAddresses, log.Logger)
	if esErr != nil {
		log.Warn().Err(esErr).Msg("elasticsearch unavailable — using noop client")
		esClient = &search.NoopSearchClient{}
	} else {
		esClient = realES
		if err := esClient.EnsureIndices(context.Background()); err != nil {
			log.Warn().Err(err).Msg("failed to ensure elasticsearch indices")
		}
		log.Info().Msg("elasticsearch connected and indices ensured")
	}

	// Initialize dependencies
	auditLogger := audit.NewPostgresAuditLogger(db.SQLX)
	defer auditLogger.Close()

	fhirValidator := validator.NewFHIRValidator()
	refValidator := services.NewReferenceValidator(db.SQLX)

	// Patient (existing Week 1 resource)
	patientRepo := repository.NewPostgresPatientRepository(db.SQLX)
	patientService := services.NewPatientService(patientRepo, fhirValidator, auditLogger)
	patientHandler := handlers.NewPatientHandler(patientService)

	// Practitioner
	practitionerRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Practitioner")
	practitionerService := services.NewResourceService(
		&practitionerRepo, fhirValidator.ValidatePractitioner, refValidator, auditLogger, "Practitioner",
	)
	practitionerHandler := handlers.NewResourceHandler(practitionerService, "Practitioner", handlers.PractitionerSearchParser)

	// Organization
	organizationRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Organization")
	organizationService := services.NewResourceService(
		&organizationRepo, fhirValidator.ValidateOrganization, refValidator, auditLogger, "Organization",
	)
	organizationHandler := handlers.NewResourceHandler(organizationService, "Organization", handlers.OrganizationSearchParser)

	// Encounter — status transitions + patient ref validation
	encounterRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Encounter")
	encounterService := services.NewResourceService(
		&encounterRepo, fhirValidator.ValidateEncounter, refValidator, auditLogger, "Encounter",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			ref := extractRef(data, "subject")
			return refValidator.ValidateReference(ctx, ref, "Patient")
		}),
		services.WithPreUpdateHook(encounterStatusTransitionHook(&encounterRepo, refValidator)),
	)
	encounterHandler := handlers.NewResourceHandler(encounterService, "Encounter", handlers.EncounterSearchParser)

	// Condition — patient + encounter ref validation
	conditionRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Condition")
	conditionService := services.NewResourceService(
		&conditionRepo, fhirValidator.ValidateCondition, refValidator, auditLogger, "Condition",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
		services.WithPreUpdateHook(func(ctx context.Context, _ string, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
	)
	conditionHandler := handlers.NewResourceHandler(conditionService, "Condition", handlers.ConditionSearchParser)

	// MedicationRequest — status transitions + patient + encounter ref validation
	medReqRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "MedicationRequest")
	medReqService := services.NewResourceService(
		&medReqRepo, fhirValidator.ValidateMedicationRequest, refValidator, auditLogger, "MedicationRequest",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
		services.WithPreUpdateHook(medReqStatusTransitionHook(&medReqRepo, refValidator)),
	)
	medReqHandler := handlers.NewResourceHandler(medReqService, "MedicationRequest", handlers.MedicationRequestSearchParser)

	// Observation — patient + encounter ref validation
	observationRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Observation")
	observationService := services.NewResourceService(
		&observationRepo, fhirValidator.ValidateObservation, refValidator, auditLogger, "Observation",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
		services.WithPreUpdateHook(func(ctx context.Context, _ string, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
	)
	observationHandler := handlers.NewResourceHandler(observationService, "Observation", handlers.ObservationSearchParser)

	// DiagnosticReport — patient + encounter + result refs
	diagReportRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "DiagnosticReport")
	diagReportService := services.NewResourceService(
		&diagReportRepo, fhirValidator.ValidateDiagnosticReport, refValidator, auditLogger, "DiagnosticReport",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
		services.WithPreUpdateHook(func(ctx context.Context, _ string, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			return refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter")
		}),
	)
	diagReportHandler := handlers.NewResourceHandler(diagReportService, "DiagnosticReport", handlers.DiagnosticReportSearchParser)

	// AllergyIntolerance — patient ref (uses "patient" field, not "subject")
	allergyRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "AllergyIntolerance")
	allergyService := services.NewResourceService(
		&allergyRepo, fhirValidator.ValidateAllergyIntolerance, refValidator, auditLogger, "AllergyIntolerance",
		services.WithPatientRefExtractor(services.ExtractPatientFieldRef),
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			return refValidator.ValidateReference(ctx, extractRef(data, "patient"), "Patient")
		}),
		services.WithPreUpdateHook(func(ctx context.Context, _ string, data json.RawMessage) error {
			return refValidator.ValidateReference(ctx, extractRef(data, "patient"), "Patient")
		}),
	)
	allergyHandler := handlers.NewResourceHandler(allergyService, "AllergyIntolerance", handlers.AllergyIntoleranceSearchParser)

	// Immunization — patient ref (uses "patient" field, not "subject")
	immunizationRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "Immunization")
	immunizationService := services.NewResourceService(
		&immunizationRepo, fhirValidator.ValidateImmunization, refValidator, auditLogger, "Immunization",
		services.WithPatientRefExtractor(services.ExtractPatientFieldRef),
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			return refValidator.ValidateReference(ctx, extractRef(data, "patient"), "Patient")
		}),
		services.WithPreUpdateHook(func(ctx context.Context, _ string, data json.RawMessage) error {
			return refValidator.ValidateReference(ctx, extractRef(data, "patient"), "Patient")
		}),
	)
	immunizationHandler := handlers.NewResourceHandler(immunizationService, "Immunization", handlers.ImmunizationSearchParser)

	// Timeline and Lab Trends services
	timelineService := services.NewTimelineService(db.SQLX, auditLogger)
	timelineHandler := handlers.NewTimelineHandler(timelineService)

	labTrendsService := services.NewLabTrendsService(db.SQLX, auditLogger)
	labTrendsHandler := handlers.NewLabTrendsHandler(labTrendsService)

	// Setup Gin
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// Middleware stack — order matters
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.RequestLoggingMiddleware(log.Logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())

	// FHIR R4 routes
	v1 := router.Group("/fhir/R4")
	v1.Use(auth.AuthMiddleware())
	{
		// Patient (Week 1)
		patients := v1.Group("/Patient")
		{
			patients.POST("", patientHandler.CreatePatient)
			patients.GET("", patientHandler.SearchPatients)
			patients.GET("/:id", patientHandler.GetPatient)
			patients.PUT("/:id", patientHandler.UpdatePatient)
			patients.DELETE("/:id", patientHandler.DeletePatient)
			patients.GET("/:id/_history", patientHandler.GetPatientHistory)
			patients.GET("/:id/_history/:vid", patientHandler.GetPatientVersion)
		}

		// Timeline (special endpoint under Patient)
		v1.GET("/Patient/:id/$timeline", timelineHandler.GetTimeline)

		// Week 2 resources — all follow the same pattern
		resourceHandlers := map[string]*handlers.ResourceHandler{
			"Practitioner":       practitionerHandler,
			"Organization":       organizationHandler,
			"Encounter":          encounterHandler,
			"Condition":          conditionHandler,
			"MedicationRequest":  medReqHandler,
			"Observation":        observationHandler,
			"DiagnosticReport":   diagReportHandler,
			"AllergyIntolerance": allergyHandler,
			"Immunization":       immunizationHandler,
		}

		for resourceType, handler := range resourceHandlers {
			group := v1.Group("/" + resourceType)
			{
				group.POST("", handler.Create)
				group.GET("", handler.Search)
				group.GET("/:id", handler.Get)
				group.PUT("/:id", handler.Update)
				group.DELETE("/:id", handler.Delete)
				group.GET("/:id/_history", handler.GetHistory)
				group.GET("/:id/_history/:vid", handler.GetVersion)
			}
		}

		// Lab Trends (special endpoint under Observation)
		v1.GET("/Observation/$lab-trends", labTrendsHandler.GetLabTrends)
	}

	// Health check endpoints — no auth, no rate limit
	router.GET("/health", healthCheckHandler(db, redisClient))
	router.GET("/ready", readinessHandler(db, redisClient, esClient))

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("MediLink API server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}

// extractRef extracts a reference string from a FHIR resource field.
func extractRef(data json.RawMessage, fieldName string) string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if field, ok := m[fieldName].(map[string]interface{}); ok {
		if ref, ok := field["reference"].(string); ok {
			return ref
		}
	}
	return ""
}

// encounterStatusTransitionHook validates Encounter status transitions.
func encounterStatusTransitionHook(repo *repository.BaseRepository, rv *services.ReferenceValidator) func(ctx context.Context, resourceID string, data json.RawMessage) error {
	validTransitions := map[string]map[string]bool{
		"planned":     {"arrived": true, "cancelled": true},
		"arrived":     {"triaged": true, "in-progress": true, "cancelled": true},
		"triaged":     {"in-progress": true, "cancelled": true},
		"in-progress": {"onleave": true, "finished": true, "cancelled": true},
		"onleave":     {"in-progress": true, "finished": true, "cancelled": true},
	}
	terminalStates := map[string]bool{"finished": true, "cancelled": true}

	return func(ctx context.Context, resourceID string, data json.RawMessage) error {
		// Validate patient reference
		if err := rv.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
			return err
		}

		currentStatus, err := repo.GetCurrentStatus(ctx, resourceID)
		if err != nil {
			return err
		}

		newStatus := services.ExtractStatusField(data)
		if newStatus == "" || currentStatus == newStatus {
			return nil
		}

		if terminalStates[currentStatus] {
			return fhirerrors.NewConflictError(
				fmt.Sprintf("Cannot change status of %s encounter — %s is a terminal state", resourceID, currentStatus),
			)
		}

		if allowed, ok := validTransitions[currentStatus]; ok {
			if !allowed[newStatus] {
				return fhirerrors.NewValidationError(
					fmt.Sprintf("Invalid status transition from '%s' to '%s'", currentStatus, newStatus),
				)
			}
		}

		return nil
	}
}

// medReqStatusTransitionHook validates MedicationRequest status transitions.
func medReqStatusTransitionHook(repo *repository.BaseRepository, rv *services.ReferenceValidator) func(ctx context.Context, resourceID string, data json.RawMessage) error {
	validTransitions := map[string]map[string]bool{
		"draft":   {"active": true, "cancelled": true},
		"active":  {"on-hold": true, "completed": true, "stopped": true, "cancelled": true},
		"on-hold": {"active": true, "stopped": true, "cancelled": true},
	}
	terminalStates := map[string]bool{"completed": true, "stopped": true, "cancelled": true, "entered-in-error": true}

	return func(ctx context.Context, resourceID string, data json.RawMessage) error {
		// Validate patient reference
		if err := rv.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
			return err
		}
		if err := rv.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter"); err != nil {
			return err
		}

		currentStatus, err := repo.GetCurrentStatus(ctx, resourceID)
		if err != nil {
			return err
		}

		newStatus := services.ExtractStatusField(data)
		if newStatus == "" || currentStatus == newStatus {
			return nil
		}

		if terminalStates[currentStatus] {
			return fhirerrors.NewConflictError(
				fmt.Sprintf("Cannot change status of %s medication request — %s is a terminal state", resourceID, currentStatus),
			)
		}

		if allowed, ok := validTransitions[currentStatus]; ok {
			if !allowed[newStatus] {
				return fhirerrors.NewValidationError(
					fmt.Sprintf("Invalid status transition from '%s' to '%s'", currentStatus, newStatus),
				)
			}
		}

		return nil
	}
}

func setupLogger(level string) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
}

func healthCheckHandler(db *database.PostgresConnections, redis *cache.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := map[string]string{
			"database": "ok",
			"redis":    "ok",
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
			"checks":    checks,
		})
	}
}

func readinessHandler(db *database.PostgresConnections, redis *cache.RedisClient, es search.SearchClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := map[string]string{}
		allOk := true

		if err := db.HealthCheck(); err != nil {
			checks["database"] = "unavailable"
			allOk = false
		} else {
			checks["database"] = "ok"
		}

		if err := redis.HealthCheck(); err != nil {
			checks["redis"] = "unavailable"
			allOk = false
		} else {
			checks["redis"] = "ok"
		}

		if !es.Health(c.Request.Context()) {
			checks["elasticsearch"] = "unavailable"
			allOk = false
		} else {
			checks["elasticsearch"] = "ok"
		}

		status := http.StatusOK
		statusText := "ok"
		if !allOk {
			status = http.StatusServiceUnavailable
			statusText = "unavailable"
		}

		c.JSON(status, gin.H{
			"status":    statusText,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
			"checks":    checks,
		})
	}
}
