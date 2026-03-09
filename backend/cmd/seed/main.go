package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Siddharthk17/MediLink/internal/config"
	"github.com/Siddharthk17/MediLink/pkg/crypto"
	"github.com/Siddharthk17/MediLink/pkg/database"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	email          string
	password       string
	fullName       string
	role           string
	status         string
	mciNumber      *string
	specialization *string
	dob            string
	gender         string
	phone          string
}

func ptr(s string) *string { return &s }

func hashEmail(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h[:])
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	encryptor, err := crypto.NewAESEncryptor(cfg.Encryption.Key)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	db, err := database.NewPostgresConnections(
		cfg.Database.DSN,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.SQLX.Close()

	users := []seedUser{
		{
			email:          "admin@medilink.dev",
			password:       "Admin@Medi2026!",
			fullName:       "System Administrator",
			role:           "admin",
			status:         "active",
			phone:          "+91-9000000001",
		},
		{
			email:          "dr.sharma@medilink.dev",
			password:       "Doctor@Medi2026!",
			fullName:       "Dr. Priya Sharma",
			role:           "physician",
			status:         "active",
			mciNumber:      ptr("MCI/A-100001"),
			specialization: ptr("Cardiology"),
			phone:          "+91-9000000002",
		},
		{
			email:          "dr.patel@medilink.dev",
			password:       "Doctor@Medi2026!",
			fullName:       "Dr. Arjun Patel",
			role:           "physician",
			status:         "active",
			mciNumber:      ptr("MCI/A-100002"),
			specialization: ptr("Endocrinology"),
			phone:          "+91-9000000003",
		},
		{
			email:          "dr.pending@medilink.dev",
			password:       "Doctor@Medi2026!",
			fullName:       "Dr. Pending Approval",
			role:           "physician",
			status:         "pending",
			mciNumber:      ptr("MCI/A-100003"),
			specialization: ptr("Neurology"),
			phone:          "+91-9000000004",
		},
		{
			email:          "patient.meera@medilink.dev",
			password:       "Patient@Medi2026!",
			fullName:       "Meera Krishnan",
			role:           "patient",
			status:         "active",
			dob:            "1990-06-15",
			gender:         "Female",
			phone:          "+91-9000000005",
		},
		{
			email:          "patient.rahul@medilink.dev",
			password:       "Patient@Medi2026!",
			fullName:       "Rahul Verma",
			role:           "patient",
			status:         "active",
			dob:            "1985-03-22",
			gender:         "Male",
			phone:          "+91-9000000006",
		},
	}

	ctx := context.Background()
	now := time.Now()

	for _, u := range users {
		emailHash := hashEmail(u.email)

		// Check if user already exists
		var exists bool
		err := db.SQLX.GetContext(ctx, &exists,
			"SELECT EXISTS(SELECT 1 FROM users WHERE email_hash = $1 AND deleted_at IS NULL)", emailHash)
		if err != nil {
			log.Printf("⚠ Error checking user %s: %v", u.email, err)
			continue
		}
		if exists {
			fmt.Printf("⏭ User %s already exists, skipping\n", u.email)
			continue
		}

		pwHash, err := hashPassword(u.password)
		if err != nil {
			log.Printf("⚠ Failed to hash password for %s: %v", u.email, err)
			continue
		}

		emailEnc, err := encryptor.EncryptString(u.email)
		if err != nil {
			log.Printf("⚠ Failed to encrypt email for %s: %v", u.email, err)
			continue
		}

		nameEnc, err := encryptor.EncryptString(u.fullName)
		if err != nil {
			log.Printf("⚠ Failed to encrypt name for %s: %v", u.email, err)
			continue
		}

		var phoneEnc []byte
		if u.phone != "" {
			phoneEnc, err = encryptor.EncryptString(u.phone)
			if err != nil {
				log.Printf("⚠ Failed to encrypt phone for %s: %v", u.email, err)
				continue
			}
		}

		var dobEnc []byte
		if u.dob != "" {
			dobEnc, err = encryptor.EncryptString(u.dob)
			if err != nil {
				log.Printf("⚠ Failed to encrypt DOB for %s: %v", u.email, err)
				continue
			}
		}

		userID := uuid.New()

		query := `INSERT INTO users (
			id, email, email_hash, email_enc, password_hash,
			role, status, full_name_enc, phone_enc, dob_enc,
			mci_number, specialization,
			totp_enabled, totp_verified,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12,
			false, false,
			$13, $14
		)`

		_, err = db.SQLX.ExecContext(ctx, query,
			userID, u.email, emailHash, emailEnc, pwHash,
			u.role, u.status, nameEnc, phoneEnc, dobEnc,
			u.mciNumber, u.specialization,
			now, now,
		)
		if err != nil {
			log.Printf("✗ Failed to create user %s: %v", u.email, err)
			continue
		}

		// Create FHIR Patient resource for patient-role users
		if u.role == "patient" {
			fhirID := uuid.New().String()
			fhirData := fmt.Sprintf(`{
				"resourceType": "Patient",
				"id": "%s",
				"active": true,
				"name": [{"use": "official", "text": "%s"}],
				"gender": "%s",
				"birthDate": "%s"
			}`, fhirID, u.fullName, strings.ToLower(u.gender), u.dob)

			_, err = db.SQLX.ExecContext(ctx,
				`INSERT INTO fhir_resources (resource_type, resource_id, data, status, created_by)
				 VALUES ('Patient', $1, $2, 'active', $3)`,
				fhirID, fhirData, userID,
			)
			if err != nil {
				log.Printf("⚠ Failed to create FHIR resource for %s: %v", u.email, err)
			}

			_, err = db.SQLX.ExecContext(ctx,
				`UPDATE users SET fhir_patient_id = $1 WHERE id = $2`,
				fhirID, userID,
			)
			if err != nil {
				log.Printf("⚠ Failed to link FHIR ID for %s: %v", u.email, err)
			}
		}

		fmt.Printf("✓ Created %s: %s (%s) [%s]\n", u.role, u.fullName, u.email, u.status)
	}

	// ── Seed consent records: patients grant access to Dr. Sharma ──
	seedConsents(ctx, db.SQLX)

	// ── Seed FHIR clinical data for timeline ──
	seedClinicalData(ctx, db.SQLX)

	// ── Seed additional consents (both patients → Dr. Patel) ──
	seedAdditionalConsents(ctx, db.SQLX)

	// ── Seed Rahul's clinical data ──
	seedRahulClinicalData(ctx, db.SQLX)

	// ── Seed extra lab observations for Meera (for lab trends) ──
	seedMeeraLabTrends(ctx, db.SQLX)

	// ── Seed document jobs ──
	seedDocumentJobs(ctx, db.SQLX)

	// ── Seed audit logs ──
	seedAuditLogs(ctx, db.SQLX)

	// ── Seed notification preferences ──
	seedNotificationPreferences(ctx, db.SQLX)

	// ── Seed drug interactions and classes ──
	seedDrugData(ctx, db.SQLX)

	fmt.Println("\n══════════════════════════════════════════════")
	fmt.Println("  MediLink Test Credentials")
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  ADMIN:")
	fmt.Println("    Email:    admin@medilink.dev")
	fmt.Println("    Password: Admin@Medi2026!")
	fmt.Println()
	fmt.Println("  PHYSICIANS (active):")
	fmt.Println("    Email:    dr.sharma@medilink.dev")
	fmt.Println("    Password: Doctor@Medi2026!")
	fmt.Println()
	fmt.Println("    Email:    dr.patel@medilink.dev")
	fmt.Println("    Password: Doctor@Medi2026!")
	fmt.Println()
	fmt.Println("  PHYSICIAN (pending approval):")
	fmt.Println("    Email:    dr.pending@medilink.dev")
	fmt.Println("    Password: Doctor@Medi2026!")
	fmt.Println()
	fmt.Println("  PATIENTS:")
	fmt.Println("    Email:    patient.meera@medilink.dev")
	fmt.Println("    Password: Patient@Medi2026!")
	fmt.Println()
	fmt.Println("    Email:    patient.rahul@medilink.dev")
	fmt.Println("    Password: Patient@Medi2026!")
	fmt.Println("══════════════════════════════════════════════")

	os.Exit(0)
}

// ── helper to look up a user by email ──
type userRef struct {
	ID            string  `db:"id"`
	FHIRPatientID *string `db:"fhir_patient_id"`
}

func lookupUser(ctx context.Context, db *sqlx.DB, email string) (*userRef, error) {
	var u userRef
	err := db.GetContext(ctx, &u,
		"SELECT id, fhir_patient_id FROM users WHERE email_hash = $1 AND deleted_at IS NULL",
		hashEmail(email))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ── helper to insert FHIR resource if not exists ──
func insertFHIR(ctx context.Context, db *sqlx.DB, resType, resID, data, patientRef, createdBy string) {
	var exists bool
	_ = db.GetContext(ctx, &exists,
		"SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND deleted_at IS NULL)", resID)
	if exists {
		fmt.Printf("⏭ %s/%s already exists\n", resType, resID[:8])
		return
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO fhir_resources (resource_type, resource_id, data, patient_ref, status, created_by)
		 VALUES ($1, $2, $3, $4, 'active', $5)`,
		resType, resID, data, patientRef, createdBy)
	if err != nil {
		fmt.Printf("✗ Failed to create %s/%s: %v\n", resType, resID[:8], err)
	} else {
		fmt.Printf("✓ Created %s: %s\n", resType, resID[:8])
	}
}

func seedConsents(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding consent records ──")

	patients := []string{"patient.meera@medilink.dev", "patient.rahul@medilink.dev"}
	physicians := []string{"dr.sharma@medilink.dev"}

	var physicianIDs []string
	for _, email := range physicians {
		u, err := lookupUser(ctx, db, email)
		if err != nil {
			fmt.Printf("⚠ Physician %s not found: %v\n", email, err)
			continue
		}
		physicianIDs = append(physicianIDs, u.ID)
	}

	for _, email := range patients {
		u, err := lookupUser(ctx, db, email)
		if err != nil {
			fmt.Printf("⚠ Patient %s not found: %v\n", email, err)
			continue
		}

		for _, physID := range physicianIDs {
			var exists bool
			_ = db.GetContext(ctx, &exists,
				"SELECT EXISTS(SELECT 1 FROM consents WHERE patient_id = $1 AND provider_id = $2 AND revoked_at IS NULL)",
				u.ID, physID)
			if exists {
				fmt.Printf("⏭ Consent %s → %s already exists\n", email, physID[:8])
				continue
			}

			_, err = db.ExecContext(ctx,
				`INSERT INTO consents (patient_id, provider_id, scope, purpose, notes, created_by)
				 VALUES ($1, $2, '["*"]', 'treatment', 'Seeded test consent', $1)`,
				u.ID, physID)
			if err != nil {
				fmt.Printf("✗ Failed to create consent %s → %s: %v\n", email, physID[:8], err)
			} else {
				fmt.Printf("✓ Consent: %s → physician %s\n", email, physID[:8])
			}
		}
	}
}

func seedClinicalData(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding FHIR clinical data ──")

	meera, err := lookupUser(ctx, db, "patient.meera@medilink.dev")
	if err != nil || meera.FHIRPatientID == nil {
		fmt.Println("⚠ Patient Meera not found or no FHIR ID, skipping clinical data")
		return
	}

	drSharma, err := lookupUser(ctx, db, "dr.sharma@medilink.dev")
	if err != nil {
		fmt.Println("⚠ Dr. Sharma not found, skipping clinical data")
		return
	}

	patientRef := "Patient/" + *meera.FHIRPatientID
	drID := drSharma.ID

	type clinicalResource struct {
		resourceType string
		resourceID   string
		data         string
	}

	encID1 := uuid.New().String()
	encID2 := uuid.New().String()
	condID1 := uuid.New().String()
	condID2 := uuid.New().String()
	obsID1 := uuid.New().String()
	obsID2 := uuid.New().String()
	obsID3 := uuid.New().String()
	medID1 := uuid.New().String()

	resources := []clinicalResource{
		{
			resourceType: "Encounter",
			resourceID:   encID1,
			data: fmt.Sprintf(`{
				"resourceType": "Encounter",
				"id": "%s",
				"status": "finished",
				"class": {"code": "AMB", "display": "ambulatory"},
				"type": [{"coding": [{"system": "http://snomed.info/sct", "code": "185349003", "display": "Encounter for check up"}]}],
				"subject": {"reference": "%s"},
				"participant": [{"individual": {"display": "Dr. Priya Sharma"}}],
				"period": {"start": "2026-01-15T09:00:00Z", "end": "2026-01-15T09:30:00Z"},
				"reasonCode": [{"text": "Annual cardiovascular checkup"}]
			}`, encID1, patientRef),
		},
		{
			resourceType: "Encounter",
			resourceID:   encID2,
			data: fmt.Sprintf(`{
				"resourceType": "Encounter",
				"id": "%s",
				"status": "finished",
				"class": {"code": "AMB", "display": "ambulatory"},
				"type": [{"coding": [{"system": "http://snomed.info/sct", "code": "185349003", "display": "Encounter for follow up"}]}],
				"subject": {"reference": "%s"},
				"participant": [{"individual": {"display": "Dr. Priya Sharma"}}],
				"period": {"start": "2026-02-20T10:00:00Z", "end": "2026-02-20T10:45:00Z"},
				"reasonCode": [{"text": "Follow-up: blood pressure monitoring"}]
			}`, encID2, patientRef),
		},
		{
			resourceType: "Condition",
			resourceID:   condID1,
			data: fmt.Sprintf(`{
				"resourceType": "Condition",
				"id": "%s",
				"clinicalStatus": {"coding": [{"code": "active"}]},
				"verificationStatus": {"coding": [{"code": "confirmed"}]},
				"category": [{"coding": [{"code": "encounter-diagnosis", "display": "Encounter Diagnosis"}]}],
				"code": {"coding": [{"system": "http://snomed.info/sct", "code": "38341003", "display": "Hypertension"}], "text": "Essential Hypertension"},
				"subject": {"reference": "%s"},
				"onsetDateTime": "2025-06-01",
				"recorder": {"display": "Dr. Priya Sharma"}
			}`, condID1, patientRef),
		},
		{
			resourceType: "Condition",
			resourceID:   condID2,
			data: fmt.Sprintf(`{
				"resourceType": "Condition",
				"id": "%s",
				"clinicalStatus": {"coding": [{"code": "active"}]},
				"verificationStatus": {"coding": [{"code": "confirmed"}]},
				"code": {"coding": [{"system": "http://snomed.info/sct", "code": "44054006", "display": "Type 2 diabetes mellitus"}], "text": "Type 2 Diabetes"},
				"subject": {"reference": "%s"},
				"onsetDateTime": "2024-11-15",
				"recorder": {"display": "Dr. Priya Sharma"}
			}`, condID2, patientRef),
		},
		{
			resourceType: "Observation",
			resourceID:   obsID1,
			data: fmt.Sprintf(`{
				"resourceType": "Observation",
				"id": "%s",
				"status": "final",
				"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "vital-signs"}]}],
				"code": {"coding": [{"system": "http://loinc.org", "code": "85354-9", "display": "Blood pressure panel"}], "text": "Blood Pressure"},
				"subject": {"reference": "%s"},
				"effectiveDateTime": "2026-02-20T10:15:00Z",
				"component": [
					{"code": {"coding": [{"code": "8480-6", "display": "Systolic"}]}, "valueQuantity": {"value": 142, "unit": "mmHg"}},
					{"code": {"coding": [{"code": "8462-4", "display": "Diastolic"}]}, "valueQuantity": {"value": 88, "unit": "mmHg"}}
				]
			}`, obsID1, patientRef),
		},
		{
			resourceType: "Observation",
			resourceID:   obsID2,
			data: fmt.Sprintf(`{
				"resourceType": "Observation",
				"id": "%s",
				"status": "final",
				"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
				"code": {"coding": [{"system": "http://loinc.org", "code": "4548-4", "display": "HbA1c"}], "text": "Hemoglobin A1c"},
				"subject": {"reference": "%s"},
				"effectiveDateTime": "2026-01-15T09:20:00Z",
				"valueQuantity": {"value": 7.2, "unit": "%%"}
			}`, obsID2, patientRef),
		},
		{
			resourceType: "Observation",
			resourceID:   obsID3,
			data: fmt.Sprintf(`{
				"resourceType": "Observation",
				"id": "%s",
				"status": "final",
				"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "vital-signs"}]}],
				"code": {"coding": [{"system": "http://loinc.org", "code": "29463-7", "display": "Body weight"}], "text": "Body Weight"},
				"subject": {"reference": "%s"},
				"effectiveDateTime": "2026-01-15T09:10:00Z",
				"valueQuantity": {"value": 72.5, "unit": "kg"}
			}`, obsID3, patientRef),
		},
		{
			resourceType: "MedicationRequest",
			resourceID:   medID1,
			data: fmt.Sprintf(`{
				"resourceType": "MedicationRequest",
				"id": "%s",
				"status": "active",
				"intent": "order",
				"medicationCodeableConcept": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "200031", "display": "Amlodipine 5mg"}], "text": "Amlodipine 5mg tablet"},
				"subject": {"reference": "%s"},
				"authoredOn": "2026-01-15",
				"requester": {"display": "Dr. Priya Sharma"},
				"dosageInstruction": [{"text": "Take 1 tablet daily in the morning", "timing": {"repeat": {"frequency": 1, "period": 1, "periodUnit": "d"}}}]
			}`, medID1, patientRef),
		},
	}

	for _, r := range resources {
		var exists bool
		err := db.GetContext(ctx, &exists,
			"SELECT EXISTS(SELECT 1 FROM fhir_resources WHERE resource_id = $1 AND deleted_at IS NULL)",
			r.resourceID)
		if err != nil {
			fmt.Printf("⚠ Error checking %s/%s: %v\n", r.resourceType, r.resourceID[:8], err)
			continue
		}
		if exists {
			fmt.Printf("⏭ %s/%s already exists\n", r.resourceType, r.resourceID[:8])
			continue
		}

		_, err = db.ExecContext(ctx,
			`INSERT INTO fhir_resources (resource_type, resource_id, data, patient_ref, status, created_by)
			 VALUES ($1, $2, $3, $4, 'active', $5)`,
			r.resourceType, r.resourceID, r.data, patientRef, drID)
		if err != nil {
			fmt.Printf("✗ Failed to create %s/%s: %v\n", r.resourceType, r.resourceID[:8], err)
		} else {
			fmt.Printf("✓ Created %s: %s\n", r.resourceType, r.resourceID[:8])
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Additional consents: both patients → Dr. Patel
// ═══════════════════════════════════════════════════════════════
func seedAdditionalConsents(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding additional consent records ──")

	drPatel, err := lookupUser(ctx, db, "dr.patel@medilink.dev")
	if err != nil {
		fmt.Println("⚠ Dr. Patel not found, skipping additional consents")
		return
	}

	patients := []string{"patient.meera@medilink.dev", "patient.rahul@medilink.dev"}
	for _, email := range patients {
		p, err := lookupUser(ctx, db, email)
		if err != nil {
			fmt.Printf("⚠ Patient %s not found: %v\n", email, err)
			continue
		}

		var exists bool
		_ = db.GetContext(ctx, &exists,
			"SELECT EXISTS(SELECT 1 FROM consents WHERE patient_id = $1 AND provider_id = $2 AND revoked_at IS NULL)",
			p.ID, drPatel.ID)
		if exists {
			fmt.Printf("⏭ Consent %s → Dr. Patel already exists\n", email)
			continue
		}

		_, err = db.ExecContext(ctx,
			`INSERT INTO consents (patient_id, provider_id, scope, purpose, notes, created_by)
			 VALUES ($1, $2, '["*"]', 'treatment', 'Seeded test consent', $1)`,
			p.ID, drPatel.ID)
		if err != nil {
			fmt.Printf("✗ Failed to create consent %s → Dr. Patel: %v\n", email, err)
		} else {
			fmt.Printf("✓ Consent: %s → Dr. Patel\n", email)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Clinical data for Rahul
// ═══════════════════════════════════════════════════════════════
func seedRahulClinicalData(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding Rahul's clinical data ──")

	rahul, err := lookupUser(ctx, db, "patient.rahul@medilink.dev")
	if err != nil || rahul.FHIRPatientID == nil {
		fmt.Println("⚠ Rahul not found or no FHIR ID, skipping")
		return
	}
	drSharma, err := lookupUser(ctx, db, "dr.sharma@medilink.dev")
	if err != nil {
		fmt.Println("⚠ Dr. Sharma not found, skipping")
		return
	}

	patientRef := "Patient/" + *rahul.FHIRPatientID
	drID := drSharma.ID

	// Encounters
	enc1 := uuid.New().String()
	enc2 := uuid.New().String()
	enc3 := uuid.New().String()
	insertFHIR(ctx, db, "Encounter", enc1, fmt.Sprintf(`{
		"resourceType": "Encounter", "id": "%s", "status": "finished",
		"class": {"code": "AMB", "display": "ambulatory"},
		"type": [{"coding": [{"system": "http://snomed.info/sct", "code": "185349003", "display": "Encounter for check up"}]}],
		"subject": {"reference": "%s"},
		"participant": [{"individual": {"display": "Dr. Priya Sharma"}}],
		"period": {"start": "2026-01-10T11:00:00Z", "end": "2026-01-10T11:30:00Z"},
		"reasonCode": [{"text": "Routine check-up with lipid review"}]
	}`, enc1, patientRef), patientRef, drID)

	insertFHIR(ctx, db, "Encounter", enc2, fmt.Sprintf(`{
		"resourceType": "Encounter", "id": "%s", "status": "finished",
		"class": {"code": "AMB", "display": "ambulatory"},
		"type": [{"coding": [{"system": "http://snomed.info/sct", "code": "185349003", "display": "Encounter for follow up"}]}],
		"subject": {"reference": "%s"},
		"participant": [{"individual": {"display": "Dr. Priya Sharma"}}],
		"period": {"start": "2026-02-25T14:00:00Z", "end": "2026-02-25T14:40:00Z"},
		"reasonCode": [{"text": "Follow-up: cholesterol management"}]
	}`, enc2, patientRef), patientRef, drID)

	insertFHIR(ctx, db, "Encounter", enc3, fmt.Sprintf(`{
		"resourceType": "Encounter", "id": "%s", "status": "finished",
		"class": {"code": "AMB", "display": "ambulatory"},
		"subject": {"reference": "%s"},
		"participant": [{"individual": {"display": "Dr. Arjun Patel"}}],
		"period": {"start": "2026-03-05T09:00:00Z", "end": "2026-03-05T09:45:00Z"},
		"reasonCode": [{"text": "Thyroid function review"}]
	}`, enc3, patientRef), patientRef, drID)

	// Conditions
	cond1 := uuid.New().String()
	cond2 := uuid.New().String()
	insertFHIR(ctx, db, "Condition", cond1, fmt.Sprintf(`{
		"resourceType": "Condition", "id": "%s",
		"clinicalStatus": {"coding": [{"code": "active"}]},
		"verificationStatus": {"coding": [{"code": "confirmed"}]},
		"code": {"coding": [{"system": "http://snomed.info/sct", "code": "13644009", "display": "Hypercholesterolemia"}], "text": "High Cholesterol"},
		"subject": {"reference": "%s"},
		"onsetDateTime": "2025-08-10",
		"recorder": {"display": "Dr. Priya Sharma"}
	}`, cond1, patientRef), patientRef, drID)

	insertFHIR(ctx, db, "Condition", cond2, fmt.Sprintf(`{
		"resourceType": "Condition", "id": "%s",
		"clinicalStatus": {"coding": [{"code": "active"}]},
		"verificationStatus": {"coding": [{"code": "confirmed"}]},
		"code": {"coding": [{"system": "http://snomed.info/sct", "code": "40930008", "display": "Hypothyroidism"}], "text": "Hypothyroidism"},
		"subject": {"reference": "%s"},
		"onsetDateTime": "2025-11-20",
		"recorder": {"display": "Dr. Arjun Patel"}
	}`, cond2, patientRef), patientRef, drID)

	// Observations - Vitals
	obs1 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs1, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "vital-signs"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "85354-9", "display": "Blood pressure panel"}], "text": "Blood Pressure"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-02-25T14:10:00Z",
		"component": [
			{"code": {"coding": [{"code": "8480-6", "display": "Systolic"}]}, "valueQuantity": {"value": 128, "unit": "mmHg"}},
			{"code": {"coding": [{"code": "8462-4", "display": "Diastolic"}]}, "valueQuantity": {"value": 82, "unit": "mmHg"}}
		]
	}`, obs1, patientRef), patientRef, drID)

	obs2 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs2, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "vital-signs"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "29463-7", "display": "Body weight"}], "text": "Body Weight"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-02-25T14:05:00Z",
		"valueQuantity": {"value": 85.3, "unit": "kg"}
	}`, obs2, patientRef), patientRef, drID)

	// Observations - Labs
	obs3 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs3, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "2093-3", "display": "Cholesterol [Mass/volume] in Serum or Plasma"}], "text": "Total Cholesterol"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-01-10T11:20:00Z",
		"valueQuantity": {"value": 252, "unit": "mg/dL"},
		"referenceRange": [{"low": {"value": 0, "unit": "mg/dL"}, "high": {"value": 200, "unit": "mg/dL"}, "text": "Desirable: < 200 mg/dL"}],
		"interpretation": [{"coding": [{"code": "H", "display": "High"}]}]
	}`, obs3, patientRef), patientRef, drID)

	obs4 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs4, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "3016-3", "display": "TSH"}], "text": "Thyroid Stimulating Hormone"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-03-05T09:20:00Z",
		"valueQuantity": {"value": 6.8, "unit": "mIU/L"},
		"referenceRange": [{"low": {"value": 0.4, "unit": "mIU/L"}, "high": {"value": 4.0, "unit": "mIU/L"}}],
		"interpretation": [{"coding": [{"code": "H", "display": "High"}]}]
	}`, obs4, patientRef), patientRef, drID)

	obs5 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs5, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "718-7", "display": "Hemoglobin [Mass/volume] in Blood"}], "text": "Hemoglobin"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-01-10T11:20:00Z",
		"valueQuantity": {"value": 14.2, "unit": "g/dL"},
		"referenceRange": [{"low": {"value": 13.5, "unit": "g/dL"}, "high": {"value": 17.5, "unit": "g/dL"}}]
	}`, obs5, patientRef), patientRef, drID)

	obs6 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs6, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine [Mass/volume] in Serum or Plasma"}], "text": "Creatinine"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-01-10T11:20:00Z",
		"valueQuantity": {"value": 0.9, "unit": "mg/dL"},
		"referenceRange": [{"low": {"value": 0.7, "unit": "mg/dL"}, "high": {"value": 1.3, "unit": "mg/dL"}}]
	}`, obs6, patientRef), patientRef, drID)

	obs7 := uuid.New().String()
	insertFHIR(ctx, db, "Observation", obs7, fmt.Sprintf(`{
		"resourceType": "Observation", "id": "%s", "status": "final",
		"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
		"code": {"coding": [{"system": "http://loinc.org", "code": "1742-6", "display": "ALT [Enzymatic activity/volume] in Serum or Plasma"}], "text": "ALT (SGPT)"},
		"subject": {"reference": "%s"},
		"effectiveDateTime": "2026-01-10T11:20:00Z",
		"valueQuantity": {"value": 28, "unit": "U/L"},
		"referenceRange": [{"low": {"value": 7, "unit": "U/L"}, "high": {"value": 56, "unit": "U/L"}}]
	}`, obs7, patientRef), patientRef, drID)

	// MedicationRequests
	med1 := uuid.New().String()
	med2 := uuid.New().String()
	insertFHIR(ctx, db, "MedicationRequest", med1, fmt.Sprintf(`{
		"resourceType": "MedicationRequest", "id": "%s", "status": "active", "intent": "order",
		"medicationCodeableConcept": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "36567", "display": "Atorvastatin 20mg"}], "text": "Atorvastatin 20mg tablet"},
		"subject": {"reference": "%s"},
		"authoredOn": "2026-01-10",
		"requester": {"display": "Dr. Priya Sharma"},
		"dosageInstruction": [{"text": "Take 1 tablet daily at bedtime", "timing": {"repeat": {"frequency": 1, "period": 1, "periodUnit": "d"}}}]
	}`, med1, patientRef), patientRef, drID)

	insertFHIR(ctx, db, "MedicationRequest", med2, fmt.Sprintf(`{
		"resourceType": "MedicationRequest", "id": "%s", "status": "active", "intent": "order",
		"medicationCodeableConcept": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "10582", "display": "Levothyroxine 50mcg"}], "text": "Levothyroxine 50mcg tablet"},
		"subject": {"reference": "%s"},
		"authoredOn": "2026-03-05",
		"requester": {"display": "Dr. Arjun Patel"},
		"dosageInstruction": [{"text": "Take 1 tablet on empty stomach in the morning", "timing": {"repeat": {"frequency": 1, "period": 1, "periodUnit": "d"}}}]
	}`, med2, patientRef), patientRef, drID)

	// AllergyIntolerance
	allergy1 := uuid.New().String()
	insertFHIR(ctx, db, "AllergyIntolerance", allergy1, fmt.Sprintf(`{
		"resourceType": "AllergyIntolerance", "id": "%s",
		"clinicalStatus": {"coding": [{"code": "active"}]},
		"verificationStatus": {"coding": [{"code": "confirmed"}]},
		"type": "allergy",
		"category": ["medication"],
		"criticality": "high",
		"code": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "723", "display": "Amoxicillin"}], "text": "Amoxicillin"},
		"patient": {"reference": "%s"},
		"recordedDate": "2025-05-12",
		"recorder": {"display": "Dr. Priya Sharma"},
		"reaction": [{"manifestation": [{"coding": [{"display": "Anaphylaxis"}], "text": "Severe anaphylactic reaction"}], "severity": "severe"}]
	}`, allergy1, patientRef), patientRef, drID)
}

// ═══════════════════════════════════════════════════════════════
// Extra lab observations for Meera (historical data for lab trends)
// ═══════════════════════════════════════════════════════════════
func seedMeeraLabTrends(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding Meera's lab trend data ──")

	meera, err := lookupUser(ctx, db, "patient.meera@medilink.dev")
	if err != nil || meera.FHIRPatientID == nil {
		fmt.Println("⚠ Meera not found or no FHIR ID, skipping")
		return
	}
	drSharma, err := lookupUser(ctx, db, "dr.sharma@medilink.dev")
	if err != nil {
		return
	}

	patientRef := "Patient/" + *meera.FHIRPatientID
	drID := drSharma.ID

	// HbA1c historical readings (4548-4) — creates a trend
	hba1cReadings := []struct{ date string; value float64 }{
		{"2025-07-15T09:00:00Z", 8.1},
		{"2025-10-10T09:00:00Z", 7.6},
		{"2026-03-01T09:00:00Z", 7.0},
	}
	for _, r := range hba1cReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "4548-4", "display": "HbA1c"}], "text": "Hemoglobin A1c"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.1f, "unit": "%%"},
			"referenceRange": [{"low": {"value": 4.0, "unit": "%%"}, "high": {"value": 5.6, "unit": "%%"}, "text": "Normal: < 5.7%%"}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// Creatinine historical readings (2160-0)
	creatReadings := []struct{ date string; value float64 }{
		{"2025-07-15T09:00:00Z", 0.8},
		{"2025-10-10T09:00:00Z", 0.9},
		{"2026-01-15T09:00:00Z", 0.85},
		{"2026-03-01T09:00:00Z", 0.82},
	}
	for _, r := range creatReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine"}], "text": "Creatinine"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.2f, "unit": "mg/dL"},
			"referenceRange": [{"low": {"value": 0.6, "unit": "mg/dL"}, "high": {"value": 1.2, "unit": "mg/dL"}}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// Hemoglobin (718-7)
	hbReadings := []struct{ date string; value float64 }{
		{"2025-07-15T09:00:00Z", 11.8},
		{"2025-10-10T09:00:00Z", 12.1},
		{"2026-01-15T09:00:00Z", 12.4},
		{"2026-03-01T09:00:00Z", 12.6},
	}
	for _, r := range hbReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "718-7", "display": "Hemoglobin"}], "text": "Hemoglobin"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.1f, "unit": "g/dL"},
			"referenceRange": [{"low": {"value": 12.0, "unit": "g/dL"}, "high": {"value": 16.0, "unit": "g/dL"}}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// TSH (3016-3)
	tshReadings := []struct{ date string; value float64 }{
		{"2025-10-10T09:00:00Z", 3.2},
		{"2026-01-15T09:00:00Z", 2.8},
		{"2026-03-01T09:00:00Z", 2.5},
	}
	for _, r := range tshReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "3016-3", "display": "TSH"}], "text": "Thyroid Stimulating Hormone"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.1f, "unit": "mIU/L"},
			"referenceRange": [{"low": {"value": 0.4, "unit": "mIU/L"}, "high": {"value": 4.0, "unit": "mIU/L"}}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// ALT (1742-6)
	altReadings := []struct{ date string; value float64 }{
		{"2025-10-10T09:00:00Z", 22},
		{"2026-01-15T09:00:00Z", 25},
		{"2026-03-01T09:00:00Z", 20},
	}
	for _, r := range altReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "1742-6", "display": "ALT"}], "text": "ALT (SGPT)"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.0f, "unit": "U/L"},
			"referenceRange": [{"low": {"value": 7, "unit": "U/L"}, "high": {"value": 56, "unit": "U/L"}}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// Cholesterol (2093-3)
	cholReadings := []struct{ date string; value float64 }{
		{"2025-07-15T09:00:00Z", 210},
		{"2025-10-10T09:00:00Z", 198},
		{"2026-01-15T09:00:00Z", 190},
		{"2026-03-01T09:00:00Z", 185},
	}
	for _, r := range cholReadings {
		id := uuid.New().String()
		insertFHIR(ctx, db, "Observation", id, fmt.Sprintf(`{
			"resourceType": "Observation", "id": "%s", "status": "final",
			"category": [{"coding": [{"system": "http://terminology.hl7.org/CodeSystem/observation-category", "code": "laboratory"}]}],
			"code": {"coding": [{"system": "http://loinc.org", "code": "2093-3", "display": "Cholesterol"}], "text": "Total Cholesterol"},
			"subject": {"reference": "%s"},
			"effectiveDateTime": "%s",
			"valueQuantity": {"value": %.0f, "unit": "mg/dL"},
			"referenceRange": [{"low": {"value": 0, "unit": "mg/dL"}, "high": {"value": 200, "unit": "mg/dL"}}]
		}`, id, patientRef, r.date, r.value), patientRef, drID)
	}

	// AllergyIntolerance for Meera
	allergy1 := uuid.New().String()
	allergy2 := uuid.New().String()
	insertFHIR(ctx, db, "AllergyIntolerance", allergy1, fmt.Sprintf(`{
		"resourceType": "AllergyIntolerance", "id": "%s",
		"clinicalStatus": {"coding": [{"code": "active"}]},
		"verificationStatus": {"coding": [{"code": "confirmed"}]},
		"type": "allergy",
		"category": ["medication"],
		"criticality": "low",
		"code": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "2670", "display": "Codeine"}], "text": "Codeine"},
		"patient": {"reference": "%s"},
		"recordedDate": "2024-08-20",
		"recorder": {"display": "Dr. Priya Sharma"},
		"reaction": [{"manifestation": [{"text": "Nausea, vomiting"}], "severity": "mild"}]
	}`, allergy1, patientRef), patientRef, drID)

	insertFHIR(ctx, db, "AllergyIntolerance", allergy2, fmt.Sprintf(`{
		"resourceType": "AllergyIntolerance", "id": "%s",
		"clinicalStatus": {"coding": [{"code": "active"}]},
		"verificationStatus": {"coding": [{"code": "confirmed"}]},
		"type": "intolerance",
		"category": ["food"],
		"criticality": "low",
		"code": {"text": "Shellfish"},
		"patient": {"reference": "%s"},
		"recordedDate": "2023-01-05",
		"recorder": {"display": "Dr. Priya Sharma"},
		"reaction": [{"manifestation": [{"text": "Urticaria (hives)"}], "severity": "moderate"}]
	}`, allergy2, patientRef), patientRef, drID)

	// Additional MedicationRequest for Meera
	med2 := uuid.New().String()
	insertFHIR(ctx, db, "MedicationRequest", med2, fmt.Sprintf(`{
		"resourceType": "MedicationRequest", "id": "%s", "status": "active", "intent": "order",
		"medicationCodeableConcept": {"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "6809", "display": "Metformin 500mg"}], "text": "Metformin 500mg tablet"},
		"subject": {"reference": "%s"},
		"authoredOn": "2026-01-15",
		"requester": {"display": "Dr. Priya Sharma"},
		"dosageInstruction": [{"text": "Take 1 tablet twice daily with meals", "timing": {"repeat": {"frequency": 2, "period": 1, "periodUnit": "d"}}}]
	}`, med2, patientRef), patientRef, drID)
}

// ═══════════════════════════════════════════════════════════════
// Document Jobs — various statuses for testing
// ═══════════════════════════════════════════════════════════════
func seedDocumentJobs(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding document jobs ──")

	meera, err := lookupUser(ctx, db, "patient.meera@medilink.dev")
	if err != nil || meera.FHIRPatientID == nil {
		fmt.Println("⚠ Meera not found, skipping document jobs")
		return
	}
	rahul, err := lookupUser(ctx, db, "patient.rahul@medilink.dev")
	if err != nil || rahul.FHIRPatientID == nil {
		fmt.Println("⚠ Rahul not found, skipping document jobs")
		return
	}
	drSharma, err := lookupUser(ctx, db, "dr.sharma@medilink.dev")
	if err != nil {
		return
	}
	drPatel, err := lookupUser(ctx, db, "dr.patel@medilink.dev")
	if err != nil {
		return
	}

	var count int
	_ = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM document_jobs")
	if count > 0 {
		fmt.Printf("⏭ %d document jobs already exist, skipping\n", count)
		return
	}

	now := time.Now()
	type docJob struct {
		patientID, patientFHIRID, filename, contentType, status, uploadedBy string
		fileSize                                                            int64
		uploadedAt                                                          time.Time
		completedAt, processingAt                                           *time.Time
		obsCreated, loincMapped                                             *int
		ocrConfidence                                                       *float64
		llmProvider, errorMsg, fhirReportID                                 *string
	}

	intPtr := func(i int) *int { return &i }
	floatPtr := func(f float64) *float64 { return &f }
	timePtr := func(t time.Time) *time.Time { return &t }

	jobs := []docJob{
		{
			patientID: meera.ID, patientFHIRID: *meera.FHIRPatientID,
			filename: "meera_CBC_report_jan2026.pdf", fileSize: 245760, contentType: "application/pdf",
			status: "completed", uploadedBy: drSharma.ID,
			uploadedAt: now.Add(-45 * 24 * time.Hour), completedAt: timePtr(now.Add(-45*24*time.Hour + 3*time.Minute)),
			processingAt: timePtr(now.Add(-45*24*time.Hour + 30*time.Second)),
			obsCreated: intPtr(8), loincMapped: intPtr(6), ocrConfidence: floatPtr(0.94),
			llmProvider: ptr("gemini-1.5-flash"), fhirReportID: ptr(uuid.New().String()),
		},
		{
			patientID: meera.ID, patientFHIRID: *meera.FHIRPatientID,
			filename: "meera_lipid_panel_mar2026.pdf", fileSize: 189440, contentType: "application/pdf",
			status: "completed", uploadedBy: drSharma.ID,
			uploadedAt: now.Add(-5 * 24 * time.Hour), completedAt: timePtr(now.Add(-5*24*time.Hour + 2*time.Minute)),
			processingAt: timePtr(now.Add(-5*24*time.Hour + 20*time.Second)),
			obsCreated: intPtr(5), loincMapped: intPtr(5), ocrConfidence: floatPtr(0.97),
			llmProvider: ptr("gemini-1.5-flash"), fhirReportID: ptr(uuid.New().String()),
		},
		{
			patientID: meera.ID, patientFHIRID: *meera.FHIRPatientID,
			filename: "meera_thyroid_panel_mar2026.jpg", fileSize: 1548800, contentType: "image/jpeg",
			status: "processing", uploadedBy: drSharma.ID,
			uploadedAt: now.Add(-30 * time.Minute),
			processingAt: timePtr(now.Add(-28 * time.Minute)),
		},
		{
			patientID: rahul.ID, patientFHIRID: *rahul.FHIRPatientID,
			filename: "rahul_LFT_report_feb2026.pdf", fileSize: 312320, contentType: "application/pdf",
			status: "completed", uploadedBy: drPatel.ID,
			uploadedAt: now.Add(-20 * 24 * time.Hour), completedAt: timePtr(now.Add(-20*24*time.Hour + 4*time.Minute)),
			processingAt: timePtr(now.Add(-20*24*time.Hour + 25*time.Second)),
			obsCreated: intPtr(6), loincMapped: intPtr(4), ocrConfidence: floatPtr(0.91),
			llmProvider: ptr("gemini-1.5-flash"), fhirReportID: ptr(uuid.New().String()),
		},
		{
			patientID: rahul.ID, patientFHIRID: *rahul.FHIRPatientID,
			filename: "rahul_xray_chest_mar2026.png", fileSize: 2097152, contentType: "image/png",
			status: "failed", uploadedBy: drSharma.ID,
			uploadedAt: now.Add(-3 * 24 * time.Hour),
			processingAt: timePtr(now.Add(-3*24*time.Hour + 15*time.Second)),
			errorMsg: ptr("OCR confidence too low (0.32): unable to extract structured data from X-ray image"),
		},
		{
			patientID: rahul.ID, patientFHIRID: *rahul.FHIRPatientID,
			filename: "rahul_cholesterol_mar2026.pdf", fileSize: 156672, contentType: "application/pdf",
			status: "pending", uploadedBy: drPatel.ID,
			uploadedAt: now.Add(-10 * time.Minute),
		},
	}

	for _, j := range jobs {
		jobID := uuid.New()
		minioKey := fmt.Sprintf("documents/%s/%s/%s", j.patientFHIRID, jobID.String(), j.filename)

		_, err := db.ExecContext(ctx,
			`INSERT INTO document_jobs (
				id, patient_id, patient_fhir_id, original_filename, minio_bucket, minio_key,
				file_size, content_type, status, uploaded_at, processing_started_at, completed_at,
				observations_created, loinc_mapped, ocr_confidence, llm_provider,
				error_message, fhir_report_id, uploaded_by
			) VALUES (
				$1, $2, $3, $4, 'medilink-documents', $5,
				$6, $7, $8, $9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18
			)`,
			jobID, j.patientID, j.patientFHIRID, j.filename, minioKey,
			j.fileSize, j.contentType, j.status, j.uploadedAt, j.processingAt, j.completedAt,
			j.obsCreated, j.loincMapped, j.ocrConfidence, j.llmProvider,
			j.errorMsg, j.fhirReportID, j.uploadedBy,
		)
		if err != nil {
			fmt.Printf("✗ Failed to create document job %s: %v\n", j.filename, err)
		} else {
			fmt.Printf("✓ Document job: %s [%s]\n", j.filename, j.status)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Audit Logs — sample entries for admin audit-logs page
// ═══════════════════════════════════════════════════════════════
func seedAuditLogs(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding audit log entries ──")

	var count int
	_ = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM audit_logs")
	if count >= 10 {
		fmt.Printf("⏭ %d audit logs already exist, skipping\n", count)
		return
	}

	drSharma, _ := lookupUser(ctx, db, "dr.sharma@medilink.dev")
	drPatel, _ := lookupUser(ctx, db, "dr.patel@medilink.dev")
	admin, _ := lookupUser(ctx, db, "admin@medilink.dev")
	meera, _ := lookupUser(ctx, db, "patient.meera@medilink.dev")
	rahul, _ := lookupUser(ctx, db, "patient.rahul@medilink.dev")

	if drSharma == nil || admin == nil {
		fmt.Println("⚠ Required users not found, skipping audit logs")
		return
	}

	now := time.Now()

	meeraRef := ""
	rahulRef := ""
	if meera != nil && meera.FHIRPatientID != nil {
		meeraRef = "Patient/" + *meera.FHIRPatientID
	}
	if rahul != nil && rahul.FHIRPatientID != nil {
		rahulRef = "Patient/" + *rahul.FHIRPatientID
	}

	type auditEntry struct {
		userID, userRole, resType, resID, action, patientRef, purpose, ip string
		success                                                          bool
		statusCode                                                       int
		createdAt                                                        time.Time
	}

	entries := []auditEntry{
		{drSharma.ID, "physician", "Patient", meeraRef, "read", meeraRef, "treatment", "10.90.1.46", true, 200, now.Add(-48 * time.Hour)},
		{drSharma.ID, "physician", "Observation", "", "read", meeraRef, "treatment", "10.90.1.46", true, 200, now.Add(-47 * time.Hour)},
		{drSharma.ID, "physician", "MedicationRequest", "", "create", meeraRef, "treatment", "10.90.1.46", true, 201, now.Add(-46 * time.Hour)},
		{drSharma.ID, "physician", "Patient", rahulRef, "read", rahulRef, "treatment", "10.90.1.46", true, 200, now.Add(-36 * time.Hour)},
		{drSharma.ID, "physician", "DocumentReference", "", "create", meeraRef, "treatment", "10.90.1.46", true, 201, now.Add(-24 * time.Hour)},
		{admin.ID, "admin", "User", "", "read", "", "admin", "10.90.1.100", true, 200, now.Add(-20 * time.Hour)},
		{admin.ID, "admin", "AuditLog", "", "read", "", "admin", "10.90.1.100", true, 200, now.Add(-18 * time.Hour)},
		{admin.ID, "admin", "User", "", "update", "", "admin", "10.90.1.100", true, 200, now.Add(-12 * time.Hour)},
	}

	if drPatel != nil {
		entries = append(entries,
			auditEntry{drPatel.ID, "physician", "Patient", rahulRef, "read", rahulRef, "treatment", "10.90.1.52", true, 200, now.Add(-6 * time.Hour)},
			auditEntry{drPatel.ID, "physician", "Observation", "", "read", rahulRef, "treatment", "10.90.1.52", true, 200, now.Add(-5 * time.Hour)},
		)
	}

	for _, e := range entries {
		_, err := db.ExecContext(ctx,
			`INSERT INTO audit_logs (user_id, user_role, user_email_hash, resource_type, resource_id, action, patient_ref, purpose, success, status_code, ip_address, created_at)
			 VALUES ($1, $2, '', $3, $4, $5, $6, $7, $8, $9, $10::inet, $11)`,
			e.userID, e.userRole, e.resType, e.resID, e.action, e.patientRef, e.purpose, e.success, e.statusCode, e.ip, e.createdAt)
		if err != nil {
			fmt.Printf("✗ Failed to create audit log (%s %s): %v\n", e.action, e.resType, err)
		} else {
			fmt.Printf("✓ Audit log: %s %s %s\n", e.userRole, e.action, e.resType)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Notification Preferences — for all active users
// ═══════════════════════════════════════════════════════════════
func seedNotificationPreferences(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding notification preferences ──")

	emails := []string{
		"admin@medilink.dev",
		"dr.sharma@medilink.dev",
		"dr.patel@medilink.dev",
		"patient.meera@medilink.dev",
		"patient.rahul@medilink.dev",
	}

	for _, email := range emails {
		u, err := lookupUser(ctx, db, email)
		if err != nil {
			continue
		}

		var exists bool
		_ = db.GetContext(ctx, &exists,
			"SELECT EXISTS(SELECT 1 FROM notification_preferences WHERE user_id = $1)", u.ID)
		if exists {
			fmt.Printf("⏭ Notification prefs for %s already exist\n", email)
			continue
		}

		_, err = db.ExecContext(ctx,
			`INSERT INTO notification_preferences (
				user_id,
				email_document_complete, email_document_failed,
				email_consent_granted, email_consent_revoked,
				email_break_glass, email_account_locked,
				push_enabled, push_document_complete,
				push_new_prescription, push_lab_result_ready,
				push_consent_request, push_critical_lab,
				preferred_language
			) VALUES (
				$1,
				true, true, true, true, true, true,
				true, true, true, true, false, true,
				'en'
			)`, u.ID)
		if err != nil {
			fmt.Printf("✗ Failed to create notification prefs for %s: %v\n", email, err)
		} else {
			fmt.Printf("✓ Notification prefs: %s\n", email)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Drug Interactions & Drug Classes — for drug checking feature
// ═══════════════════════════════════════════════════════════════
func seedDrugData(ctx context.Context, db *sqlx.DB) {
	fmt.Println("\n── Seeding drug interaction data ──")

	var count int
	_ = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM drug_interactions")
	if count > 0 {
		fmt.Printf("⏭ %d drug interactions already exist, skipping\n", count)
	} else {
		interactions := []struct {
			drugARx, drugBRx, drugAName, drugBName, severity, desc, mechanism, effect, management string
		}{
			{
				"6809", "11289", "Metformin", "Warfarin", "moderate",
				"Metformin may enhance the anticoagulant effect of Warfarin.",
				"Competition for protein binding sites",
				"Increased risk of bleeding; INR may be elevated",
				"Monitor INR closely when initiating or adjusting Metformin therapy in patients on Warfarin.",
			},
			{
				"17767", "36567", "Amlodipine", "Atorvastatin", "moderate",
				"Amlodipine may increase the serum concentration of Atorvastatin.",
				"Inhibition of CYP3A4 metabolism",
				"Increased risk of myopathy and rhabdomyolysis",
				"Limit Atorvastatin to 20mg daily when used with Amlodipine. Monitor for muscle pain.",
			},
			{
				"1191", "32968", "Aspirin", "Clopidogrel", "major",
				"Concurrent use of Aspirin and Clopidogrel increases bleeding risk significantly.",
				"Dual antiplatelet mechanism — COX-1 inhibition plus P2Y12 blockade",
				"Significantly increased risk of GI and intracranial bleeding",
				"Use combination only when clinically indicated (e.g., post-ACS, post-stent). Add PPI for GI protection.",
			},
			{
				"6809", "4815", "Metformin", "Furosemide", "moderate",
				"Furosemide may increase the risk of lactic acidosis with Metformin.",
				"Furosemide increases Metformin plasma levels by competing for renal tubular secretion",
				"Elevated Metformin levels; increased risk of lactic acidosis",
				"Monitor renal function and clinical status. Consider dose adjustment of Metformin.",
			},
			{
				"10582", "1191", "Levothyroxine", "Aspirin", "minor",
				"Aspirin may slightly increase free T4 levels by displacing Levothyroxine from protein binding.",
				"Displacement from thyroid-binding globulin",
				"Transiently elevated free T4; usually not clinically significant",
				"Generally no dose adjustment needed. Monitor thyroid function if high-dose aspirin is used.",
			},
		}

		for _, i := range interactions {
			a, b := i.drugARx, i.drugBRx
			na, nb := i.drugAName, i.drugBName
			if a > b {
				a, b = b, a
				na, nb = nb, na
			}
			_, err := db.ExecContext(ctx,
				`INSERT INTO drug_interactions (drug_a_rxnorm, drug_b_rxnorm, drug_a_name, drug_b_name, severity, description, mechanism, clinical_effect, management)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 ON CONFLICT (drug_a_rxnorm, drug_b_rxnorm) DO NOTHING`,
				a, b, na, nb, i.severity, i.desc, i.mechanism, i.effect, i.management)
			if err != nil {
				fmt.Printf("✗ Failed to create interaction %s + %s: %v\n", na, nb, err)
			} else {
				fmt.Printf("✓ Drug interaction: %s + %s [%s]\n", na, nb, i.severity)
			}
		}
	}

	// Drug classes
	_ = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM drug_classes")
	if count > 0 {
		fmt.Printf("⏭ %d drug classes already exist, skipping\n", count)
		return
	}

	classes := []struct{ rxnorm, name, class, subclass string }{
		{"6809", "Metformin", "Biguanides", "Antidiabetics"},
		{"17767", "Amlodipine", "Calcium Channel Blockers", "Dihydropyridines"},
		{"36567", "Atorvastatin", "Statins", "HMG-CoA Reductase Inhibitors"},
		{"11289", "Warfarin", "Anticoagulants", "Vitamin K Antagonists"},
		{"1191", "Aspirin", "NSAIDs", "Salicylates"},
		{"32968", "Clopidogrel", "Antiplatelets", "P2Y12 Inhibitors"},
		{"10582", "Levothyroxine", "Thyroid Hormones", "Synthetic T4"},
		{"4815", "Furosemide", "Loop Diuretics", "Sulfonamide Diuretics"},
		{"723", "Amoxicillin", "Penicillins", "Aminopenicillins"},
		{"2670", "Codeine", "Opioid Analgesics", "Phenanthrene Opioids"},
	}

	for _, c := range classes {
		_, err := db.ExecContext(ctx,
			`INSERT INTO drug_classes (rxnorm_code, drug_name, drug_class, drug_subclass)
			 VALUES ($1, $2, $3, $4) ON CONFLICT (rxnorm_code) DO NOTHING`,
			c.rxnorm, c.name, c.class, c.subclass)
		if err != nil {
			fmt.Printf("✗ Failed to create drug class %s: %v\n", c.name, err)
		} else {
			fmt.Printf("✓ Drug class: %s → %s\n", c.name, c.class)
		}
	}
}
