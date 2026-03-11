package documents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/documents/llm"
	"github.com/Siddharthk17/MediLink/internal/documents/loinc"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	"github.com/Siddharthk17/MediLink/pkg/storage"
)

// TaskProcessDocument is the Asynq task type for document processing.
const TaskProcessDocument = "document:process"

// DocumentProcessor orchestrates the full OCR → LLM → FHIR pipeline.
type DocumentProcessor struct {
	jobs    DocumentJobRepository
	storage storage.StorageClient
	ocr     OCREngine
	llm     llm.LLMExtractor
	loinc   loinc.LOINCMapper
	db      *sqlx.DB
	notify  notifications.EmailService
	logger  zerolog.Logger
}

// NewDocumentProcessor creates a new DocumentProcessor.
func NewDocumentProcessor(
	jobs DocumentJobRepository,
	store storage.StorageClient,
	ocr OCREngine,
	llmExtractor llm.LLMExtractor,
	loincMapper loinc.LOINCMapper,
	db *sqlx.DB,
	notify notifications.EmailService,
	logger zerolog.Logger,
) *DocumentProcessor {
	return &DocumentProcessor{
		jobs:    jobs,
		storage: store,
		ocr:     ocr,
		llm:     llmExtractor,
		loinc:   loincMapper,
		db:      db,
		notify:  notify,
		logger:  logger,
	}
}

// ProcessDocumentPayload is the task payload.
type ProcessDocumentPayload struct {
	JobID string `json:"jobId"`
}

// ProcessDocument handles an Asynq document processing task.
func (p *DocumentProcessor) ProcessDocument(ctx context.Context, t *asynq.Task) error {
	var payload ProcessDocumentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("parse job ID: %w", err)
	}

	p.logger.Info().Str("jobId", payload.JobID).Msg("processing document")

	// Step 1: Load job from DB
	job, err := p.jobs.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("load job: %w", err)
	}

	// Step 2: Update status to processing
	if err := p.jobs.UpdateProcessingStarted(ctx, jobID); err != nil {
		return fmt.Errorf("update processing started: %w", err)
	}

	// Step 3: Download file from MinIO
	fileBytes, err := p.storage.DownloadFile(ctx, job.MinioBucket, job.MinioKey)
	if err != nil {
		errMsg := "failed to download file from MinIO: " + err.Error()
		_ = p.jobs.UpdateStatus(ctx, jobID, "failed", &errMsg)
		return nil
	}

	// Step 4: Run OCR
	ocrResult, err := p.ocr.ExtractText(ctx, fileBytes, job.ContentType)
	if err != nil || ocrResult.Confidence < 30.0 {
		errMsg := "Image quality too low for OCR"
		if err != nil {
			errMsg = "OCR failed: " + err.Error()
		}
		_ = p.jobs.UpdateStatus(ctx, jobID, "failed", &errMsg)
		return nil // don't retry image quality issues
	}

	// Step 5: Run LLM extraction
	if p.llm == nil {
		errMsg := "no LLM provider available"
		_ = p.jobs.UpdateStatus(ctx, jobID, "failed", &errMsg)
		return nil
	}

	extractedData, err := p.llm.ExtractLabResults(ctx, ocrResult.Text, job.ContentType)
	if err != nil {
		return fmt.Errorf("llm extraction failed: %w", err) // Asynq will retry
	}

	// Step 6: Validate LLM output
	if err := validateExtractionResult(extractedData); err != nil {
		errMsg := "LLM output validation failed: " + err.Error()
		_ = p.jobs.UpdateStatus(ctx, jobID, "needs-manual-review", &errMsg)
		return nil
	}

	// Step 7: Map test names to LOINC codes
	loincMapped := 0
	for i, result := range extractedData.Results {
		if loincResult, found := p.loinc.Lookup(ctx, result.TestName); found {
			extractedData.Results[i].LOINCCode = loincResult.Code
			extractedData.Results[i].LOINCDisplay = loincResult.Display
			loincMapped++
		}
	}

	// Step 8: Create FHIR Observation resources
	observations := make([]string, 0, len(extractedData.Results))
	for _, result := range extractedData.Results {
		obs := buildObservationJSON(result, job)
		obsID, err := createFHIRResource(ctx, p.db, "Observation", obs, job.PatientFHIRID)
		if err != nil {
			p.logger.Error().Err(err).Str("test", result.TestName).Msg("failed to create observation")
			continue
		}
		observations = append(observations, "Observation/"+obsID)
	}

	// Step 9: Create FHIR DiagnosticReport
	report := buildDiagnosticReportJSON(job, observations, extractedData)
	reportID, err := createFHIRResource(ctx, p.db, "DiagnosticReport", report, job.PatientFHIRID)
	if err != nil {
		errMsg := "failed to create DiagnosticReport: " + err.Error()
		_ = p.jobs.UpdateStatus(ctx, jobID, "failed", &errMsg)
		return nil
	}

	// Step 10: Update job record
	llmProvider := ""
	if p.llm != nil {
		llmProvider = p.llm.ProviderName()
	}
	if err := p.jobs.UpdateCompleted(ctx, jobID, reportID, len(observations), loincMapped, ocrResult.Confidence, llmProvider); err != nil {
		p.logger.Error().Err(err).Msg("failed to update job completion")
	}

	p.logger.Info().
		Str("jobId", payload.JobID).
		Int("observations", len(observations)).
		Int("loincMapped", loincMapped).
		Msg("document processing completed")

	return nil
}

// validateExtractionResult checks if the LLM output is valid.
func validateExtractionResult(result *llm.ExtractionResult) error {
	if result == nil {
		return fmt.Errorf("nil extraction result")
	}
	if len(result.Results) == 0 {
		return fmt.Errorf("no test results extracted")
	}
	for i, r := range result.Results {
		if r.TestName == "" {
			return fmt.Errorf("result %d: missing test name", i)
		}
	}
	return nil
}

// buildObservationJSON creates a FHIR Observation JSON from a test result.
func buildObservationJSON(result llm.TestResult, job *DocumentJob) json.RawMessage {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"status":       "final",
		"category": []map[string]interface{}{{
			"coding": []map[string]string{{
				"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
				"code":    "laboratory",
				"display": "Laboratory",
			}},
		}},
		"subject": map[string]string{
			"reference": "Patient/" + job.PatientFHIRID,
		},
	}

	// Code
	code := map[string]interface{}{
		"text": result.TestName,
	}
	if result.LOINCCode != "" {
		code["coding"] = []map[string]string{{
			"system":  "http://loinc.org",
			"code":    result.LOINCCode,
			"display": result.LOINCDisplay,
		}}
	}
	obs["code"] = code

	// Value
	if result.IsNumeric {
		obs["valueQuantity"] = map[string]interface{}{
			"value":  result.Value,
			"unit":   result.Unit,
			"system": "http://unitsofmeasure.org",
		}
	} else if result.ValueString != "" {
		obs["valueString"] = result.ValueString
	}

	// Reference range
	if result.RefRangeLow > 0 || result.RefRangeHigh > 0 {
		refRange := map[string]interface{}{}
		if result.RefRangeLow > 0 {
			refRange["low"] = map[string]interface{}{"value": result.RefRangeLow, "unit": result.Unit}
		}
		if result.RefRangeHigh > 0 {
			refRange["high"] = map[string]interface{}{"value": result.RefRangeHigh, "unit": result.Unit}
		}
		obs["referenceRange"] = []interface{}{refRange}
	}

	// Interpretation
	if result.IsAbnormal && result.AbnormalFlag != "" {
		flagMap := map[string]string{
			"H":  "High",
			"L":  "Low",
			"HH": "Critical high",
			"LL": "Critical low",
		}
		display := flagMap[result.AbnormalFlag]
		if display == "" {
			display = result.AbnormalFlag
		}
		obs["interpretation"] = []map[string]interface{}{{
			"coding": []map[string]string{{
				"system":  "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
				"code":    result.AbnormalFlag,
				"display": display,
			}},
		}}
	}

	data, _ := json.Marshal(obs)
	return data
}

// buildDiagnosticReportJSON creates a FHIR DiagnosticReport JSON.
func buildDiagnosticReportJSON(job *DocumentJob, observationRefs []string, extracted *llm.ExtractionResult) json.RawMessage {
	results := make([]map[string]string, 0, len(observationRefs))
	for _, ref := range observationRefs {
		results = append(results, map[string]string{"reference": ref})
	}

	report := map[string]interface{}{
		"resourceType": "DiagnosticReport",
		"status":       "final",
		"category": []map[string]interface{}{{
			"coding": []map[string]string{{
				"system":  "http://terminology.hl7.org/CodeSystem/v2-0074",
				"code":    "LAB",
				"display": "Laboratory",
			}},
		}},
		"subject": map[string]string{
			"reference": "Patient/" + job.PatientFHIRID,
		},
		"result": results,
		"presentedForm": []map[string]interface{}{{
			"contentType": job.ContentType,
			"title":       job.OriginalFilename,
		}},
	}

	if extracted.ReportDate != "" {
		report["effectiveDateTime"] = extracted.ReportDate
	} else {
		report["effectiveDateTime"] = time.Now().Format("2006-01-02")
	}

	if extracted.ReportType != "" {
		report["code"] = map[string]interface{}{
			"text": extracted.ReportType,
		}
	}

	data, _ := json.Marshal(report)
	return data
}

// createFHIRResource inserts a FHIR resource into the database.
func createFHIRResource(ctx context.Context, db *sqlx.DB, resourceType string, data json.RawMessage, patientFHIRID string) (string, error) {
	resourceID := uuid.New().String()

	// Set server-generated fields
	var resource map[string]interface{}
	if err := json.Unmarshal(data, &resource); err != nil {
		return "", err
	}
	resource["id"] = resourceID
	resource["meta"] = map[string]interface{}{
		"versionId":   "1",
		"lastUpdated": time.Now().Format(time.RFC3339),
	}
	updatedData, _ := json.Marshal(resource)

	// Extract status
	status := "final"
	if s, ok := resource["status"].(string); ok {
		status = s
	}

	_, err := db.ExecContext(ctx,
		`INSERT INTO fhir_resources (resource_type, resource_id, version_id, data, patient_ref, status, created_at, updated_at)
		 VALUES ($1, $2, 1, $3, $4, $5, NOW(), NOW())`,
		resourceType, resourceID, updatedData, "Patient/"+patientFHIRID, status)
	if err != nil {
		return "", fmt.Errorf("insert %s: %w", resourceType, err)
	}

	// Insert history record
	_, _ = db.ExecContext(ctx,
		`INSERT INTO fhir_resource_history (id, resource_type, resource_id, version_id, data, operation, operated_at)
		 VALUES ($1, $2, $3, 1, $4, 'create', NOW())`,
		uuid.New(), resourceType, resourceID, updatedData)

	return resourceID, nil
}
