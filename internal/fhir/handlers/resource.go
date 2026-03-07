package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Siddharthk17/medilink/internal/auth"
	"github.com/Siddharthk17/medilink/internal/fhir/services"
	fhirerrors "github.com/Siddharthk17/medilink/pkg/errors"
)

// SearchParamParser is a function that extracts resource-specific search parameters
// from a gin.Context and returns SQL conditions, args, count, and offset.
type SearchParamParser func(c *gin.Context) (conditions []string, args []interface{}, argIdx int, count int, offset int)

// ResourceHandler handles HTTP requests for any FHIR resource type.
type ResourceHandler struct {
	service      services.ResourceService
	resourceType string
	searchParser SearchParamParser
}

// NewResourceHandler creates a new ResourceHandler.
func NewResourceHandler(service services.ResourceService, resourceType string, searchParser SearchParamParser) *ResourceHandler {
	return &ResourceHandler{
		service:      service,
		resourceType: resourceType,
		searchParser: searchParser,
	}
}

// Create handles POST /fhir/R4/{ResourceType}.
func (h *ResourceHandler) Create(c *gin.Context) {
	actorID := auth.GetActorID(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("failed to read request body"))
		return
	}

	if !json.Valid(body) {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid JSON in request body"))
		return
	}

	resp, svcErr := h.service.Create(c.Request.Context(), body, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	var m map[string]interface{}
	json.Unmarshal(resp.Data, &m)
	resourceID, _ := m["id"].(string)

	c.Header("Location", "/fhir/R4/"+h.resourceType+"/"+resourceID)
	c.Data(http.StatusCreated, "application/fhir+json; charset=utf-8", resp.Data)
}

// Get handles GET /fhir/R4/{ResourceType}/:id.
func (h *ResourceHandler) Get(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	resp, svcErr := h.service.Get(c.Request.Context(), resourceID, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// Update handles PUT /fhir/R4/{ResourceType}/:id.
func (h *ResourceHandler) Update(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("failed to read request body"))
		return
	}

	if !json.Valid(body) {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid JSON in request body"))
		return
	}

	resp, svcErr := h.service.Update(c.Request.Context(), resourceID, body, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// Delete handles DELETE /fhir/R4/{ResourceType}/:id.
func (h *ResourceHandler) Delete(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	svcErr := h.service.Delete(c.Request.Context(), resourceID, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Status(http.StatusNoContent)
}

// Search handles GET /fhir/R4/{ResourceType}.
func (h *ResourceHandler) Search(c *gin.Context) {
	actorID := auth.GetActorID(c)

	conditions, args, _, count, offset := h.searchParser(c)

	baseURL := getBaseURL(c)
	resp, svcErr := h.service.Search(c.Request.Context(), conditions, args, count, offset, actorID, baseURL)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// GetHistory handles GET /fhir/R4/{ResourceType}/:id/_history.
func (h *ResourceHandler) GetHistory(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")

	count := 20
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	resp, svcErr := h.service.GetHistory(c.Request.Context(), resourceID, actorID, count, offset)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// GetVersion handles GET /fhir/R4/{ResourceType}/:id/_history/:vid.
func (h *ResourceHandler) GetVersion(c *gin.Context) {
	actorID := auth.GetActorID(c)
	resourceID := c.Param("id")
	vidStr := c.Param("vid")

	vid, err := strconv.Atoi(vidStr)
	if err != nil {
		writeFHIRError(c, fhirerrors.NewValidationError("invalid version ID"))
		return
	}

	resp, svcErr := h.service.GetVersion(c.Request.Context(), resourceID, vid, actorID)
	if svcErr != nil {
		if fhirErr, ok := svcErr.(*fhirerrors.FHIRError); ok {
			writeFHIRError(c, fhirErr)
			return
		}
		writeFHIRError(c, fhirerrors.NewProcessingError("internal server error"))
		return
	}

	c.Data(http.StatusOK, "application/fhir+json; charset=utf-8", resp.Data)
}

// parseDatePrefix extracts a comparison operator prefix from a FHIR date search value.
func parseDatePrefix(value string) (string, string) {
	if len(value) > 2 {
		prefix := value[:2]
		switch prefix {
		case "lt":
			return "<", value[2:]
		case "gt":
			return ">", value[2:]
		case "le":
			return "<=", value[2:]
		case "ge":
			return ">=", value[2:]
		case "ne":
			return "!=", value[2:]
		case "eq":
			return "=", value[2:]
		}
	}
	return "=", value
}

// parsePagination extracts _count and _offset query parameters with defaults.
func parsePagination(c *gin.Context) (count, offset int) {
	count = 20
	if countStr := c.Query("_count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}
	if count > 100 {
		count = 100
	}
	offset = 0
	if offsetStr := c.Query("_offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Search param parsers
// ---------------------------------------------------------------------------

// PractitionerSearchParser extracts Practitioner search parameters from a request.
func PractitionerSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if family := c.Query("family"); family != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'name' @> $%d::jsonb`, argIdx))
		familyJSON, _ := json.Marshal([]map[string]string{{"family": family}})
		args = append(args, string(familyJSON))
		argIdx++
	}

	if given := c.Query("given"); given != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE EXISTS (SELECT 1 FROM jsonb_array_elements_text(n->'given') AS g WHERE LOWER(g) = LOWER($%d)))`, argIdx))
		args = append(args, given)
		argIdx++
	}

	if name := c.Query("name"); name != "" {
		namePattern := "%" + name + "%"
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'name') AS n WHERE LOWER(n->>'family') LIKE LOWER($%d) OR LOWER(n->>'text') LIKE LOWER($%d) OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(n->'given') AS g WHERE LOWER(g) LIKE LOWER($%d)))`, argIdx, argIdx, argIdx))
		args = append(args, namePattern)
		argIdx++
	}

	if identifier := c.Query("identifier"); identifier != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'identifier') AS ident WHERE ident->>'value' = $%d)`, argIdx))
		args = append(args, identifier)
		argIdx++
	}

	if gender := c.Query("gender"); gender != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'gender' = $%d`, argIdx))
		args = append(args, gender)
		argIdx++
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		conditions = append(conditions, fmt.Sprintf(`(data->>'active')::boolean = $%d`, argIdx))
		args = append(args, active)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// OrganizationSearchParser extracts Organization search parameters from a request.
func OrganizationSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if name := c.Query("name"); name != "" {
		conditions = append(conditions, fmt.Sprintf(`LOWER(data->>'name') LIKE LOWER($%d)`, argIdx))
		args = append(args, "%"+name+"%")
		argIdx++
	}

	if identifier := c.Query("identifier"); identifier != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'identifier') AS ident WHERE ident->>'value' = $%d)`, argIdx))
		args = append(args, identifier)
		argIdx++
	}

	if typeCode := c.Query("type"); typeCode != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'type') AS t WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(t->'coding') AS c WHERE c->>'code' = $%d))`, argIdx))
		args = append(args, typeCode)
		argIdx++
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		conditions = append(conditions, fmt.Sprintf(`(data->>'active')::boolean = $%d`, argIdx))
		args = append(args, active)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// EncounterSearchParser extracts Encounter search parameters from a request.
func EncounterSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if status := c.Query("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'status' = $%d`, argIdx))
		args = append(args, status)
		argIdx++
	}

	if class := c.Query("class"); class != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'class'->>'code' = $%d`, argIdx))
		args = append(args, class)
		argIdx++
	}

	if dateStr := c.Query("date"); dateStr != "" {
		op, dateVal := parseDatePrefix(dateStr)
		conditions = append(conditions, fmt.Sprintf(`(data->'period'->>'start')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	if practitioner := c.Query("practitioner"); practitioner != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'participant') AS p WHERE p->'individual'->>'reference' = $%d)`, argIdx))
		args = append(args, practitioner)
		argIdx++
	}

	if organization := c.Query("organization"); organization != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'serviceProvider'->>'reference' = $%d`, argIdx))
		args = append(args, organization)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// ConditionSearchParser extracts Condition search parameters from a request.
func ConditionSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if clinicalStatus := c.Query("clinical-status"); clinicalStatus != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'clinicalStatus'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, clinicalStatus)
		argIdx++
	}

	if verificationStatus := c.Query("verification-status"); verificationStatus != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'verificationStatus'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, verificationStatus)
		argIdx++
	}

	if code := c.Query("code"); code != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'code'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, code)
		argIdx++
	}

	if onsetDate := c.Query("onset-date"); onsetDate != "" {
		op, dateVal := parseDatePrefix(onsetDate)
		conditions = append(conditions, fmt.Sprintf(`(data->>'onsetDateTime')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	if encounter := c.Query("encounter"); encounter != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'encounter'->>'reference' = $%d`, argIdx))
		args = append(args, encounter)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// MedicationRequestSearchParser extracts MedicationRequest search parameters from a request.
func MedicationRequestSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if status := c.Query("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'status' = $%d`, argIdx))
		args = append(args, status)
		argIdx++
	}

	if medication := c.Query("medication"); medication != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'medicationCodeableConcept'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, medication)
		argIdx++
	}

	if intent := c.Query("intent"); intent != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'intent' = $%d`, argIdx))
		args = append(args, intent)
		argIdx++
	}

	if encounter := c.Query("encounter"); encounter != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'encounter'->>'reference' = $%d`, argIdx))
		args = append(args, encounter)
		argIdx++
	}

	if authoredOn := c.Query("authored-on"); authoredOn != "" {
		op, dateVal := parseDatePrefix(authoredOn)
		conditions = append(conditions, fmt.Sprintf(`(data->>'authoredOn')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	if requester := c.Query("requester"); requester != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'requester'->>'reference' = $%d`, argIdx))
		args = append(args, requester)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// ObservationSearchParser extracts Observation search parameters from a request.
func ObservationSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if code := c.Query("code"); code != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'code'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, code)
		argIdx++
	}

	if category := c.Query("category"); category != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'category') AS cat WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(cat->'coding') AS c WHERE c->>'code' = $%d))`, argIdx))
		args = append(args, category)
		argIdx++
	}

	if status := c.Query("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'status' = $%d`, argIdx))
		args = append(args, status)
		argIdx++
	}

	if dateStr := c.Query("date"); dateStr != "" {
		op, dateVal := parseDatePrefix(dateStr)
		conditions = append(conditions, fmt.Sprintf(`(data->>'effectiveDateTime')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	if encounter := c.Query("encounter"); encounter != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'encounter'->>'reference' = $%d`, argIdx))
		args = append(args, encounter)
		argIdx++
	}

	if vq := c.Query("value-quantity"); vq != "" {
		op, numVal := parseDatePrefix(vq)
		conditions = append(conditions, fmt.Sprintf(`(data->'valueQuantity'->>'value')::numeric %s $%d::numeric`, op, argIdx))
		args = append(args, numVal)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// DiagnosticReportSearchParser extracts DiagnosticReport search parameters from a request.
func DiagnosticReportSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if status := c.Query("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'status' = $%d`, argIdx))
		args = append(args, status)
		argIdx++
	}

	if category := c.Query("category"); category != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(data->'category') AS cat WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(cat->'coding') AS c WHERE c->>'code' = $%d))`, argIdx))
		args = append(args, category)
		argIdx++
	}

	if code := c.Query("code"); code != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'code'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, code)
		argIdx++
	}

	if dateStr := c.Query("date"); dateStr != "" {
		op, dateVal := parseDatePrefix(dateStr)
		conditions = append(conditions, fmt.Sprintf(`(data->>'effectiveDateTime')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	if encounter := c.Query("encounter"); encounter != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'encounter'->>'reference' = $%d`, argIdx))
		args = append(args, encounter)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// AllergyIntoleranceSearchParser extracts AllergyIntolerance search parameters from a request.
func AllergyIntoleranceSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if clinicalStatus := c.Query("clinical-status"); clinicalStatus != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'clinicalStatus'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, clinicalStatus)
		argIdx++
	}

	if criticality := c.Query("criticality"); criticality != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'criticality' = $%d`, argIdx))
		args = append(args, criticality)
		argIdx++
	}

	if code := c.Query("code"); code != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'code'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, code)
		argIdx++
	}

	if category := c.Query("category"); category != "" {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements_text(data->'category') AS cat WHERE cat = $%d)`, argIdx))
		args = append(args, category)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}

// ImmunizationSearchParser extracts Immunization search parameters from a request.
func ImmunizationSearchParser(c *gin.Context) ([]string, []interface{}, int, int, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if patient := c.Query("patient"); patient != "" {
		conditions = append(conditions, fmt.Sprintf(`patient_ref = $%d`, argIdx))
		args = append(args, patient)
		argIdx++
	}

	if status := c.Query("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf(`data->>'status' = $%d`, argIdx))
		args = append(args, status)
		argIdx++
	}

	if vaccineCode := c.Query("vaccine-code"); vaccineCode != "" {
		conditions = append(conditions, fmt.Sprintf(`data->'vaccineCode'->'coding'->0->>'code' = $%d`, argIdx))
		args = append(args, vaccineCode)
		argIdx++
	}

	if dateStr := c.Query("date"); dateStr != "" {
		op, dateVal := parseDatePrefix(dateStr)
		conditions = append(conditions, fmt.Sprintf(`(data->>'occurrenceDateTime')::date %s $%d`, op, argIdx))
		args = append(args, dateVal)
		argIdx++
	}

	count, offset := parsePagination(c)
	return conditions, args, argIdx, count, offset
}
