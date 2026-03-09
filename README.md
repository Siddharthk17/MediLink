# MediLink

A FHIR R4–compliant medical backend built with Go, designed for the Indian healthcare context.

## Overview

MediLink provides a secure, standards-compliant backend for managing electronic health records. It implements the HL7 FHIR R4 specification with Indian healthcare–specific extensions including LOINC-mapped lab reports, drug interaction checking via OpenFDA, and OCR-based lab report digitization.

## Architecture

- **API Server** — Gin-based REST API serving FHIR R4 endpoints
- **Worker** — Asynq-based background processor for document OCR/LLM pipeline
- **PostgreSQL** — Primary data store with JSONB for FHIR resources
- **Redis** — Caching (consent, RxNorm), rate limiting, JWT blacklist, Asynq broker
- **Elasticsearch** — Full-text search across FHIR resources
- **MinIO** — Object storage for lab report PDFs and images
- **Gemini** — LLM extraction of structured lab data from OCR text

## Features

- **10 FHIR Resources**: Patient, Practitioner, Organization, Encounter, Condition, MedicationRequest, Observation, DiagnosticReport, AllergyIntolerance, Immunization
- **Authentication**: JWT (HS256) with TOTP for physicians, email OTP for patients
- **Consent Engine**: Granular, scope-based consent with Redis caching and break-glass
- **Drug Interaction Checker**: OpenFDA-backed with PostgreSQL cache-aside, allergy cross-reactivity
- **Document Pipeline**: OCR (Tesseract) → LLM extraction → LOINC mapping → FHIR Observations
- **Audit Logging**: Immutable PostgreSQL audit trail with async batching
- **PII Encryption**: AES-256-GCM for names, phones; SHA-256 for email lookups

## Prerequisites

- Go 1.22+
- Docker & Docker Compose
- (Optional) Tesseract OCR for local development

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Siddharthk17/MediLink.git
cd MediLink

# Copy environment file
cp backend/.env.example backend/.env

# Start all services
docker compose up -d

# Run migrations
docker compose run --rm migrate

# Verify
curl http://localhost:8580/health
```

## Running Tests

```bash
cd backend

# Run all tests with race detector
go test -race -count=1 ./...

# Run with coverage
go test -race -coverprofile=coverage.out ./tests/...
go tool cover -func=coverage.out | tail -1
```

## Project Structure

```
MediLink/
├── backend/
│   ├── cmd/api/          # API server entrypoint
│   ├── cmd/worker/       # Asynq worker entrypoint
│   ├── internal/
│   │   ├── audit/        # Immutable audit logging
│   │   ├── auth/         # JWT, TOTP, password, middleware
│   │   ├── clinical/     # Drug interaction checker
│   │   ├── config/       # Viper configuration
│   │   ├── consent/      # Consent engine + middleware
│   │   ├── documents/    # OCR/LLM document pipeline
│   │   ├── fhir/         # FHIR handlers, services, validators
│   │   ├── middleware/   # Rate limiting, CORS, security headers
│   │   └── notifications/# Email via Resend
│   ├── pkg/              # Shared libraries
│   ├── migrations/       # PostgreSQL migrations (000001-000006)
│   └── tests/            # Unit and integration tests
├── docker-compose.yml
├── docs/                 # API, Architecture, FHIR, Security docs
├── infra/                # nginx, Prometheus, Grafana configs
└── scripts/              # Utility scripts
```

## API Endpoints

| Endpoint | Description |
|---|---|
| `POST /auth/register/patient` | Patient registration |
| `POST /auth/register/physician` | Physician registration (requires admin approval) |
| `POST /auth/login` | Login (returns JWT) |
| `/fhir/R4/{Resource}` | FHIR CRUD + search + history (10 resources) |
| `GET /fhir/R4/Patient/:id/$timeline` | Patient timeline |
| `GET /fhir/R4/Observation/$lab-trends` | Lab trend analysis |
| `POST /clinical/drug-check` | Drug interaction check |
| `POST /documents/upload` | Lab report upload (async processing) |
| `GET /health` | Health check |
| `GET /ready` | Readiness check (all dependencies) |

## License

See [LICENSE](LICENSE) for details.
