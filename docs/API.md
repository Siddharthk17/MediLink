# MediLink API Reference

Base URL: `http://localhost:8580` (direct) or `http://localhost:8180/api` (via nginx)

All API responses follow the [FHIR R4 OperationOutcome](https://hl7.org/fhir/R4/operationoutcome.html) format for errors:

```json
{
  "resourceType": "OperationOutcome",
  "issue": [{
    "severity": "error",
    "code": "invalid",
    "diagnostics": "Human-readable error description"
  }]
}
```

---

## Table of Contents

- [Authentication](#authentication)
- [FHIR Resources](#fhir-resources)
- [Consent Management](#consent-management)
- [Clinical](#clinical)
- [Documents](#documents)
- [Search](#search)
- [Admin](#admin)
- [Notifications](#notifications)
- [Research](#research)
- [System](#system)

---

## Authentication

All protected endpoints require the `Authorization: Bearer <token>` header.

### Register Physician

```
POST /auth/register/physician
```

Creates a physician account in **pending** status. Requires admin approval before login.

**Request:**
```json
{
  "email": "dr.sharma@example.com",
  "password": "SecurePass123!",
  "fullName": "Dr. Anil Sharma",
  "phone": "+91-9876543210",
  "mciNumber": "MCI-12345",
  "specialization": "Cardiology",
  "organizationId": "uuid-optional"
}
```

**Response (201):**
```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Physician registration submitted. Your account is pending admin approval."
}
```

**Password requirements:** minimum 8 characters, must include uppercase, lowercase, digit, and special character. Cannot contain the email prefix. Common passwords are blocked.

### Register Patient

```
POST /auth/register/patient
```

Creates a patient account in **active** status. Also creates a FHIR Patient resource.

**Request:**
```json
{
  "email": "meera.patel@example.com",
  "password": "SecurePass123!",
  "fullName": "Meera Patel",
  "dateOfBirth": "1990-06-15",
  "gender": "female",
  "phone": "+91-9876543210",
  "preferredLanguage": "en"
}
```

**Response (201):**
```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440001",
  "fhirPatientId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "active",
  "message": "Patient registration successful."
}
```

### Login

```
POST /auth/login
```

**Request:**
```json
{
  "email": "dr.sharma@example.com",
  "password": "SecurePass123!"
}
```

**Response — no MFA (200):**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 7200,
  "role": "physician",
  "requiresMFASetup": true
}
```

**Response — MFA required (200):**
```json
{
  "accessToken": "eyJ...(partial token)...",
  "expiresIn": 7200,
  "role": "physician",
  "requiresTOTP": true
}
```

When `requiresTOTP` is true, the `accessToken` is a partial token. Use it to call the TOTP verification endpoint.

**Rate limits:** 5 failed attempts per email per 15 minutes. 10 failed attempts per IP per 15 minutes.

### Verify TOTP

```
POST /auth/login/verify-totp
Authorization: Bearer <partial-token>
```

**Request:**
```json
{
  "code": "123456"
}
```

**Response (200):**
```json
{
  "accessToken": "eyJ...(full token)...",
  "refreshToken": "eyJ...",
  "expiresIn": 7200,
  "role": "physician"
}
```

**Lockout:** 5 failed TOTP attempts triggers a 30-minute lockout.

### Refresh Token

```
POST /auth/refresh
```

Rotates the refresh token. The old token is revoked. If a revoked token is reused, **all** tokens for that user are revoked (suspected theft).

**Request:**
```json
{
  "refreshToken": "eyJ..."
}
```

**Response (200):**
```json
{
  "accessToken": "eyJ...(new)...",
  "refreshToken": "eyJ...(new)...",
  "expiresIn": 7200,
  "role": "physician"
}
```

### Logout

```
POST /auth/logout
Authorization: Bearer <token>
```

**Request (optional):**
```json
{
  "refreshToken": "eyJ..."
}
```

**Response:** `204 No Content`

### Get Current User

```
GET /auth/me
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "dr.sharma@example.com",
  "fullName": "Dr. Anil Sharma",
  "phone": "+91-9876543210",
  "role": "physician",
  "status": "active",
  "totpEnabled": true,
  "specialization": "Cardiology",
  "mciNumber": "MCI-12345",
  "createdAt": "2026-01-15T10:30:00Z",
  "lastLoginAt": "2026-03-12T06:00:00Z"
}
```

### Change Password

```
POST /auth/password/change
Authorization: Bearer <token>
```

**Request:**
```json
{
  "oldPassword": "CurrentPass123!",
  "newPassword": "NewSecurePass456!"
}
```

**Response (200):**
```json
{
  "message": "Password changed successfully"
}
```

All refresh tokens are revoked after a password change.

### Setup TOTP

```
POST /auth/totp/setup
Authorization: Bearer <token>
```

Available to physicians and admins only.

**Response (200):**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qrCode": "data:image/png;base64,..."
}
```

### Verify TOTP Setup

```
POST /auth/totp/verify-setup
Authorization: Bearer <token>
```

**Request:**
```json
{
  "code": "123456"
}
```

**Response (200):**
```json
{
  "backupCodes": ["abc123", "def456", "ghi789", "..."],
  "message": "TOTP has been enabled. Save your backup codes securely — they will not be shown again."
}
```

---

## FHIR Resources

All FHIR endpoints are under `/fhir/R4/{ResourceType}` and require authentication.

### Supported Resources

| Resource | Create | Read | Update | Delete | Search | History |
|---|---|---|---|---|---|---|
| Patient | Physician | All | Physician | Admin | All | All |
| Practitioner | Admin | All | Admin | Admin | All | All |
| Organization | Admin | All | Admin | Admin | All | All |
| Encounter | Physician | All | Physician | Admin | All | All |
| Condition | Physician | All | Physician | Admin | All | All |
| MedicationRequest | Physician | All | Physician | Admin | All | All |
| Observation | Physician | All | Physician | Admin | All | All |
| DiagnosticReport | Physician | All | Physician | Admin | All | All |
| AllergyIntolerance | Physician | All | Physician | Admin | All | All |
| Immunization | Physician | All | Physician | Admin | All | All |

### Create Resource

```
POST /fhir/R4/{ResourceType}
Authorization: Bearer <token>
Content-Type: application/json
```

**Request body:** A valid FHIR R4 resource JSON. The `id` field is auto-generated if not provided.

**Response (201):** The created resource with server-assigned `id` and `meta.versionId`.

### Read Resource

```
GET /fhir/R4/{ResourceType}/{id}
Authorization: Bearer <token>
```

**Consent enforcement:** Physicians can only read resources for patients who have granted them consent. Patients can only read their own resources.

**Response (200):** The FHIR resource JSON.

### Update Resource

```
PUT /fhir/R4/{ResourceType}/{id}
Authorization: Bearer <token>
Content-Type: application/json
```

**Response (200):** The updated resource with incremented `meta.versionId`.

### Delete Resource

```
DELETE /fhir/R4/{ResourceType}/{id}
Authorization: Bearer <token>
```

Soft delete. The resource is marked as deleted but retained for audit purposes.

**Response:** `204 No Content`

### Search Resources

```
GET /fhir/R4/{ResourceType}?param=value
Authorization: Bearer <token>
```

Returns a FHIR Bundle of matching resources.

**Common parameters (all resources):**

| Parameter | Type | Description |
|---|---|---|
| `_count` | integer | Results per page (default: 20, max: 100) |
| `_offset` | integer | Pagination offset |
| `patient` | reference | Filter by patient (e.g., `Patient/uuid`) |
| `status` | string | Filter by status |

**Resource-specific search parameters:**

| Resource | Parameters |
|---|---|
| Patient | `name`, `gender`, `birthdate` |
| Practitioner | `name`, `specialty` |
| Organization | `name`, `type` |
| Encounter | `patient`, `status`, `class`, `date` (prefix: `ge`, `le`, `gt`, `lt`) |
| Condition | `patient`, `clinical-status`, `category`, `code` |
| MedicationRequest | `patient`, `status`, `intent`, `code` |
| Observation | `patient`, `code`, `category`, `date` (prefix support), `status` |
| DiagnosticReport | `patient`, `code`, `category`, `status`, `date` (prefix support) |
| AllergyIntolerance | `patient`, `clinical-status`, `criticality` |
| Immunization | `patient`, `status`, `vaccine-code`, `date` (prefix support) |

**Response (200):**
```json
{
  "resourceType": "Bundle",
  "type": "searchset",
  "total": 42,
  "entry": [
    {
      "resource": { "resourceType": "Observation", "id": "...", "..." : "..." }
    }
  ]
}
```

### Resource History

```
GET /fhir/R4/{ResourceType}/{id}/_history
GET /fhir/R4/{ResourceType}/{id}/_history/{versionId}
```

Returns the version history of a resource, or a specific version.

### Special Operations

**Patient Timeline:**
```
GET /fhir/R4/Patient/{id}/$timeline?_count=50&_offset=0
```
Returns a chronological bundle of all resources for a patient (Encounters, Conditions, Observations, etc.).

**Lab Trends:**
```
GET /fhir/R4/Observation/$lab-trends?patient=Patient/{id}&code={loinc-code}
```
Returns historical lab values for trend analysis.

---

## Consent Management

All consent endpoints are under `/consent` and require authentication.

### Grant Consent

```
POST /consent/grant
Authorization: Bearer <token>
Role: patient or admin
```

**Request:**
```json
{
  "providerId": "physician-user-uuid",
  "scope": ["Observation", "MedicationRequest", "Condition"],
  "expiresAt": "2027-01-01T00:00:00Z",
  "purpose": "treatment",
  "notes": "Cardiology follow-up"
}
```

If `scope` is omitted, defaults to `["*"]` (all resource types). Creates a **pending** consent that the physician must accept.

**Response (201):** The created consent record.

### Accept Consent

```
PUT /consent/{consentId}/accept
Authorization: Bearer <token>
Role: physician or admin
```

Only the physician named in the consent can accept it.

**Response (200):**
```json
{
  "message": "Consent accepted"
}
```

### Decline Consent

```
PUT /consent/{consentId}/decline
Authorization: Bearer <token>
Role: physician or admin
```

**Request:**
```json
{
  "reason": "Patient not under my care"
}
```

**Response (200):**
```json
{
  "message": "Consent declined"
}
```

### Revoke Consent

```
DELETE /consent/{consentId}/revoke
Authorization: Bearer <token>
Role: patient, physician, or admin
```

Idempotent — revoking an already-revoked consent returns success.

**Response (200):**
```json
{
  "message": "Consent revoked"
}
```

### Get My Grants (Patient)

```
GET /consent/my-grants
Authorization: Bearer <token>
Role: patient or admin
```

Returns all consent records for the authenticated patient.

### Get My Patients (Physician)

```
GET /consent/my-patients
Authorization: Bearer <token>
Role: physician or admin
```

Returns all patients who have granted active consent to the authenticated physician.

### Get Pending Requests (Physician)

```
GET /consent/pending-requests
Authorization: Bearer <token>
Role: physician or admin
```

Returns pending consent requests awaiting the physician's acceptance.

### Break-Glass Emergency Access

```
POST /consent/break-glass
Authorization: Bearer <token>
Role: physician or admin
```

Creates temporary 24-hour full-access consent for emergency situations.

**Request:**
```json
{
  "patientId": "fhir-patient-uuid",
  "reason": "Patient unconscious in emergency room, need immediate access to medical history and allergies for treatment",
  "clinicalContext": "Emergency department"
}
```

**Constraints:**
- Reason must be at least 20 characters
- Maximum 3 break-glass accesses per physician per 24 hours
- Patient is notified by email
- Fully audited

**Response (201):** The created emergency consent record.

### Get Access Log (Patient)

```
GET /consent/access-log
Authorization: Bearer <token>
Role: patient or admin
```

Returns the audit trail of who accessed the patient's data.

---

## Clinical

### Check Drug Interactions

```
POST /clinical/drug-check
Authorization: Bearer <token>
Role: physician
```

Checks a new medication against the patient's current prescriptions and allergies.

**Request:**
```json
{
  "patientId": "fhir-patient-uuid",
  "rxnormCode": "161",
  "medicationName": "Aspirin"
}
```

**Response (200):**
```json
{
  "newMedication": { "rxnormCode": "161", "name": "Aspirin" },
  "interactions": [
    {
      "drugA": { "rxnormCode": "161", "name": "Aspirin" },
      "drugB": { "rxnormCode": "11289", "name": "Warfarin" },
      "severity": "major",
      "description": "Increased risk of bleeding",
      "mechanism": "Additive antiplatelet effects",
      "management": "Monitor INR closely",
      "source": "openfda",
      "cached": false
    }
  ],
  "allergyConflicts": [],
  "highestSeverity": "major",
  "hasContraindication": false,
  "checkComplete": true
}
```

**Severity levels:** `contraindicated` > `major` > `moderate` > `minor` > `unknown` > `none`

If the check finds a `contraindicated` interaction, MedicationRequest creation is blocked until the physician acknowledges the risk.

### Acknowledge Interaction

```
POST /clinical/drug-check/acknowledge
Authorization: Bearer <token>
Role: physician
```

Acknowledges a contraindicated interaction to allow prescription creation.

### Get Check History

```
GET /clinical/drug-check/history/{patientId}
Authorization: Bearer <token>
Role: physician
```

Returns previous drug interaction check results for a patient.

---

## Documents

### Upload Document

```
POST /documents/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Form fields:**
- `file` — The document file (PDF, PNG, JPG, JPEG)
- `patientId` — FHIR Patient ID

Uploads a lab report for asynchronous processing. The document goes through:
1. OCR text extraction (Tesseract)
2. LLM-based structured data extraction (Gemini)
3. LOINC code mapping
4. FHIR Observation resource creation

**Response (202):**
```json
{
  "jobId": "job-uuid",
  "status": "pending",
  "estimatedProcessingTime": "30-60 seconds"
}
```

### Get Job Status

```
GET /documents/jobs/{jobId}
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "jobId": "job-uuid",
  "status": "completed",
  "observationsCreated": 12,
  "loincMapped": 10,
  "ocrConfidence": 0.87,
  "llmProvider": "gemini",
  "fhirReportId": "report-uuid",
  "uploadedAt": "2026-03-12T06:00:00Z",
  "completedAt": "2026-03-12T06:00:45Z"
}
```

**Job statuses:** `pending` → `processing` → `completed` | `failed` | `needs-manual-review`

### List Jobs

```
GET /documents/jobs?patientId={fhirPatientId}
Authorization: Bearer <token>
```

### Delete Job

```
DELETE /documents/jobs/{jobId}
Authorization: Bearer <token>
```

---

## Search

### Unified Search

```
GET /search?q={query}&patient={patientId}
Authorization: Bearer <token>
```

Full-text search across all FHIR resources using Elasticsearch. Physicians must provide a `patient` parameter. Patients are automatically scoped to their own data.

**Response (200):** FHIR Bundle of matching resources.

---

## Admin

All admin endpoints require the `admin` role.

### User Management

```
GET    /admin/users                          # List all users
GET    /admin/users/{userId}                 # Get user details
PUT    /admin/users/{userId}/role            # Update user role
POST   /admin/physicians/{userId}/approve    # Approve pending physician
POST   /admin/physicians/{userId}/suspend    # Suspend a physician
POST   /admin/physicians/{userId}/reinstate  # Reinstate suspended physician
POST   /admin/researchers/invite             # Invite a researcher
GET    /doctors                              # List active physicians (all roles)
```

### Audit Logs

```
GET /admin/audit-logs                        # All audit logs
GET /admin/audit-logs/patient/{id}           # Logs for a specific patient
GET /admin/audit-logs/actor/{id}             # Logs for a specific actor
GET /admin/audit-logs/break-glass            # Break-glass events only
```

### System

```
GET  /admin/stats                            # Dashboard statistics
GET  /admin/system/health                    # Detailed system health
POST /admin/search/reindex                   # Trigger Elasticsearch reindex
POST /admin/tasks/cleanup-tokens             # Trigger expired token cleanup
```

---

## Notifications

```
GET  /notifications/preferences              # Get notification preferences
PUT  /notifications/preferences              # Update preferences
POST /notifications/fcm-token                # Register push notification token
DELETE /notifications/fcm-token              # Revoke push notification token
```

---

## Research

```
POST   /research/export                      # Request de-identified data export
GET    /research/export/{exportId}            # Get export status
GET    /research/exports                      # List all exports
DELETE /research/export/{exportId}            # Delete export (admin only)
```

---

## System

### Health Check

```
GET /health
```

Returns `200 OK` if the API server is running.

### Readiness Check

```
GET /ready
```

Returns `200 OK` only if all dependencies (PostgreSQL, Redis, Elasticsearch, MinIO) are reachable. Returns `503 Service Unavailable` if any dependency is down.

### Metrics

```
GET /metrics
```

Prometheus-compatible metrics endpoint.

---

## Error Codes

| HTTP Status | Meaning |
|---|---|
| 400 | Invalid request (missing fields, validation failure) |
| 401 | Invalid credentials or expired token |
| 403 | Forbidden (wrong role, no consent, account suspended) |
| 404 | Resource not found |
| 409 | Conflict (duplicate email, TOTP already enabled) |
| 429 | Rate limit exceeded (includes `Retry-After` header) |
| 500 | Internal server error |
