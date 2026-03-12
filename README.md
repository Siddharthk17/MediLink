<div align="center">

# 🏥 MediLink

**A full-stack healthcare platform built for the Indian medical ecosystem.**

FHIR R4 compliant · Consent-driven access · End-to-end encrypted PII

[Quick Start](#quick-start) · [Architecture](#architecture) · [API Reference](docs/API.md) · [Security](docs/SECURITY.md) · [FHIR Compliance](docs/FHIR_COMPLIANCE.md)

</div>

---

## What is MediLink?

MediLink is a production healthcare platform that connects physicians and patients through a standards-compliant electronic health record system. It is designed specifically for the Indian healthcare context.

**For physicians**, MediLink provides a dashboard to manage patients, view medical records, prescribe medications with real-time drug interaction checking, and upload lab reports that are automatically digitized.

**For patients**, MediLink provides a portal to view their own health records, manage consent (who can see their data), and track their medical history.

**For administrators**, MediLink provides tools to manage users, review audit logs, monitor system health, and oversee break-glass emergency access events.

### Key Capabilities

| Capability | What It Does |
|---|---|
| **10 FHIR R4 Resources** | Patient, Practitioner, Organization, Encounter, Condition, MedicationRequest, Observation, DiagnosticReport, AllergyIntolerance, Immunization |
| **Consent Engine** | Patients grant granular, scoped consent to physicians. Every data access is checked against active consent. |
| **Drug Interaction Checker** | Real-time checking via OpenFDA before prescriptions are created. Blocks contraindicated combinations. |
| **Lab Report Pipeline** | Upload PDF/image → OCR (Tesseract) → LLM extraction (Gemini) → LOINC mapping → FHIR Observations |
| **Break-Glass Access** | Emergency override for physicians (rate-limited, fully audited, patient notified by email) |
| **PII Encryption** | AES-256-GCM encryption at rest for names, phone numbers, dates of birth. SHA-256 hashed email for lookups. |
| **Audit Trail** | Immutable, append-only audit log of every data access, login, consent change, and admin action |
| **MFA for Physicians** | TOTP-based two-factor authentication with backup codes. Account locks after 5 failed attempts. |

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend API | Go 1.25 · Gin · sqlx |
| Physician Dashboard | Next.js 15 · React 19 · TypeScript · Tailwind CSS |
| Patient Dashboard | Next.js 15 · React 19 · TypeScript · Tailwind CSS |
| Database | PostgreSQL 15 (JSONB for FHIR resources) |
| Cache & Queue | Redis 7 (consent cache, rate limiting, JWT blacklist, Asynq broker) |
| Search | Elasticsearch 8.11 (full-text search across FHIR resources) |
| Object Storage | MinIO (lab report PDFs and images) |
| OCR | Tesseract (English, Hindi, Marathi) |
| LLM | Google Gemini (structured lab data extraction) |
| Monitoring | Prometheus + Grafana |
| Reverse Proxy | nginx |
| CI/CD | GitHub Actions |

---

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (v2+)
- [Git](https://git-scm.com/)

### 1. Clone and start

```bash
git clone https://github.com/Siddharthk17/MediLink.git
cd MediLink
docker compose up -d
```

This starts **14 services**: the Go API, background worker, both frontend dashboards, PostgreSQL, Redis, Elasticsearch, MinIO, Prometheus, Grafana, Asynqmon, nginx reverse proxy, and initialization containers.

On first run, Docker will build the images. This takes a few minutes. Subsequent starts are fast.

### 2. Verify everything is running

```bash
# Check all services are healthy
docker compose ps

# API health check
curl http://localhost:8580/health

# Full readiness check (verifies DB, Redis, Elasticsearch, MinIO)
curl http://localhost:8580/ready
```

### 3. Open the application

| Service | URL | Notes |
|---|---|---|
| **Physician Dashboard** | http://localhost:8180 | Main physician portal |
| **Patient Dashboard** | http://localhost:8180/patient | Patient portal |
| **Go API** | http://localhost:8580 | Direct API access |
| **Grafana** | http://localhost:8180/grafana | Monitoring (admin / MediLink_grafana) |
| **Asynqmon** | http://localhost:8581 | Background job monitor |
| **MinIO Console** | http://localhost:9051 | Object storage (MediLink / MediLink_minio_dev) |
| **Prometheus** | http://localhost:9190 | Metrics |
| **PostgreSQL** | localhost:5532 | Database (MediLink / MediLink_dev) |
| **Redis** | localhost:6479 | Cache |
| **Elasticsearch** | http://localhost:9280 | Search |

### 4. Seed test data (optional)

```bash
docker compose exec api /bin/sh -c "cd /app && go run cmd/seed/main.go"
```

This creates test accounts:

| Role | Email | Password |
|---|---|---|
| Admin | admin@medilink.dev | Admin@Medi2026! |
| Physician | dr.sharma@medilink.dev | Doctor@Medi2026! |
| Patient | patient.meera@medilink.dev | Patient@Medi2026! |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        nginx (port 8180)                        │
│   /  → Physician Dashboard    /patient/ → Patient Dashboard     │
│   /grafana/ → Grafana                                           │
└──────────────┬────────────────────────┬─────────────────────────┘
               │                        │
    ┌──────────▼──────────┐  ┌──────────▼──────────┐
    │ Physician Dashboard │  │  Patient Dashboard  │
    │   Next.js (3000)    │  │   Next.js (3002)    │
    │   /api/* → Go API   │  │   /api/* → Go API   │
    └──────────┬──────────┘  └──────────┬──────────┘
               │                        │
        ┌──────▼────────────────────────▼──────┐
        │           Go API (port 8080)         │
        │  Auth · FHIR · Consent · Clinical    │
        │  Documents · Admin · Search · Audit  │
        └──┬──────┬───────┬───────┬──────┬─────┘
           │      │       │       │      │
    ┌──────▼─┐ ┌──▼────┐ ┌▼───┐ ┌─▼───┐ ┌▼──────────┐
    │Postgres│ │ Redis │ │ ES │ │MinIO│ │   Worker  │
    │ (5432) │ │(6379) │ │    │ │     │ │  (Asynq)  │
    └────────┘ └───────┘ └────┘ └─────┘ └───────────┘
```

**How requests flow:**

1. Browser hits nginx on port 8180
2. nginx routes to the correct frontend based on URL path
3. Frontend makes API calls to `/api/*`, which Next.js rewrites to the Go API
4. Go API checks JWT authentication → checks consent → processes request
5. Background jobs (OCR, email) are queued in Redis and processed by the Worker

For the full architecture documentation, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Project Structure

```
MediLink/
├── backend/
│   ├── cmd/
│   │   ├── api/                 # API server entrypoint (main.go — route registration, DI)
│   │   ├── worker/              # Background job processor (Asynq)
│   │   └── seed/                # Test data seeder
│   ├── internal/
│   │   ├── admin/               # Admin panel (user management, stats, audit logs)
│   │   ├── anonymization/       # Research data exports (de-identified)
│   │   ├── audit/               # Immutable audit logging with async batching
│   │   ├── auth/                # Authentication (JWT, TOTP, password, middleware)
│   │   ├── clinical/            # Drug interaction checker (OpenFDA + cache)
│   │   ├── config/              # Configuration loading and validation
│   │   ├── consent/             # Consent engine, middleware, cache
│   │   ├── documents/           # OCR → LLM → LOINC → FHIR pipeline
│   │   ├── fhir/                # FHIR resource handlers, services, validators
│   │   ├── middleware/          # CORS, rate limiting, security headers, request logging
│   │   ├── notifications/       # Email (Resend) and push (Firebase) notifications
│   │   └── search/              # Unified cross-resource search (Elasticsearch)
│   ├── pkg/                     # Shared libraries (crypto, database, storage, metrics)
│   ├── migrations/              # PostgreSQL schema migrations (000001–000009)
│   └── tests/                   # Unit and integration tests
├── frontend/
│   ├── shared/                  # Shared TypeScript package (@medilink/shared)
│   │   └── src/
│   │       ├── api/             # API client, auth, consent, FHIR, doctors, clinical
│   │       ├── types/           # TypeScript types (FHIR resources, API responses)
│   │       └── utils/           # Shared utilities (error parsing, FHIR helpers)
│   ├── physician-dashboard/     # Physician portal (Next.js 15)
│   │   └── src/
│   │       ├── app/             # Next.js App Router pages
│   │       ├── components/      # UI components (auth, layout, FHIR, consent)
│   │       ├── hooks/           # Custom hooks (document jobs, keyboard shortcuts)
│   │       ├── store/           # Zustand stores (auth, UI)
│   │       └── lib/             # Utilities (motion, query client, fonts)
│   └── patient-dashboard/       # Patient portal (Next.js 15)
│       └── src/                 # Same structure as physician dashboard
├── infra/
│   ├── nginx/                   # Reverse proxy configuration
│   ├── prometheus/              # Metrics collection config
│   └── grafana/                 # Dashboard and datasource provisioning
├── docs/                        # Documentation
├── docker-compose.yml           # Production Docker Compose (14 services)
├── docker-compose.dev.yml       # Development overrides
├── pnpm-workspace.yaml          # pnpm monorepo workspace
└── .github/workflows/ci.yml     # CI pipeline (build, lint, type check, test)
```

---

## Running Tests

MediLink has **1,058 automated tests** across the backend and both frontends.

### Backend (Go)

```bash
cd backend

# Run all tests
go test ./... -count=1

# Run with verbose output
go test ./... -count=1 -v

# Run a specific package
go test ./tests/unit/auth/... -count=1 -v
```

**478 test functions** covering authentication, consent enforcement, FHIR search parsing, middleware, drug interactions, document processing, admin operations, notifications, search, and cryptography.

### Physician Dashboard

```bash
cd frontend/physician-dashboard

# Run tests
npx vitest run

# Run with coverage
npx vitest run --coverage

# Run in watch mode
npx vitest
```

**412 tests** across 43 test files.

### Patient Dashboard

```bash
cd frontend/patient-dashboard

# Run tests
npx vitest run
```

**168 tests** across 16 test files.

### Full CI Check

The CI pipeline (`.github/workflows/ci.yml`) runs on every push and PR:

```
Backend:    go build → go vet → go test
Physician:  pnpm install → lint → tsc --noEmit → vitest
Patient:    pnpm install → lint → tsc --noEmit → vitest
```

---

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable | Description | Required |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `REDIS_URL` | Redis connection string | Yes |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | Yes |
| `ENCRYPTION_KEY` | AES-256 key (64 hex chars) | Yes |
| `ELASTICSEARCH_URL` | Elasticsearch URL | Yes |
| `MINIO_ENDPOINT` | MinIO endpoint | Yes |
| `MINIO_ACCESS_KEY` | MinIO access key | Yes |
| `MINIO_SECRET_KEY` | MinIO secret key | Yes |
| `GEMINI_API_KEY` | Google Gemini API key (for LLM extraction) | No |
| `OPENFDA_API_KEY` | OpenFDA API key (for drug interactions) | No |
| `RESEND_API_KEY` | Resend API key (for email notifications) | No |

> **Important for production:** Generate secure values for `JWT_SECRET` and `ENCRYPTION_KEY`. Do not use the development defaults.
>
> ```bash
> # Generate a secure JWT secret
> openssl rand -base64 48
>
> # Generate a secure AES-256 encryption key
> openssl rand -hex 32
> ```

---

## Documentation

| Document | Description |
|---|---|
| [Architecture](docs/ARCHITECTURE.md) | System design, service descriptions, data flow, infrastructure |
| [API Reference](docs/API.md) | All endpoints with request/response examples |
| [Security Model](docs/SECURITY.md) | Authentication, encryption, consent, rate limiting, audit |
| [FHIR Compliance](docs/FHIR_COMPLIANCE.md) | Supported resources, search parameters, validation rules |

---

## License

Licensed under the [Apache License 2.0](LICENSE).
