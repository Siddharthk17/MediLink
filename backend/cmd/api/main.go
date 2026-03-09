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
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Siddharthk17/MediLink/internal/admin"
	"github.com/Siddharthk17/MediLink/internal/anonymization"
	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/auth"
	"github.com/Siddharthk17/MediLink/internal/clinical"
	"github.com/Siddharthk17/MediLink/internal/config"
	"github.com/Siddharthk17/MediLink/internal/consent"
	"github.com/Siddharthk17/MediLink/internal/documents"
	"github.com/Siddharthk17/MediLink/internal/fhir/handlers"
	"github.com/Siddharthk17/MediLink/internal/fhir/repository"
	"github.com/Siddharthk17/MediLink/internal/fhir/services"
	"github.com/Siddharthk17/MediLink/internal/fhir/validator"
	"github.com/Siddharthk17/MediLink/internal/middleware"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	internalsearch "github.com/Siddharthk17/MediLink/internal/search"
	"github.com/Siddharthk17/MediLink/pkg/cache"
	"github.com/Siddharthk17/MediLink/pkg/crypto"
	"github.com/Siddharthk17/MediLink/pkg/database"
	fhirerrors "github.com/Siddharthk17/MediLink/pkg/errors"
	"github.com/Siddharthk17/MediLink/pkg/metrics"
	"github.com/Siddharthk17/MediLink/pkg/search"
	"github.com/Siddharthk17/MediLink/pkg/storage"
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

	encryptor, _ := crypto.NewAESEncryptor(cfg.Encryption.Key)

	jwtSvc := auth.NewJWTService(cfg.JWT.Secret, redisClient.Client)

	// Email service — use Noop in dev if no Resend key
	var emailSvc notifications.EmailService
	if cfg.Resend.APIKey != "" {
		emailSvc = notifications.NewResendEmailService(cfg.Resend.APIKey, cfg.Resend.FromEmail, log.Logger)
	} else {
		emailSvc = &notifications.NoopEmailService{}
	}

	// Auth system
	authRepo := auth.NewAuthRepository(db.SQLX)
	totpSvc := auth.NewTOTPService(encryptor)
	otpSvc := auth.NewOTPService()
	authService := auth.NewAuthService(authRepo, jwtSvc, totpSvc, otpSvc, encryptor, emailSvc, auditLogger, db.SQLX, log.Logger, redisClient.Client)
	authHandler := auth.NewAuthHandler(authService)

	// Consent system
	consentRepo := consent.NewConsentRepository(db.SQLX)
	consentCache := consent.NewConsentCache(redisClient.Client)
	userLookup := consent.NewSQLUserLookup(db.SQLX)
	consentEngine := consent.NewConsentEngine(consentRepo, consentCache, auditLogger, log.Logger, emailSvc, userLookup, redisClient.Client, db.SQLX)
	consentHandler := consent.NewConsentHandler(consentEngine)

	fhirValidator := validator.NewFHIRValidator()
	refValidator := services.NewReferenceValidator(db.SQLX)

	// Drug Interaction Checker (needed for MedicationRequest pre-create hook)
	drugCheckerRepo := clinical.NewDrugCheckerRepository(db.SQLX, log.Logger)
	openFDAClient := clinical.NewOpenFDAClient(cfg.OpenFDA.BaseURL, log.Logger)
	rxNormClient := clinical.NewRxNormClient(redisClient.Client, log.Logger)
	allergyChecker := clinical.NewAllergyChecker(db.SQLX, rxNormClient, drugCheckerRepo, log.Logger)
	drugChecker := clinical.NewDrugChecker(drugCheckerRepo, openFDAClient, rxNormClient, allergyChecker, db.SQLX, redisClient.Client, log.Logger)
	clinicalHandler := clinical.NewClinicalHandler(drugChecker, auditLogger, log.Logger)

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

	// Asynq client for task queueing (used by FHIR post-create hooks, documents, admin)
	redisAddr := cfg.Redis.URL
	if len(redisAddr) > 8 && redisAddr[:8] == "redis://" {
		redisAddr = redisAddr[8:]
	}
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

	// MedicationRequest — status transitions + patient + encounter ref validation + drug interaction check
	medReqRepo := repository.NewBaseRepository(db.SQLX, esClient, log.Logger, "MedicationRequest")
	medReqService := services.NewResourceService(
		&medReqRepo, fhirValidator.ValidateMedicationRequest, refValidator, auditLogger, "MedicationRequest",
		services.WithPreCreateHook(func(ctx context.Context, data json.RawMessage) error {
			if err := refValidator.ValidateReference(ctx, extractRef(data, "subject"), "Patient"); err != nil {
				return err
			}
			if err := refValidator.ValidateOptionalReference(ctx, extractRef(data, "encounter"), "Encounter"); err != nil {
				return err
			}

			// Drug interaction check — extract RxNorm code and patient FHIR ID
			rxCode := extractRxNormCode(data)
			patientRef := extractRef(data, "subject")
			if rxCode != "" && patientRef != "" {
				patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
				result, err := drugChecker.CheckInteractions(ctx, rxCode, patientFHIRID)
				if err != nil {
					log.Error().Err(err).Msg("drug interaction check failed")
					// Don't block on check failure — proceed with save
				} else if result != nil && result.HasContraindication {
					physicianID := ""
					if actorID, ok := ctx.Value("actor_id").(uuid.UUID); ok {
						physicianID = actorID.String()
					}
					if physicianID == "" || !drugChecker.HasValidAcknowledgment(ctx, physicianID, patientFHIRID, rxCode) {
						return fmt.Errorf("contraindicated drug interaction detected — acknowledge via POST /clinical/drug-check/acknowledge before prescribing")
					}
				}
			}
			return nil
		}),
		services.WithPreUpdateHook(medReqStatusTransitionHook(&medReqRepo, refValidator)),
		services.WithPostCreateHook(func(_ context.Context, resourceID string, data json.RawMessage) {
			patientRef := extractRef(data, "subject")
			patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
			patientUserID := lookupUserIDByFHIRPatient(db, patientFHIRID)
			if patientUserID == "" {
				return
			}
			// Extract medication name for notification body
			medName := "a new medication"
			if rxCode := extractRxNormCode(data); rxCode != "" {
				medName = rxCode
			}
			queuePush(asynqClient, log.Logger, patientUserID, notifications.PushNotification{
				Title: "New Prescription",
				Body:  "A new prescription for " + medName + " has been added to your record.",
				Data: map[string]string{
					"type":          notifications.PushTypeNewPrescription,
					"resourceId":    "MedicationRequest/" + resourceID,
					"patientFhirId": patientFHIRID,
				},
			}, notifications.PushTypeNewPrescription)
		}),
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
		services.WithPostCreateHook(func(_ context.Context, resourceID string, data json.RawMessage) {
			// Only notify for critical lab results (HH or LL interpretation)
			interpCode := extractInterpretationCode(data)
			if interpCode != "HH" && interpCode != "LL" {
				return
			}
			patientRef := extractRef(data, "subject")
			patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
			patientUserID := lookupUserIDByFHIRPatient(db, patientFHIRID)
			if patientUserID == "" {
				return
			}
			flag := "high"
			if interpCode == "LL" {
				flag = "low"
			}
			testName := extractObservationCodeDisplay(data)
			queuePush(asynqClient, log.Logger, patientUserID, notifications.PushNotification{
				Title: "⚠️ Critical Lab Result",
				Body:  testName + " is critically " + flag + ".",
				Data: map[string]string{
					"type":           notifications.PushTypeCriticalLab,
					"resourceId":     "Observation/" + resourceID,
					"patientFhirId":  patientFHIRID,
					"interpretation": interpCode,
				},
			}, notifications.PushTypeCriticalLab)
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
		services.WithPostCreateHook(func(_ context.Context, resourceID string, data json.RawMessage) {
			patientRef := extractRef(data, "subject")
			patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
			patientUserID := lookupUserIDByFHIRPatient(db, patientFHIRID)
			if patientUserID == "" {
				return
			}
			queuePush(asynqClient, log.Logger, patientUserID, notifications.PushNotification{
				Title: "New Lab Results Available",
				Body:  "Your lab report results are now in your health record.",
				Data: map[string]string{
					"type":       notifications.PushTypeLabResultReady,
					"resourceId": "DiagnosticReport/" + resourceID,
				},
			}, notifications.PushTypeLabResultReady)
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

	// MinIO storage
	var storageClient storage.StorageClient
	mc, mcErr := storage.NewMinIOClient(
		cfg.Storage.Endpoint, cfg.Storage.AccessKey, cfg.Storage.SecretKey,
		cfg.Storage.Bucket, cfg.Storage.UseSSL, log.Logger,
	)
	if mcErr != nil {
		log.Warn().Err(mcErr).Msg("MinIO unavailable — using noop storage")
		storageClient = &storage.NoopStorageClient{}
	} else {
		storageClient = mc
	}

	// Document pipeline
	docJobRepo := documents.NewDocumentJobRepository(db.SQLX, log.Logger)
	documentHandler := documents.NewDocumentHandler(docJobRepo, storageClient, asynqClient, cfg.Storage.Bucket, auditLogger, log.Logger)

	// Admin panel
	adminRepo := admin.NewAdminRepository(db.SQLX)
	adminService := admin.NewAdminService(adminRepo, log.Logger)
	adminHandler := admin.NewAdminHandler(adminService, encryptor, asynqClient, log.Logger)

	// Notification preferences + push service
	prefRepo := notifications.NewPostgresPrefsRepository(db.SQLX, log.Logger)
	var pushSvc notifications.PushService
	fcm, fcmErr := notifications.NewFCMPushService(cfg, prefRepo, log.Logger)
	if fcmErr != nil {
		log.Warn().Err(fcmErr).Msg("Firebase unavailable — using noop push service")
		pushSvc = &notifications.NoopPushService{}
	} else {
		pushSvc = fcm
	}
	notifHandler := notifications.NewNotificationHandler(prefRepo, pushSvc, log.Logger)

	// Research exports (anonymization)
	exportRepo := anonymization.NewPostgresExportRepository(db.SQLX)
	researchHandler := anonymization.NewResearchHandler(exportRepo, storageClient, asynqClient, auditLogger, log.Logger)

	// Unified search
	searchSvc := internalsearch.NewSearchService(esClient, db.SQLX, log.Logger)
	searchHandler := internalsearch.NewSearchHandler(searchSvc, db.SQLX, log.Logger)

	// Setup Gin
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// Middleware stack — order matters
	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.RequestLoggingMiddleware(log.Logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(metrics.GinMiddleware())

	// FHIR R4 routes
	v1 := router.Group("/fhir/R4")
	v1.Use(auth.AuthMiddleware(jwtSvc))
	v1.Use(consent.ConsentMiddleware(consentEngine, db.SQLX, log.Logger))
	v1.Use(middleware.RateLimitMiddleware(redisClient.Client))
	{
		// Patient (Week 1)
		patients := v1.Group("/Patient")
		{
			patients.POST("", auth.RequirePhysician(), patientHandler.CreatePatient)
			patients.GET("", patientHandler.SearchPatients)
			patients.GET("/:id", patientHandler.GetPatient)
			patients.PUT("/:id", patientHandler.UpdatePatient)
			patients.DELETE("/:id", auth.RequireRole("admin"), patientHandler.DeletePatient)
			patients.GET("/:id/_history", patientHandler.GetPatientHistory)
			patients.GET("/:id/_history/:vid", patientHandler.GetPatientVersion)
		}

		// Timeline (special endpoint under Patient)
		v1.GET("/Patient/:id/$timeline", timelineHandler.GetTimeline)

		// Week 2 resources — clinical writes require physician role
		clinicalResources := map[string]*handlers.ResourceHandler{
			"Encounter":          encounterHandler,
			"Condition":          conditionHandler,
			"MedicationRequest":  medReqHandler,
			"Observation":        observationHandler,
			"DiagnosticReport":   diagReportHandler,
			"AllergyIntolerance": allergyHandler,
			"Immunization":       immunizationHandler,
		}

		for resourceType, handler := range clinicalResources {
			group := v1.Group("/" + resourceType)
			{
				group.POST("", auth.RequirePhysician(), handler.Create)
				group.GET("", handler.Search)
				group.GET("/:id", handler.Get)
				group.PUT("/:id", auth.RequirePhysician(), handler.Update)
				group.DELETE("/:id", auth.RequireRole("admin"), handler.Delete)
				group.GET("/:id/_history", handler.GetHistory)
				group.GET("/:id/_history/:vid", handler.GetVersion)
			}
		}

		// Practitioner and Organization — admin-only writes
		adminWriteResources := map[string]*handlers.ResourceHandler{
			"Practitioner": practitionerHandler,
			"Organization": organizationHandler,
		}

		for resourceType, handler := range adminWriteResources {
			group := v1.Group("/" + resourceType)
			{
				group.POST("", auth.RequireRole("admin"), handler.Create)
				group.GET("", handler.Search)
				group.GET("/:id", handler.Get)
				group.PUT("/:id", auth.RequireRole("admin"), handler.Update)
				group.DELETE("/:id", auth.RequireRole("admin"), handler.Delete)
				group.GET("/:id/_history", handler.GetHistory)
				group.GET("/:id/_history/:vid", handler.GetVersion)
			}
		}

		// Lab Trends (special endpoint under Observation)
		v1.GET("/Observation/$lab-trends", labTrendsHandler.GetLabTrends)
	}

	// Auth routes — public (no JWT required)
	authPublic := router.Group("/auth")
	authPublic.Use(middleware.AuthRateLimitMiddleware(redisClient.Client, "auth_login"))
	{
		authPublic.POST("/register/physician", authHandler.RegisterPhysician)
		authPublic.POST("/register/patient", authHandler.RegisterPatient)
		authPublic.POST("/login", authHandler.Login)
		authPublic.POST("/refresh", authHandler.RefreshToken)
	}

	// Auth routes — authenticated
	authProtected := router.Group("/auth")
	authProtected.Use(auth.AuthMiddleware(jwtSvc))
	{
		authProtected.POST("/login/verify-totp", authHandler.VerifyTOTP)
		authProtected.POST("/logout", authHandler.Logout)
		authProtected.GET("/me", authHandler.GetMe)
		authProtected.POST("/password/change", authHandler.ChangePassword)
		authProtected.POST("/totp/setup", auth.RequireRole("physician", "admin"), authHandler.SetupTOTP)
		authProtected.POST("/totp/verify-setup", auth.RequireRole("physician", "admin"), authHandler.VerifyTOTPSetup)
	}

	// Consent routes — authenticated
	consentRoutes := router.Group("/consent")
	consentRoutes.Use(auth.AuthMiddleware(jwtSvc))
	{
		consentRoutes.POST("/grant", auth.RequireRole("patient", "admin"), consentHandler.GrantConsent)
		consentRoutes.DELETE("/:consentId/revoke", auth.RequireRole("patient", "physician", "admin"), consentHandler.RevokeConsent)
		consentRoutes.GET("/my-grants", auth.RequireRole("patient", "admin"), consentHandler.GetMyGrants)
		consentRoutes.GET("/my-patients", auth.RequireRole("physician", "admin"), consentHandler.GetMyPatients)
		consentRoutes.GET("/:consentId", consentHandler.GetConsent)
		consentRoutes.POST("/break-glass", auth.RequireRole("physician", "admin"), consentHandler.BreakGlass)
		consentRoutes.GET("/access-log", auth.RequireRole("patient", "admin"), consentHandler.GetAccessLog)
	}

	// Clinical routes — drug interaction checker
	clinicalRoutes := router.Group("/clinical")
	clinicalRoutes.Use(auth.AuthMiddleware(jwtSvc))
	{
		clinicalRoutes.POST("/drug-check", auth.RequirePhysician(), clinicalHandler.CheckDrugInteractions)
		clinicalRoutes.POST("/drug-check/acknowledge", auth.RequirePhysician(), clinicalHandler.AcknowledgeInteraction)
		clinicalRoutes.GET("/drug-check/history/:patientId", auth.RequirePhysician(), clinicalHandler.GetCheckHistory)
	}

	// Document routes — upload / status / list / delete
	documentRoutes := router.Group("/documents")
	documentRoutes.Use(auth.AuthMiddleware(jwtSvc))
	{
		documentRoutes.POST("/upload", documentHandler.UploadDocument)
		documentRoutes.GET("/jobs/:jobId", documentHandler.GetJobStatus)
		documentRoutes.GET("/jobs", documentHandler.ListJobs)
		documentRoutes.DELETE("/jobs/:jobId", documentHandler.DeleteJob)
	}

	// Admin routes — admin only
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(auth.AuthMiddleware(jwtSvc))
	adminRoutes.Use(auth.RequireRole("admin"))
	{
		adminRoutes.POST("/physicians/:userId/approve", func(c *gin.Context) {
			userID, err := uuid.Parse(c.Param("userId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
				return
			}
			adminID := auth.GetActorID(c)
			if err := authService.ApprovePhysician(c.Request.Context(), userID, adminID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"resourceType": "OperationOutcome",
					"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Physician approved"})
		})
		adminRoutes.POST("/physicians/:userId/suspend", func(c *gin.Context) {
			userID, err := uuid.Parse(c.Param("userId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
				return
			}
			adminID := auth.GetActorID(c)
			var body struct {
				Reason string `json:"reason"`
			}
			_ = c.ShouldBindJSON(&body)
			if err := authService.SuspendPhysician(c.Request.Context(), userID, adminID, body.Reason); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"resourceType": "OperationOutcome",
					"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Physician suspended"})
		})
		adminRoutes.POST("/physicians/:userId/reinstate", func(c *gin.Context) {
			userID, err := uuid.Parse(c.Param("userId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
				return
			}
			adminID := auth.GetActorID(c)
			if err := authService.ReinstatePhysician(c.Request.Context(), userID, adminID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"resourceType": "OperationOutcome",
					"issue": []gin.H{{"severity": "error", "code": "invalid", "diagnostics": err.Error()}},
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Physician reinstated"})
		})

		// Admin panel endpoints
		adminRoutes.GET("/users", adminHandler.ListUsers)
		adminRoutes.GET("/users/:userId", adminHandler.GetUser)
		adminRoutes.PUT("/users/:userId/role", adminHandler.UpdateUserRole)
		adminRoutes.POST("/researchers/invite", adminHandler.InviteResearcher)
		adminRoutes.GET("/audit-logs", adminHandler.GetAuditLogs)
		adminRoutes.GET("/audit-logs/patient/:id", adminHandler.GetPatientAuditLog)
		adminRoutes.GET("/audit-logs/actor/:id", adminHandler.GetActorAuditLog)
		adminRoutes.GET("/audit-logs/break-glass", adminHandler.GetBreakGlassEvents)
		adminRoutes.GET("/stats", adminHandler.GetStats)
		adminRoutes.GET("/system/health", adminHandler.GetSystemHealth)
		adminRoutes.POST("/search/reindex", adminHandler.TriggerReindex)
		adminRoutes.POST("/tasks/cleanup-tokens", adminHandler.TriggerTokenCleanup)
	}

	// Notifications
	notif := router.Group("/notifications")
	notif.Use(auth.AuthMiddleware(jwtSvc))
	{
		notif.GET("/preferences", notifHandler.GetPreferences)
		notif.PUT("/preferences", notifHandler.UpdatePreferences)
		notif.POST("/fcm-token", notifHandler.RegisterFCMToken)
		notif.DELETE("/fcm-token", notifHandler.RevokeFCMToken)
	}

	// Research Exports
	research := router.Group("/research")
	research.Use(auth.AuthMiddleware(jwtSvc))
	research.Use(auth.RequireRole("admin", "researcher"))
	{
		research.POST("/export", researchHandler.RequestExport)
		research.GET("/export/:exportId", researchHandler.GetExportStatus)
		research.GET("/exports", researchHandler.ListExports)
		research.DELETE("/export/:exportId", auth.RequireRole("admin"), researchHandler.DeleteExport)
	}

	// Unified Search
	router.GET("/search",
		auth.AuthMiddleware(jwtSvc),
		consent.ConsentMiddleware(consentEngine, db.SQLX, log.Logger),
		searchHandler.UnifiedSearch,
	)

	// Health check endpoints — no auth, no rate limit
	router.GET("/health", healthCheckHandler(db, redisClient))
	router.GET("/ready", readinessHandler(db, redisClient, esClient, storageClient))
	router.GET("/metrics", metrics.Handler())

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

// extractRxNormCode extracts the RxNorm code from a MedicationRequest's medicationCodeableConcept.
func extractRxNormCode(data json.RawMessage) string {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	med, ok := m["medicationCodeableConcept"].(map[string]interface{})
	if !ok {
		return ""
	}
	codings, ok := med["coding"].([]interface{})
	if !ok {
		return ""
	}
	for _, c := range codings {
		coding, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		system, _ := coding["system"].(string)
		code, _ := coding["code"].(string)
		if system == "http://www.nlm.nih.gov/research/umls/rxnorm" && code != "" {
			return code
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

func readinessHandler(db *database.PostgresConnections, redis *cache.RedisClient, es search.SearchClient, sc storage.StorageClient) gin.HandlerFunc {
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

		if !sc.Health(c.Request.Context()) {
			checks["minio"] = "unavailable"
			allOk = false
		} else {
			checks["minio"] = "ok"
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

// queuePush enqueues a push notification task via Asynq. Errors are logged, never returned.
func queuePush(client *asynq.Client, logger zerolog.Logger, userID string, notif notifications.PushNotification, notifType string) {
	task, err := notifications.NewPushTask(userID, notif, notifType)
	if err != nil {
		logger.Error().Err(err).Str("userID", userID).Msg("failed to create push task")
		return
	}
	if _, err := client.Enqueue(task, asynq.Queue("notifications")); err != nil {
		logger.Error().Err(err).Str("userID", userID).Msg("failed to enqueue push task")
	}
}

// lookupUserIDByFHIRPatient resolves a user ID from a FHIR Patient reference.
func lookupUserIDByFHIRPatient(db *database.PostgresConnections, patientFHIRID string) string {
	var userID string
	err := db.SQLX.Get(&userID, "SELECT id::text FROM users WHERE fhir_patient_id = $1", patientFHIRID)
	if err != nil {
		return ""
	}
	return userID
}

// extractInterpretationCode extracts the first interpretation code from an Observation.
func extractInterpretationCode(data json.RawMessage) string {
	var obs struct {
		Interpretation []struct {
			Coding []struct {
				Code string `json:"code"`
			} `json:"coding"`
		} `json:"interpretation"`
	}
	if err := json.Unmarshal(data, &obs); err != nil {
		return ""
	}
	if len(obs.Interpretation) > 0 && len(obs.Interpretation[0].Coding) > 0 {
		return obs.Interpretation[0].Coding[0].Code
	}
	return ""
}

// extractObservationCodeDisplay extracts the display text from an Observation's code.
func extractObservationCodeDisplay(data json.RawMessage) string {
	var obs struct {
		Code struct {
			Text   string `json:"text"`
			Coding []struct {
				Display string `json:"display"`
			} `json:"coding"`
		} `json:"code"`
	}
	if err := json.Unmarshal(data, &obs); err != nil {
		return "Lab result"
	}
	if obs.Code.Text != "" {
		return obs.Code.Text
	}
	if len(obs.Code.Coding) > 0 && obs.Code.Coding[0].Display != "" {
		return obs.Code.Coding[0].Display
	}
	return "Lab result"
}
