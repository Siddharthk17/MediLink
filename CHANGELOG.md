# Changelog

All notable changes to MediLink are documented here.  
Format: [Keep a Changelog](https://keepachangelog.com/)

---

## [1.2.0] — Week 8: Documentation & Production Readiness

### Added
- Complete `README.md` rewrite (~280 lines — badges, quick start, architecture diagram, all ports)
- Full `docs/API.md` (~520 lines — every endpoint with request/response examples, error codes)
- Full `docs/ARCHITECTURE.md` (~280 lines — ASCII diagrams, 14 services, data flows, security boundaries)
- Full `docs/SECURITY.md` (~300 lines — auth model, PII encryption, consent, rate limiting, audit logging)
- Full `docs/FHIR_COMPLIANCE.md` (~380 lines — all 10 resources, search parameters, validation rules)
- `CHANGELOG.md` covering the full 8-week development history
- Production Dockerfiles for both frontends (multi-stage, non-root, `node:22-alpine`)
- `.dockerignore` for optimized Docker builds
- Frontend containers in `docker-compose.yml` (physician-dashboard, patient-dashboard)
- nginx path-based routing (`/` → physician, `/patient/` → patient, `/grafana/` → Grafana)

### Changed
- `AUDIT_REPORT.md` updated to reflect all resolved issues and current test counts
- `mobile/patient-app/README.md` rewritten to reflect deferred status with future plans
- `next.config.ts` updated with `basePath`, `output: 'standalone'`, and API proxy rewrites
- All 4 documentation stubs replaced with comprehensive content

---

## [0.7.0] — Week 7: Quality, Security & Performance Hardening

### Added
- CI/CD pipeline (`.github/workflows/ci.yml`) — 3 jobs: Go build/vet/test, physician lint/tsc/vitest, patient lint/tsc/vitest
- TOTP rate limiting on verify-totp endpoint (5 attempts / 10 minutes → 30-min lockout)
- JWT secret minimum length validation (32 characters enforced at startup)
- TOTP backup codes migration (`totp_backup_codes` table) with bcrypt-hashed storage
- nginx security headers (X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy, Permissions-Policy)
- nginx `/health` endpoint for Docker healthcheck
- Patient MFA flow — TOTP input component with shake animation and AnimatePresence transitions
- Docker healthchecks for Prometheus, Grafana, and nginx
- `CORS_ALLOWED_ORIGINS` environment variable for origin-based CORS allowlisting

### Fixed
- **CORS wildcard origin** → replaced `Access-Control-Allow-Origin: *` with origin-matching allowlist (C-1)
- **Hardcoded portal URLs** → replaced `:3000`/`:3002` with `NEXT_PUBLIC_*_PORTAL_URL` env vars (C-2)
- **TOTP backup codes not persisted** → added repository methods and migration (C-3)
- **Unsafe consent type assertion** → added `GetConsentByID` to interface, removed `h.engine.(*consentEngine)` cast (H-3)
- **12 `as any` casts in production code** → replaced with proper `CodeableConcept`, `Observation`, `Patient` types (H-2)
- **4 empty catch blocks** → added error logging in LoginForm, DrugCheckPanel, DocumentUpload, useDocumentJobs (H-4)
- **Search page weak typing** → proper `FHIRBundle` typing, removed double `as any` (M-7)
- **Console.error in production** → wrapped all instances in `NODE_ENV === 'development'` guards (H-3)
- **Docker first-run reliability** → increased Elasticsearch retries to 20, API retries to 15, added start_period 30s
- **Missing MedicationRequest statuses** → added `'draft' | 'unknown'` to shared fhir.ts union type

### Security
- Comprehensive codebase audit: 48 findings (4 critical, 12 high, 19 medium, 13 low)
- All 4 critical and all 12 high-priority issues resolved
- Origin-based CORS with `Vary: Origin` and `Access-Control-Allow-Credentials: true`
- GEMINI_MODEL config field properly loaded from environment
- Hardcoded `api.MediLink.health` domain replaced with configurable `FHIR_BASE_URL`
- Empty displayName in user lookup fixed — now queries `full_name` from database

### Tests
- **1,058 total tests — all passing**
- 478 Go tests (0 failures)
- 412 physician dashboard tests (43 files)
- 168 patient dashboard tests (22 files)
- All `tsc --noEmit`, `next lint`, `go vet` clean with zero warnings

---

## [0.6.5] — Week 6B: Patient Dashboard

### Added
- Complete patient dashboard frontend (Next.js 15, React 19, TypeScript)
- 17 routes: login, register, dashboard, health (overview, labs, medications, conditions, allergies, immunizations), consents, documents, timeline, search, notifications, profile, settings
- Patient-specific features: consent management, access audit log, document viewing
- Find a Doctor page — patients can search and grant consent to physicians
- Pending Requests UI — physicians can accept/decline incoming consent requests
- Backend `GET /doctors` endpoint for doctor discovery
- Backend consent pending/accept/decline flow
- Unified login with role-based redirect (physicians → physician dashboard, patients → patient portal)
- 168 patient dashboard tests across 22 test files

### Fixed
- Token refresh race condition — `useAuthStore.getState()` for stable token access instead of stale React closure
- Notification preference toggle — triple bug: partial update wipe, getActorID type mismatch, FCM token NULL scan
- PatientTimeline key prop — fallback to index for entries without resource ID
- Admin hydration error — deferred `isAdmin` behind `mounted` state

---

## [0.6.4] — Week 6A: Bug Fixes & Verification

### Fixed
- 8 backend bugs: search 400 for physicians, audit log inet NULL, OpenFDA drug name parsing, admin reindex routing, nested FHIR→ES conversion
- Auto-refresh after mutations (consent revoke, admin approval, prescription creation, document upload)
- LoginForm navigation race condition — `router.push()` instead of `window.location.href`
- Admin audit logs "Failed to load" — NULL ip_address scan with COALESCE, LEFT JOIN for userEmail

### Tests
- 412 physician dashboard tests passing (43 files, 94.9% component coverage)

---

## [0.6.3] — Week 6A: Physician Dashboard

### Added
- Physician dashboard frontend (Next.js 15, React 19, TypeScript, Tailwind CSS)
- **Meridian Dark** design system with cyan (#06B6D4) accent
- Fonts: Instrument Serif (headings), DM Sans (body), JetBrains Mono (code)
- Dark/light theme toggle with CSS custom properties
- pnpm workspace with `@medilink/shared` package
- Shared library: API clients (auth, FHIR, consent, search, notifications, doctors, documents), TypeScript types, utility functions
- 16 physician routes: login, register, dashboard, patients (list, detail, timeline), prescribe, drug checker, labs, documents, consents, admin (users, audit logs), search, notifications, settings
- TanStack React Query v5 for data fetching with auto-refresh polling
- Zustand v4 for auth and UI state management
- Framer Motion v11 for page transitions and micro-interactions
- Recharts v2 for lab trend charts
- Error boundaries, loading skeletons, empty states on all pages
- Command palette with keyboard shortcuts

### Infrastructure
- Patient portal placeholder (`frontend/patient-dashboard/`)
- Mobile app placeholder (`mobile/patient-app/`)
- 56 initial physician dashboard tests

---

## [0.5.9] — Week 5: Notifications, Admin & Search

### Added
- **Notification system** — preferences management, FCM token registration/revocation, push notification service
- **Anonymization engine** — de-identified FHIR data exports for research (k-anonymity, date shifting, identifier removal)
- **Admin panel** — user management, physician approval/suspension/reinstatement, audit log viewer, system statistics
- **Unified search** — Elasticsearch-backed search across all FHIR resource types with role-scoped results
- **System tasks** — token cleanup, search reindex via Asynq background jobs
- `/notifications/preferences` GET/PUT, `/notifications/fcm-token` register/revoke
- `/admin/users`, `/admin/physicians/{action}`, `/admin/audit-logs`, `/admin/stats`
- `/admin/system/health`, `/admin/search/reindex`, `/admin/tasks/cleanup-tokens`
- `/research/export` create/status/list/delete
- `/search` unified endpoint

### Tests
- 400+ tests (580 including subtests)

---

## [0.4.7] — Week 4: Drug Interaction Checker & Document Pipeline

### Added
- **Drug interaction checker** — OpenFDA API integration with PostgreSQL cache, allergy conflict detection, RxNorm client with Redis 24h caching
- **Document processing pipeline** — upload → MinIO → Asynq queue → Tesseract OCR → Gemini LLM extraction → LOINC mapping → FHIR Observation creation
- Pre-create hook on MedicationRequest — blocks contraindicated drug interactions automatically
- Drug interaction acknowledgment flow with history tracking
- MinIO object storage client with noop fallback
- 60+ LOINC mapping seeds for Indian lab tests
- `/clinical/drug-check`, `/clinical/drug-check/acknowledge`, `/clinical/drug-check/history`
- `/documents/upload`, `/documents/jobs/:jobId`, `/documents/jobs`, `/documents/:id`
- Dockerfile switched from distroless to `debian:bookworm-slim` (Tesseract dependency)

### Infrastructure
- Asynq worker for background document processing
- MinIO init container for automatic bucket creation
- Asynqmon UI for queue monitoring (port 8581)

### Tests
- 485 total tests

---

## [0.3.1] — Week 3: Authentication & Consent Engine

### Added
- **JWT authentication** — HS256 signed tokens, 2-hour access / 7-day refresh, token blacklisting in Redis
- **Refresh token rotation** — old token revoked on use, reuse detection revokes all user tokens
- **TOTP MFA** — setup flow with QR code, 6-digit verification, 10 backup codes (bcrypt-hashed)
- **Consent engine** — patient-to-physician consent grants with lifecycle (pending → active → revoked)
- **Consent middleware** — enforces consent on every FHIR read, Redis-backed cache with immediate invalidation
- **Break-glass emergency access** — rate-limited (3/24h), mandatory audit trail, patient email notification, 24h temporary consent
- **Immutable audit logging** — async batched writes, covers all data access and mutations
- **Rate limiting** — Redis-backed, per-role (patient 100/min, physician 200/min, admin 500/min, auth 10/min)
- **PII encryption** — AES-256-GCM for email, full name, phone, DOB; SHA-256 email hash for lookups
- **Password security** — bcrypt cost 12, complexity rules, common password blocklist
- Role-based access control (patient, physician, admin, researcher)
- `/auth/register/physician`, `/auth/register/patient`, `/auth/login`, `/auth/login/verify-totp`
- `/auth/refresh`, `/auth/logout`, `/auth/me`, `/auth/password/change`
- `/auth/totp/setup`, `/auth/totp/verify-setup`
- `/consent/grant`, `/consent/accept/:id`, `/consent/decline/:id`, `/consent/revoke/:id`
- `/consent/my-grants`, `/consent/my-patients`, `/consent/pending-requests`
- `/consent/break-glass`, `/consent/access-log`
- Security headers middleware (X-Frame-Options, X-Content-Type-Options, CSP, HSTS)
- Request ID middleware for tracing
- Database seeding with test users (admin, physician, patient)

### Tests
- 323 total tests

---

## [0.2.3] — Week 2: FHIR Resources & Search

### Added
- **9 FHIR R4 resource types** — Patient, Practitioner, Organization, Encounter, Condition, MedicationRequest, Observation, DiagnosticReport, AllergyIntolerance
- **Generic FHIR resource service** — CRUD with pre-create/pre-update hooks, soft delete, versioning
- **FHIR resource validation** — per-resource-type validators with required field checks and reference validation
- **FHIR search** — per-resource search parsers with FHIR-standard query parameters and date prefixes
- **FHIR Bundle responses** — searchset and history bundles with pagination links
- **Resource versioning** — version increment on update, history endpoint for all versions
- **Elasticsearch integration** — async FHIR resource indexing, full-text search
- **Patient timeline** — `$timeline` operation returning chronological cross-resource view
- **Lab trends** — `$lab-trends` operation for Observation time series by LOINC code
- Cross-resource reference validation (referenced resources must exist and be active)
- PostgreSQL JSONB storage with denormalized `patient_ref`, `resource_type`, `version` columns

### Infrastructure
- Docker Compose with PostgreSQL, Redis, Elasticsearch
- Database migrations with golang-migrate
- Elasticsearch with health checking and retry logic

---

## [0.1.0] — Week 1: Foundation

### Added
- Go backend with Gin HTTP framework
- PostgreSQL database with UUID primary keys and JSONB columns
- Redis for caching and rate limiting
- Project structure: `cmd/api/`, `cmd/worker/`, `internal/`, `pkg/`, `migrations/`, `tests/`
- Configuration management with Viper (environment variables)
- Graceful shutdown with signal handling
- Health (`/health`) and readiness (`/ready`) endpoints
- Prometheus metrics endpoint (`/metrics`)
- Structured logging
- Docker Compose for local development
- Makefile with build, test, migrate, and seed targets
- Apache 2.0 license

### Infrastructure
- Docker Compose services: api, postgres, redis
- golang-migrate for database schema management
- Multi-stage Docker build

### Tests
- 167 initial tests
