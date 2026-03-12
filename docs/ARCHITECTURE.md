# MediLink Architecture

This document describes the system architecture of MediLink — how the services are organized, how data flows through the system, and how the pieces connect.

---

## System Overview

MediLink is a full-stack healthcare platform composed of **14 Docker services** that work together:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         nginx (port 8180)                               │
│                                                                         │
│   /              → Physician Dashboard (Next.js, port 3000)             │
│   /patient/      → Patient Dashboard (Next.js, port 3002)               │
│   /grafana/      → Grafana (port 3000 internal)                         │
└───────┬──────────────────────┬──────────────────────┬───────────────────┘
        │                      │                      │
        ▼                      ▼                      ▼
┌───────────────┐   ┌──────────────────┐   ┌─────────────────┐
│   Physician   │   │     Patient      │   │    Grafana      │
│   Dashboard   │   │    Dashboard     │   │  (monitoring)   │
│   Next.js 15  │   │    Next.js 15    │   │                 │
│               │   │                  │   │                 │
│  /api/* proxy │   │   /api/* proxy   │   │                 │
│      │        │   │       │          │   │                 │
└──────┼────────┘   └───────┼──────────┘   └────────┬────────┘
       │                    │                       │
       └────────────┬───────┘                       │
                    ▼                               │
        ┌───────────────────────┐         ┌────────▼────────┐
        │     Go API Server     │         │   Prometheus    │
        │     (port 8080)       │◄────────│   (port 9090)   │
        │                       │ /metrics│                 │
        │  ┌─────────────────┐  │         └─────────────────┘
        │  │   Middleware    │  │
        │  │  Auth → Consent │  │
        │  │  → Rate Limit   │  │
        │  └────────┬────────┘  │
        │           │           │
        │  ┌────────▼────────┐  │
        │  │   Handlers      │  │
        │  │  Auth · FHIR    │  │
        │  │  Consent · Docs │  │
        │  │  Clinical       │  │
        │  │  Admin · Search │  │
        │  └────────┬────────┘  │
        │           │           │
        └───────────┼───────────┘
                    │
    ┌───────┬───────┼───────┬──────────┐
    ▼       ▼       ▼       ▼          ▼
┌────────┐┌─────┐┌──────┐┌─────┐┌───────────┐
│Postgres││Redis││ ES   ││MinIO││  Worker   │
│ (5432) ││(6379)│(9200)││(9000)│  (Asynq)  │
│        ││     ││      ││     ││           │
│ Users  ││Cache││Search││Files││ OCR → LLM │
│ FHIR   ││Queue││Index ││Store││ → LOINC   │
│ Audit  ││Rate ││      ││     ││ → FHIR    │
│ Consent││Limit││      ││     ││           │
└────────┘└─────┘└──────┘└─────┘└───────────┘
```

---

## Services

### Go API Server

The core backend service. Handles all business logic, authentication, and data access.

| Aspect | Detail |
|---|---|
| Language | Go 1.25 |
| Framework | Gin |
| Database | sqlx (PostgreSQL) |
| Port | 8080 (internal), 8580 (exposed) |
| Configuration | Environment variables |

**Request lifecycle:**

1. Request arrives → middleware pipeline runs in order:
   - CORS validation
   - Security headers (X-Frame-Options, CSP, etc.)
   - Request ID assignment
   - Request logging
   - Rate limiting (Redis-backed, per-role limits)
2. For protected routes:
   - JWT validation → extract user ID, role, org
   - MFA enforcement (reject partial tokens on protected endpoints)
3. For FHIR routes:
   - Consent middleware checks access permission
   - Patients are scoped to their own data
   - Physicians must have active consent from the patient
   - Admins bypass consent (access is still logged)
4. Handler processes request → service layer → repository → database
5. Audit logger records the access asynchronously

### Background Worker

Processes long-running tasks asynchronously using [Asynq](https://github.com/hibiken/asynq) (Redis-backed task queue).

**Tasks processed:**

| Task | What It Does |
|---|---|
| `document:process` | Downloads uploaded file from MinIO → OCR with Tesseract → LLM extraction with Gemini → LOINC code mapping → creates FHIR Observation and DiagnosticReport resources |

### Physician Dashboard

The web portal used by doctors to manage patients.

| Aspect | Detail |
|---|---|
| Framework | Next.js 15, React 19, TypeScript |
| Styling | Tailwind CSS with CSS custom properties |
| State | Zustand (auth + UI stores) |
| Data Fetching | TanStack Query (React Query) |
| API Communication | Axios via `@medilink/shared` package |
| Port | 3000 |

**Key pages:** Dashboard, Patient list, Patient detail (timeline, vitals, conditions, medications, labs, allergies, immunizations), Document upload, Drug interaction checker, Consent management, TOTP setup, Settings.

### Patient Dashboard

The web portal used by patients to view their health records and manage consent.

| Aspect | Detail |
|---|---|
| Framework | Next.js 15, React 19, TypeScript |
| Port | 3002 |
| Base Path | `/patient` |

**Key pages:** Dashboard (health summary), Health records, Medications, Lab results, Allergies, Immunizations, Consent management, Access log, Settings.

### nginx

Reverse proxy and single entry point for the application.

- Routes `/` to the Physician Dashboard
- Routes `/patient/` to the Patient Dashboard
- Routes `/grafana/` to Grafana
- Adds security headers on all responses
- Port 8180

### PostgreSQL

Primary data store for everything.

**Key tables:**

| Table | Purpose |
|---|---|
| `users` | User accounts (PII encrypted at rest) |
| `fhir_resources` | FHIR resource data (JSONB), versioned with history |
| `consents` | Consent grants between patients and physicians |
| `audit_logs` | Immutable audit trail |
| `login_attempts` | Login attempt tracking for rate limiting |
| `refresh_tokens` | JWT refresh token store (hashed) |
| `totp_backup_codes` | MFA backup codes (bcrypt hashed) |
| `drug_interaction_cache` | Cached OpenFDA drug interaction results |
| `loinc_codes` | LOINC code mapping table |
| `document_jobs` | Document processing job status |
| `research_exports` | De-identified data export records |
| `notification_preferences` | Per-user notification settings |

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and run automatically on startup.

### Redis

Serves multiple purposes:

| Usage | Details |
|---|---|
| Consent cache | Caches consent check results to avoid DB queries on every FHIR read |
| Rate limiting | Sliding window counters for login, API, and TOTP attempts |
| JWT blacklist | Stores revoked JWT IDs (JTI) with TTL |
| TOTP lockout | Tracks failed TOTP attempts and lockout state |
| Break-glass counter | Rate limits emergency access per physician |
| RxNorm cache | Caches drug name → RxNorm code lookups |
| Asynq broker | Task queue for background jobs |

### Elasticsearch

Full-text search across FHIR resources. The search service indexes resources when they are created or updated and provides the `/search` endpoint for cross-resource queries.

### MinIO

S3-compatible object storage for uploaded lab report files (PDF, PNG, JPG). Files are stored in the `medilink-lab-reports` bucket with private access.

### Prometheus + Grafana

Prometheus scrapes the `/metrics` endpoint on the Go API server. Grafana provides dashboards for monitoring API performance, error rates, and system health.

---

## Data Flow Examples

### Patient Login

```
Browser → nginx → Patient Dashboard (Next.js)
  → POST /api/auth/login → Go API
    → Hash email (SHA-256) → lookup in DB
    → Verify password (bcrypt)
    → Check TOTP if enabled
    → Generate JWT access + refresh tokens
    → Store refresh token hash in DB
    → Return tokens
  → Set cookie (medilink_patient_token) in browser
  → Redirect to /dashboard
```

### Physician Views Patient Data

```
Browser → nginx → Physician Dashboard (Next.js)
  → GET /api/fhir/R4/Observation?patient=Patient/uuid → Go API
    → JWT middleware: validate token, extract physician ID
    → Consent middleware: check physician has consent for this patient
      → Check Redis cache first → if miss, query DB
    → Search handler: query PostgreSQL for matching Observations
    → Audit logger: record the access asynchronously
    → Return FHIR Bundle
  → Display in dashboard
```

### Lab Report Upload

```
Browser → Physician Dashboard
  → POST /api/documents/upload (multipart) → Go API
    → Save file to MinIO
    → Create job record in PostgreSQL (status: pending)
    → Enqueue Asynq task → Redis
    → Return job ID (202 Accepted)

Worker picks up task from Redis:
  → Download file from MinIO
  → OCR with Tesseract (PDF → text, or image → text)
  → Send to Gemini LLM for structured data extraction
  → Map extracted values to LOINC codes
  → Create FHIR Observation resources in PostgreSQL
  → Create FHIR DiagnosticReport in PostgreSQL
  → Update job status to "completed"

Dashboard polls:
  → GET /api/documents/jobs/{jobId} (every 3 seconds)
  → Display results when completed
```

### Break-Glass Emergency Access

```
Physician Dashboard:
  → POST /api/consent/break-glass → Go API
    → Verify reason is ≥ 20 characters
    → Check Redis rate limit (max 3 per 24 hours per physician)
    → Find patient user from FHIR ID
    → Create temporary consent (24h, scope: *)
    → Write audit log entry
    → Send email notification to patient (async)
    → Return consent record
```

---

## Security Boundaries

```
┌─────────────────────────────────────────────────┐
│                  PUBLIC ZONE                    │
│  Login, Register, Health check                  │
└─────────────────────┬───────────────────────────┘
                      │ JWT Required
┌─────────────────────▼───────────────────────────┐
│              AUTHENTICATED ZONE                 │
│  Profile, Logout, Password change, TOTP setup   │
│  Notifications, Document upload                 │
└─────────────────────┬───────────────────────────┘
                      │ Consent Check
┌─────────────────────▼───────────────────────────┐
│            CONSENT-GATED ZONE                   │
│  All FHIR resource reads (10 resource types)    │
│  Patients: own data only                        │
│  Physicians: consented patients only            │
│  Admins: all data (logged)                      │
└─────────────────────┬───────────────────────────┘
                      │ Admin Role
┌─────────────────────▼───────────────────────────┐
│               ADMIN ZONE                        │
│  User management, audit logs, system health     │
│  Physician approval/suspension, reindexing      │
└─────────────────────────────────────────────────┘
```

---

## Deployment

### Docker Compose (Production)

```bash
docker compose up -d
```

All 14 services start with proper dependency ordering:

1. PostgreSQL and Redis start first (health checks)
2. Elasticsearch and MinIO start (health checks)
3. Migrations run against PostgreSQL
4. MinIO bucket initialization
5. Go API starts (depends on all of the above)
6. Worker starts (depends on API health)
7. Both frontend dashboards build and start
8. nginx starts last (depends on all frontends + API)
9. Monitoring (Prometheus, Grafana, Asynqmon) starts independently

### Environment Variables

See the [README](../README.md#environment-variables) for the full list of configuration variables.

### Important for Production

1. **Generate secure secrets** — do not use the development defaults for `JWT_SECRET` and `ENCRYPTION_KEY`
2. **Network isolation** — restrict database, Redis, and Elasticsearch ports to internal networks
3. **TLS termination** — add HTTPS at the nginx or load balancer level
4. **Backup strategy** — configure PostgreSQL backups and MinIO replication
5. **Log aggregation** — configure Docker log drivers for centralized logging
6. **Resource limits** — set memory and CPU limits on Docker containers
