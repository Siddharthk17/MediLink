package consent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// ConsentMiddleware checks consent on every FHIR read operation.
func ConsentMiddleware(engine ConsentEngine, db *sqlx.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check on GET requests (reads)
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		actorID := c.GetString("actor_id")
		actorRole := c.GetString("actor_role")

		// Admins bypass consent check (but access is logged)
		if actorRole == "admin" {
			c.Set("consent_bypass", true)
			c.Set("access_purpose", "admin")
			c.Next()
			return
		}

		resourceType := extractResourceType(c.FullPath())
		if resourceType == "" {
			c.Next()
			return
		}

		// Unrestricted resources (Practitioner, Organization)
		if !RequiresPatientScope(resourceType) {
			c.Next()
			return
		}

		// Handle search endpoints
		if isSearchEndpoint(c) {
			if actorRole == "patient" {
				// Get patient's FHIR ID and force-scope their search
				fhirID, err := engine.GetPatientFHIRID(c.Request.Context(), actorID)
				if err != nil || fhirID == "" {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "forbidden",
							"diagnostics": "Patient profile not linked",
						}},
					})
					return
				}
				c.Set("forced_patient_ref", "Patient/"+fhirID)
			}

			if actorRole == "physician" {
				patientRef := c.Query("patient")
				if patientRef == "" && RequiresPatientScope(resourceType) {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "required",
							"diagnostics": "Search parameter 'patient' is required for " + resourceType,
						}},
					})
					return
				}

				// Extract the FHIR patient ID from the reference
				patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
				if patientFHIRID != "" {
					granted, err := engine.CheckConsent(c.Request.Context(), actorID, patientFHIRID, resourceType)
					if err != nil {
						logger.Error().Err(err).Msg("consent check error")
						c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
							"resourceType": "OperationOutcome",
							"issue": []gin.H{{
								"severity":    "error",
								"code":        "exception",
								"diagnostics": "Consent check failed",
							}},
						})
						return
					}
					if !granted {
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
							"resourceType": "OperationOutcome",
							"issue": []gin.H{{
								"severity":    "error",
								"code":        "forbidden",
								"diagnostics": "No active consent for this patient's " + resourceType + " resources",
							}},
						})
						return
					}
				}
			}

			c.Next()
			return
		}

		// Handle read endpoints (GET /:id)
		resourceID := c.Param("id")
		if resourceID != "" && actorRole == "patient" {
			// Patient can only read their own resources
			fhirID, err := engine.GetPatientFHIRID(c.Request.Context(), actorID)
			if err != nil || fhirID == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"resourceType": "OperationOutcome",
					"issue": []gin.H{{
						"severity":    "error",
						"code":        "forbidden",
						"diagnostics": "Patient profile not linked",
					}},
				})
				return
			}

			// For Patient resource, the ID IS the patient ref
			if resourceType == "Patient" {
				if resourceID != fhirID {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "forbidden",
							"diagnostics": "Access denied: you can only view your own patient record",
						}},
					})
					return
				}
			} else {
				// For other resources, check patient_ref in DB
				var patientRef string
				err := db.GetContext(c.Request.Context(), &patientRef,
					"SELECT COALESCE(patient_ref, '') FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL",
					resourceID, resourceType)
				if err != nil {
					c.Next() // Let the handler return 404
					return
				}
				expectedRef := "Patient/" + fhirID
				if patientRef != expectedRef {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "forbidden",
							"diagnostics": "Access denied: this resource does not belong to your patient record",
						}},
					})
					return
				}
			}
		}

		if resourceID != "" && actorRole == "physician" {
			// Physician needs consent for the patient this resource belongs to
			var patientRef string
			if resourceType == "Patient" {
				patientRef = "Patient/" + resourceID
			} else {
				err := db.GetContext(c.Request.Context(), &patientRef,
					"SELECT COALESCE(patient_ref, '') FROM fhir_resources WHERE resource_id = $1 AND resource_type = $2 AND deleted_at IS NULL",
					resourceID, resourceType)
				if err != nil {
					c.Next() // Let handler return 404
					return
				}
			}

			patientFHIRID := strings.TrimPrefix(patientRef, "Patient/")
			if patientFHIRID != "" {
				granted, err := engine.CheckConsent(c.Request.Context(), actorID, patientFHIRID, resourceType)
				if err != nil {
					logger.Error().Err(err).Msg("consent check error")
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "exception",
							"diagnostics": "Consent check failed",
						}},
					})
					return
				}
				if !granted {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"resourceType": "OperationOutcome",
						"issue": []gin.H{{
							"severity":    "error",
							"code":        "forbidden",
							"diagnostics": "No active consent for this patient's data",
						}},
					})
					return
				}
			}
		}

		c.Next()
	}
}

// extractResourceType extracts the FHIR resource type from the route path.
// e.g., "/fhir/R4/Patient/:id" → "Patient"
func extractResourceType(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	for i, part := range parts {
		if part == "R4" && i+1 < len(parts) {
			resourceType := parts[i+1]
			// Remove any colon prefix (Gin params)
			if strings.HasPrefix(resourceType, ":") {
				return ""
			}
			return resourceType
		}
	}
	return ""
}

// isSearchEndpoint returns true if this is a search (GET without :id param).
func isSearchEndpoint(c *gin.Context) bool {
	return c.Param("id") == "" && !strings.Contains(c.FullPath(), "$")
}
