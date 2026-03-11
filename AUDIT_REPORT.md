# MediLink — Comprehensive Codebase Audit Report

**Audit Date:** January 2025  
**Scope:** Full stack — Go backend, two Next.js frontends, shared package, infrastructure  
**Mode:** READ ONLY — no code changes made

---

## Executive Summary

| Category | Critical | High | Medium | Low | Total |
|----------|----------|------|--------|-----|-------|
| Backend (Go) | 1 | 3 | 4 | 3 | 11 |
| Physician Dashboard | 0 | 2 | 4 | 3 | 9 |
| Patient Dashboard | 1 | 2 | 4 | 2 | 9 |
| Shared Package | 0 | 2 | 3 | 2 | 7 |
| Infrastructure | 2 | 3 | 4 | 3 | 12 |
| **TOTAL** | **4** | **12** | **19** | **13** | **48** |

### Build and Test Status
| Check | Result |
|-------|--------|
| go build ./... | Clean |
| go vet ./... | 4 test compilation errors |
| go test ./... | 6 failures (4 build, 2 assertion) |
| Physician tsc --noEmit | Clean |
| Physician next build | 18 routes, all compile |
| Physician next lint | 1 warning (missing useCallback dep) |
| Physician vitest | 412 tests passed (43 files) |
| Patient tsc --noEmit | Clean |
| Patient next build | 18 routes, all compile |
| Patient next lint | 0 warnings |
| Patient vitest | No test files exist |

---

## CRITICAL ISSUES (4)

### C-1. React Hooks Violation — Patient Health Page
**File:** frontend/patient-dashboard/src/app/(dashboard)/health/page.tsx:32-39  
**Category:** Bug  
**Description:** useQuery is called inside .map(), which violates the React Rules of Hooks. The code suppresses the ESLint warning with eslint-disable-next-line react-hooks/rules-of-hooks. This will cause unpredictable behavior if the array length changes between renders — state corruption, crashes, or stale data.  
**Impact:** Runtime crashes or state corruption when sections array changes.  
**Fix:** Extract each query into its own named hook call, or use a single query that fetches all resource types.

---

### C-2. Default Encryption Key is All Zeros
**File:** docker-compose.yml:13, .env:22  
**Category:** Security  
**Description:** The AES-256-GCM encryption key used for PII (patient names, phone numbers) is set to 64 hex zeros. This provides zero encryption — any attacker with DB access can decrypt all patient data trivially.  
**Impact:** Complete PII exposure if database is compromised.  
**Fix:** Generate a random 32-byte key: openssl rand -hex 32. Store in secure secret manager, never in version control.

---

### C-3. Hardcoded JWT Secret in Version Control
**File:** docker-compose.yml:12, .env:17  
**Category:** Security  
**Description:** JWT signing secret is a plaintext string committed to the repository: dev-secret-change-in-production-minimum-32-chars. Anyone with repo access can forge JWT tokens and impersonate any user.  
**Impact:** Complete authentication bypass.  
**Fix:** Use environment variable injection at deploy time. Remove from docker-compose.yml.

---

### C-4. CORS Test Expectation Mismatch
**File:** tests/unit/middleware/middleware_test.go:60  
**Category:** Test / Security  
**Description:** The CORS middleware was upgraded from wildcard * to origin-based allowlisting (only localhost:3000 and localhost:3001), which is the secure approach. However, the test still expects Access-Control-Allow-Origin: *. The test sends no Origin header, so the middleware correctly returns no ACAO header, but the test asserts *.  
**Impact:** Test failure masks the fact that CORS implementation is actually correct and secure.  
**Fix:** Update test to send Origin: http://localhost:3000 and assert the echoed origin, not *.

---

## HIGH ISSUES (12)

### H-1. Four Go Test Files Won't Compile — Stale Mock Interfaces
**Files:**
- tests/unit/admin/admin_test.go:231 — mockService missing ListDoctors method
- tests/unit/consent/consent_test.go:237 — mockConsentRepo missing GetPendingRequests method
- tests/unit/documents/document_test.go:40 — mockJobRepo missing ListByUploader method
- tests/unit/notifications/preferences_test.go:123 — FCMToken should be *string not string

**Category:** Test  
**Description:** When new interface methods were added to production code, mock structs in tests were not updated. These tests cannot compile, meaning their coverage is zero.  
**Fix:** Add missing methods to each mock struct.

---

### H-2. Search Test Expects 400 But Code Returns 500
**File:** tests/unit/search/search_test.go:211  
**Category:** Bug / Test  
**Description:** TestUnifiedSearch_PhysicianRequiresPatient expects 400 with "patient parameter required" when physician searches without patient param. But the search handler was refactored to auto-scope to all consented patients (DB query). Test sqlmock has no expectations → DB call fails → 500 "search failed".  
**Fix:** Update test to set DB expectations, OR add graceful fallback in SearchService.

---

### H-3. Error Status Codes Determined By String Matching
**File:** internal/consent/handlers.go:48-53  
**Category:** Bug  
**Description:** HTTP status codes determined by strings.Contains(err.Error(), "cannot grant"). Brittle — if error message changes, wrong status code returned.  
**Fix:** Use sentinel errors and errors.Is().

---

### H-4. Empty catch {} Blocks Swallow Errors Silently
**Files:** physician-dashboard LoginForm.tsx:113, DrugCheckPanel.tsx:108, DocumentUpload.tsx:65, useDocumentJobs.ts:26  
**Category:** Error Handling  
**Description:** Four catch blocks catch errors without logging or handling them. User sees no feedback on failure.  
**Fix:** Add toast.error() or console.error() in each catch block.

---

### H-5. as any Type Casts Undermine Type Safety (14 total)
**Files:** TimelineEvent.tsx (6), PatientTimeline.tsx (2), AllergyList.tsx (2), labs/page.tsx (1), search/page.tsx (2), profile/page.tsx (1)  
**Category:** Type Safety  
**Description:** 14 as any casts across both dashboards, mostly on FHIR resource properties.  
**Fix:** Properly type FHIR resources in shared package or use type guards.

---

### H-6. SELECT * Used in 11 Repository Queries
**Files:** auth/repository.go, notifications/preferences_repo.go, consent/repository.go, documents/repository.go  
**Category:** Performance / Maintainability  
**Description:** SELECT * returns all columns. Schema changes could cause scan errors or fetch unnecessary data.  
**Fix:** Explicitly list columns in SELECT statements.

---

### H-7. Patient Dashboard Has Zero Tests
**File:** frontend/patient-dashboard/  
**Category:** Test  
**Description:** 14 pages, 10 components, 2 stores, 5 lib files — all untested.  
**Fix:** Add tests for auth store, middleware, queryClient, and critical UI components.

---

### H-8. Documentation Stubs — Critical Docs Are Empty
**Files:** docs/API.md, docs/SECURITY.md, docs/ARCHITECTURE.md, docs/FHIR_COMPLIANCE.md  
**Category:** Documentation  
**Description:** All four files contain only "Coming in Week 8" placeholder text.  
**Fix:** Complete before production deployment.

---

### H-9. Elasticsearch Accessible Without Authentication
**File:** docker-compose.yml:106-120  
**Category:** Security  
**Description:** Elasticsearch runs with xpack.security.enabled=false. Contains indexed FHIR patient data.  
**Fix:** Enable xpack security or network-isolate.

---

### H-10. MinIO Console Exposed on Port 9051
**File:** docker-compose.yml:129-131  
**Category:** Security  
**Description:** MinIO web console exposed with default dev credentials. Contains uploaded medical documents.  
**Fix:** Restrict to 127.0.0.1:9051:9051 or remove port exposure.

---

### H-11. Missing Response Type Annotations on Shared API Methods
**File:** frontend/shared/src/api/consent.ts  
**Category:** Type Safety  
**Description:** Mutation methods (grantConsent, revokeConsent, acceptConsent, declineConsent, breakGlass) lack TypeScript response type annotations.  
**Fix:** Add generic type params.

---

### H-12. Documents API Missing from Shared Package
**File:** frontend/shared/src/index.ts  
**Category:** Missing API  
**Description:** Backend exposes /documents/* endpoints but shared package has no documentsAPI export.  
**Fix:** Create shared/src/api/documents.ts.

---

## MEDIUM ISSUES (19)

| ID | File | Category | Description |
|----|------|----------|-------------|
| M-1 | middleware.go:67-70 | Config | CORS default origins missing port 3002 (patient dashboard) |
| M-2 | DocumentUpload.tsx:70 | Bug | useCallback missing queryClient dependency |
| M-3 | ratelimit.go:133-134 | Security | Rate limiter fails open when Redis is unavailable |
| M-4 | consent/middleware.go:17-20 | Security | Consent middleware only checks GET; POST/PUT bypass consent |
| M-5 | Both next.config.ts | Config | API proxy defaults to localhost — breaks in production |
| M-6 | shared/api/client.ts:61 | Bug | Token refresh URL construction fragile with trailing slashes |
| M-7 | shared/api/fhir.ts:19,22,25 | Type Safety | No validation for dynamic FHIR resource type strings |
| M-8 | shared/api/doctors.ts:3-8 | Code Org | DoctorSummary type defined in API file instead of types file |
| M-9 | docker-compose.yml:76-90 | Config | No DB backup/restore strategy documented |
| M-10 | docker-compose.yml | Config | No resource limits on Docker containers |
| M-11 | docker-compose.yml | Config | No Docker log driver configuration |
| M-12 | Both TopBar components | Accessibility | Missing aria-current="page" on active nav links |
| M-13 | Both queryClient.ts | Error Handling | Global mutation error handler only does console.error() |
| M-14 | docker-compose.yml:110 | Config | Elasticsearch heap 512MB — too small for production |
| M-15 | Multiple pages | React | Array index used as React key on skeleton loaders |
| M-16 | docker-compose.yml:164-171 | Security | Asynqmon UI exposed on port 8581 without authentication |
| M-17 | docker-compose.yml | Config | No version pinning for minio, grafana, prometheus images |
| M-18 | Patient login/register | Type Safety | Manual type assertions instead of shared parseAPIError() |
| M-19 | integration/setup_test.go:21 | Test | TODO: integration tests don't manage DB schema lifecycle |

---

## LOW ISSUES (13)

| ID | File | Description |
|----|------|-------------|
| L-1 | middleware.go:139 | GetRequestID type assertion without ok-check |
| L-2 | admin/repository.go:212 | fmt.Sprintf for column list — safe but unusual |
| L-3 | Physician TOTPInput.tsx | key={i} for fixed-length OTP inputs — acceptable |
| L-4 | Patient uiStore.ts | Command palette toggle exists but no component |
| L-5 | Patient Interactions.tsx | Hardcoded colors instead of CSS variables |
| L-6 | Shared fhir.ts | FHIRBundle entry type is loose union |
| L-7 | Shared severity.ts | Severity ordering uses array position |
| L-8 | Both dashboards | localStorage for theme storage — non-sensitive |
| L-9 | Physician tests | innerHTML in test files only — no production XSS risk |
| L-10 | Shared api.ts | parseAPIError uses loose typing |
| L-11 | Shared types | No pagination wrapper type |
| L-12 | Both dashboards | Only 11 aria-* attributes total — minimal accessibility |
| L-13 | Backend | Only 1 TODO in entire codebase — clean |

---

## Strengths and Good Patterns

### Security (Well Done)
- Password hashing: bcrypt cost factor 12
- Password validation: 8+ chars, upper/lower/digit/special, common password blocklist
- Email encryption: AES-256-GCM for PII at rest
- CORS: Origin-based allowlisting, not wildcard
- Security headers: X-Frame-Options, CSP, HSTS, X-Content-Type-Options
- Rate limiting: Per-role with Redis sorted sets (patient 100/min, auth 5/min, TOTP 5/10min, break-glass 3/24h)
- No SQL injection: All queries use parameterized placeholders
- No XSS: Zero dangerouslySetInnerHTML in production code
- JWT in httpOnly cookies
- TOTP/MFA for physicians with backup codes
- Audit logging on all data access
- Token blacklisting on logout and password change
- .env not in git

### Architecture (Well Done)
- Consent middleware on every FHIR read
- Redis-backed consent cache with proper invalidation
- Break-glass emergency access with mandatory audit
- FHIR R4: 8 resource types with validation hooks
- Drug interaction pre-create hook blocks contraindicated prescriptions
- Reference validation on all FHIR create/update
- Graceful degradation: Noop clients for missing services
- Graceful shutdown with signal handling
- Health/readiness endpoints

### Frontend (Well Done)
- 412 physician tests passing (43 files)
- Both dashboards: TypeScript strict, zero type errors
- Both dashboards: production build clean
- Auto-refresh polling on all queries
- Error boundaries in both dashboards
- Loading/error/empty states on all data pages
- Hydration safety with mounted state pattern
- Event listener cleanup in all useEffects

---

## Recommended Fix Priority

### Immediate (Before Any Deployment)
1. C-2 — Replace all-zeros encryption key
2. C-3 — Move JWT secret out of version control
3. C-1 — Fix React hooks violation in patient health page
4. H-9 — Enable Elasticsearch authentication
5. H-10 — Restrict MinIO console port

### This Sprint
6. C-4, H-1, H-2 — Fix all 7 broken/failing tests
7. H-3 — Replace string-matching error handling with sentinel errors
8. H-4 — Add error handling to empty catch blocks
9. M-1 — Add port 3002 to CORS allowed origins
10. M-4 — Extend consent middleware to POST/PUT operations

### Next Sprint
11. H-5 — Eliminate as any casts with proper FHIR types
12. H-7 — Write patient dashboard tests
13. H-6 — Replace SELECT * with explicit column lists
14. H-8 — Complete documentation stubs
15. H-11, H-12 — Type shared API methods, add documents API

---

*End of Audit Report. No code was modified during this audit.*
