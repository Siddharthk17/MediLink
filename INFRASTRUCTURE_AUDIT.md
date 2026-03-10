# MediLink Infrastructure Audit Report

## CRITICAL BUGS FOUND

### 1. **MISSING GEMINI_MODEL CONFIG LOADING** 🔴 CRITICAL
- **Location**: `/home/sid/MediLink/backend/internal/config/config.go`, lines 143-144
- **Issue**: The `GEMINI_MODEL` environment variable is defined in the `GeminiConfig` struct (line 78) and has a default set (line 180), but **it is never loaded from config** in the `Load()` function.
- **Current Code**:
  ```go
  cfg.Gemini.APIKey = v.GetString("GEMINI_API_KEY")  // Line 143
  // Missing: cfg.Gemini.Model = v.GetString("GEMINI_MODEL")
  ```
- **Impact**: `cfg.Gemini.Model` will always be empty string, even though default is set. Code in `/home/sid/MediLink/backend/internal/documents/llm/factory.go:16` references `cfg.Gemini.Model`, which will always be empty.
- **Fix**: Add after line 143:
  ```go
  cfg.Gemini.Model = v.GetString("GEMINI_MODEL")
  ```

---

### 2. **ELASTICSEARCH_URL NOT IN ENV DEFAULTS** 🔴 CRITICAL
- **Location**: `/home/sid/MediLink/backend/cmd/api/main.go`, line 99
- **Issue**: Code uses `os.Getenv("ELASTICSEARCH_URL")` directly without fallback config. If the env var is missing, it defaults to `["http://localhost:9200"]` hardcoded.
- **Current Code**:
  ```go
  esAddresses := strings.Split(os.Getenv("ELASTICSEARCH_URL"), ",")
  if len(esAddresses) == 0 || esAddresses[0] == "" {
      esAddresses = []string{"http://localhost:9200"}
  }
  ```
- **Issue**: In Docker, the compose file sets `ELASTICSEARCH_URL=http://elasticsearch:9200` (line 21, 64), but this is **not in the config.go defaults**, so any local development without explicit env var will fail silently and connect to wrong host.
- **Fix**: Add to `config.go` struct and load:
  ```go
  type AppConfig struct {
      ElasticsearchURL string `mapstructure:"ELASTICSEARCH_URL"`
      // ...
  }
  ```

---

### 3. **SECURITY: HARDCODED DEV CREDENTIALS IN DOCKER-COMPOSE** 🔴 CRITICAL
- **Locations**: 
  - `/home/sid/MediLink/docker-compose.yml`, lines 10, 43, 55 (DATABASE_URL)
  - `/home/sid/MediLink/docker-compose.yml`, lines 12, 57 (JWT_SECRET)
  - `/home/sid/MediLink/docker-compose.yml`, lines 13, 58 (ENCRYPTION_KEY - all zeros!)
  - `/home/sid/MediLink/docker-compose.yml`, lines 19, 62, 124, 150
- **Issues**:
  1. **Plaintext credentials**: Database password `MediLink_dev` is visible in compose file
  2. **Zero-value ENCRYPTION_KEY**: Line 13 and 58: `ENCRYPTION_KEY=0000000000000000000000000000000000000000000000000000000000000000` - This is cryptographically invalid!
  3. **Dev secrets in production**: `JWT_SECRET=dev-secret-change-in-production-minimum-32-chars` is literally the dev secret
- **Fix**: 
  - Use `.env` file (not committed) or secrets management
  - Generate proper random encryption key: `openssl rand -hex 32`
  - Use proper JWT secret in production
- **Severity**: Production data would be at risk

---

### 4. **ENCRYPTION_KEY VALIDATION BUG** 🔴 CRITICAL  
- **Location**: `/home/sid/MediLink/backend/internal/config/config.go`, lines 196-201
- **Issue**: Encryption key validation checks length == 64, but the default in line 180 is `0000000000000000000000000000000000000000000000000000000000000000` (all zeros). This is **not cryptographically secure**.
- **Current Code**:
  ```go
  if len(cfg.Encryption.Key) != 64 {
      return fmt.Errorf("config: ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes), got %d", len(cfg.Encryption.Key))
  }
  ```
- **Issue**: No validation that the key is **different from all zeros** or other weak patterns.
- **Impact**: AES-256-GCM encryption in `/home/sid/MediLink/pkg/crypto/` would use a zero key!
- **Fix**: Add validation:
  ```go
  if cfg.Encryption.Key == "0000000000000000000000000000000000000000000000000000000000000000" {
      return fmt.Errorf("config: ENCRYPTION_KEY must not be all zeros (insecure)")
  }
  ```

---

### 5. **DOCKER HEALTHCHECK RACE CONDITION** 🟠 HIGH
- **Location**: `/home/sid/MediLink/docker-compose.yml`, line 66-71
- **Issue**: Worker depends on API with `service_healthy` condition, but API startup is not guaranteed to complete migrations first. Sequence:
  1. Migrate service runs migrations
  2. API service starts, healthcheck hits `/health` endpoint
  3. If API comes up before migrations complete (possible race condition), healthcheck passes but DB might not have schema

- **Current Dependency Chain**:
  ```yaml
  migrate:
    depends_on:
      postgres: service_healthy
  
  api:
    depends_on:
      postgres: service_healthy
      redis: service_healthy
      elasticsearch: service_healthy
  
  worker:
    depends_on:
      api: service_healthy  # ← RACE CONDITION
  ```
- **Fix**: Add explicit wait:
  ```yaml
  api:
    depends_on:
      migrate:
        condition: service_completed_successfully
  ```

---

### 6. **MISSING PORT IN DOCKER-COMPOSE.DEV.YML** 🟠 HIGH
- **Location**: `/home/sid/MediLink/docker-compose.dev.yml`, lines 8-15
- **Issue**: `api.depends_on` is redefined but doesn't include the original dependencies. The `api` service in dev profile loses elasticsearch and redis dependencies!
- **Current Code**:
  ```yaml
  api:
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      - MINIO_ENDPOINT=
  ```
- **Missing**: elasticsearch dependency is removed when dev profile is used
- **Fix**: Keep elasticsearch in dependencies or remove from dev profile entirely

---

### 7. **NGINX PROXY MISCONFIGURATION** 🟠 HIGH
- **Location**: `/home/sid/MediLink/infra/nginx/nginx.conf`, line 23
- **Issue**: Grafana location strips path but nginx still expects `/grafana/` prefix:
  ```nginx
  location /grafana/ {
      proxy_pass http://grafana:3000/;
      proxy_set_header Host $host;
  }
  ```
- **Problem**: Missing headers for Grafana subpath routing:
  ```nginx
  proxy_set_header X-Script-Name /grafana;
  ```
- **Impact**: Grafana CSS/JS may not load, links may be broken
- **Fix**: Add subpath headers

---

### 8. **NEXT.JS API REWRITE HARDCODED PORT** 🟠 MEDIUM
- **Location**: `/home/sid/MediLink/frontend/physician-dashboard/next.config.ts`, line 8
- **Issue**: API URL fallback is hardcoded to `http://localhost:8580`:
  ```typescript
  destination: `${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8580'}/:path*`
  ```
- **Problem**: 
  1. Docker-compose API is on port 8580 (line 8 of docker-compose.yml)
  2. But if frontend is containerized, it should use `http://api:8080` (internal service name)
  3. Hardcoded localhost won't work in Docker
- **Fix**: Remove hardcoded fallback or set env var in Docker:
  ```dockerfile
  ENV NEXT_PUBLIC_API_URL=http://api:8080
  ```

---

### 9. **MISSING HEALTHCHECK FOR NGINX** 🟠 MEDIUM
- **Location**: `/home/sid/MediLink/docker-compose.yml`, line 185-193
- **Issue**: Nginx service has **no healthcheck** defined, but it depends_on api with `service_healthy`. If nginx dies, nothing will know.
- **Fix**:
  ```yaml
  nginx:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 10s
      timeout: 5s
      retries: 3
  ```

---

### 10. **MINIO HEALTHCHECK CURL ISSUE** 🟠 MEDIUM
- **Location**: `/home/sid/MediLink/docker-compose.yml`, lines 130-134
- **Issue**: Healthcheck uses `curl` but minio image (`minio/minio:latest`) might not have curl installed:
  ```yaml
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
  ```
- **Better approach**: Use `["CMD", "mc", "ready", "local"]` (minio client)
- **Impact**: Healthcheck may fail, causing dependency failures

---

### 11. **MISSING LOINC_MAPPINGS DEFAULT INDEX** 🟠 MEDIUM
- **Location**: `/home/sid/MediLink/backend/migrations/000005_drug_checker.up.sql`, line ~98
- **Issue**: GIN index on `loinc_mappings` (full text search) is created but primary lookup is by `test_name_lower` which is **case-sensitive substring match**.
```sql
CREATE INDEX idx_loinc_test_name ON loinc_mappings (test_name_lower);
CREATE INDEX idx_loinc_test_name_gin ON loinc_mappings USING GIN (to_tsvector('english', test_name));
```
- **Problem**: If code does `WHERE test_name_lower = X`, it won't use GIN index efficiently. The `idx_loinc_test_name` is a B-tree on a VARCHAR column that has duplicates (test names like "Hemoglobin" appear multiple times with different LOINC codes).
- **Impact**: Lab name lookups will be slow for large datasets
- **Fix**: Consider composite index or full-text search primarily

---

### 12. **AUDIT_LOGS IMMUTABILITY NOT ENFORCED ON REPLICAS** 🟡 MEDIUM
- **Location**: `/home/sid/MediLink/backend/migrations/000001_initial_schema.up.sql`, lines 99-100
- **Issue**: Rules to prevent UPDATE/DELETE on audit_logs:
  ```sql
  CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
  CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;
  ```
- **Problem**: These rules don't prevent deletion via `TRUNCATE` or through superuser. Also, in a replicated setup, rules don't replicate reliably to standby replicas.
- **Better fix**: Use column-level security or database-level restrictions
- **Current status**: This is a minor issue since truncate is an admin-only operation, but audit logs should be immutable at all costs

---

### 13. **MISSING DEFAULT VALUES IN REFRESH_TOKENS** 🟡 MEDIUM
- **Location**: `/home/sid/MediLink/backend/migrations/000004_auth_tables.up.sql`, lines 7-18
- **Issue**: `refresh_tokens` table allows NULL for some security fields:
  ```sql
  ip_address      INET,
  user_agent      TEXT,
  ```
- **Problem**: These should have defaults or constraints. If NULL, you lose audit trail of where token was issued.
- **Better**: Add NOT NULL constraints and log defaults if not provided

---

### 14. **SEED FILE MISSING GOB/MARSHALING TYPE DEFINITIONS** 🟡 MEDIUM  
- **Location**: `/home/sid/MediLink/backend/cmd/seed/main.go`, around line 220-230
- **Issue**: Seed creates FHIR Patient resources as inline JSON strings, but uses `fmt.Sprintf` which is error-prone:
  ```go
  json := fmt.Sprintf(`{
      "resourceType": "Patient",
      "name": [{"given": ["%s"]}],
      "gender": "%s",
      "birthDate": "%s"
  }`, u.fullName, strings.ToLower(u.gender), u.dob)
  ```
- **Problem**: If `u.fullName` or `u.dob` contains quotes, this breaks JSON. No escaping!
- **Fix**: Use `json.Marshal()` instead of string formatting

---

### 15. **MISSING ELASTICSEARCH_URL ENV VAR IN .ENV.EXAMPLE** 🟡 MEDIUM
- **Location**: `/home/sid/MediLink/.env.example`, after line 50
- **Issue**: `.env.example` is missing `ELASTICSEARCH_URL` even though the code expects it (line 99 of cmd/api/main.go)
- **Current example missing**:
  ```
  ELASTICSEARCH_URL=http://localhost:9200
  FIREBASE_SERVICE_ACCOUNT_PATH=./firebase-adminsdk.json
  FIREBASE_PROJECT_ID=
  ```
- **Impact**: New developers won't know to set this and will use hardcoded localhost

---

## CONFIGURATION MISMATCHES

### 16. **PORT MAPPING INCONSISTENCY** 🟡 MEDIUM
- **Location**: Multiple files
- **Issue**: API port mapping has multiple different representations:
  - `docker-compose.yml` line 8: `8580:8080` (exposed as 8580, container uses 8080)
  - `next.config.ts` line 8: hardcoded `localhost:8580`
  - `cmd/api/main.go` line 660: uses `cfg.Server.Port` which defaults to `8080`
- **Container vs Host**:
  - Container listens on `8080` (from config default)
  - Host accesses on `8580`
  - But frontend tries to reach `localhost:8580` which works locally but fails in Docker network

---

## SCHEMA/MIGRATION ISSUES

### 17. **MISSING INDEX ON FHIR_RESOURCES.CREATED_AT** 🟡 LOW
- **Location**: `/home/sid/MediLink/backend/migrations/000002_fhir_indexes.up.sql`
- **Issue**: Timeline queries use `ORDER BY created_at DESC` but only `idx_fhir_timeline` includes it:
  ```sql
  CREATE INDEX idx_fhir_timeline
      ON fhir_resources (patient_ref, created_at DESC)
  ```
- **Problem**: Queries on resource_type without patient_ref will do full table scan
- **Example**: "Get all Observations for this provider (all patients)" would scan entire table
- **Fix**: Add:
  ```sql
  CREATE INDEX idx_fhir_resources_type_created
      ON fhir_resources (resource_type, created_at DESC)
      WHERE deleted_at IS NULL;
  ```

---

### 18. **FOREIGN KEY CONSTRAINT MISSING ON USERS.ORGANIZATION_ID** 🟡 MEDIUM
- **Location**: `/home/sid/MediLink/backend/migrations/000001_initial_schema.up.sql`, line 59
- **Issue**: `organization_id` references organizations table but has **no explicit constraint**:
  ```sql
  organization_id UUID,
  ```
- **Should be**:
  ```sql
  organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
  ```
- **Impact**: Orphaned user records if org is deleted

---

## ENVIRONMENT VARIABLE ISSUES

### 19. **MINIO_ENDPOINT EMPTY IN DEV PROFILE** 🟡 MEDIUM
- **Location**: `/home/sid/MediLink/docker-compose.dev.yml`, line 15
- **Issue**: Sets `MINIO_ENDPOINT=` (empty string) which conflicts with default in config.go:
  ```yaml
  - MINIO_ENDPOINT=
  ```
- **Code path**: `/home/sid/MediLink/backend/cmd/api/main.go` lines 57-67
- **Behavior**: When empty, defaults to `minio:9000`, but service is not running in dev profile, so storage fails silently

---

## BUILD & DEPLOYMENT ISSUES

### 20. **DOCKERFILE MISSING DBTLS CERTIFICATE SUPPORT** 🟡 LOW
- **Location**: `/home/sid/MediLink/backend/Dockerfile`, line 31
- **Issue**: Uses Debian slim which has minimal ca-certificates. In production with TLS:
  ```dockerfile
  ca-certificates \
  ```
- **Missing**: If you need to verify database SSL certs, you may need additional packages:
  ```dockerfile
  libpq5 \
  ```

---

### 21. **NO LIVENESS PROBE TIMEOUT** 🟡 LOW
- **Location**: `/home/sid/MediLink/docker-compose.yml`, lines 29-34
- **Issue**: Healthcheck timeout is 3s but API startup + migrations could take longer:
  ```yaml
  healthcheck:
    test: ["CMD-SHELL", "curl -sf http://localhost:8080/health || exit 1"]
    interval: 5s
    timeout: 3s
    retries: 10
    start_period: 10s
  ```
- **Max startup time**: 10s (start_period) + (10 retries * 5s interval) = 60s total before failing
- **Issue**: If migrations take >15s, healthcheck fails before API is ready

---

## NEXT.JS SPECIFIC ISSUES

### 22. **TYPESCRIPT STRICTNULL ERROR SUPPRESSION** 🟡 LOW
- **Location**: `/home/sid/MediLink/frontend/physician-dashboard/tsconfig.json`, line 6
- **Issue**: `"skipLibCheck": true` disables type checking for dependencies
- **Not critical but**: Hides potential type errors in third-party libraries

---

## DATABASE SEED DATA ISSUES

### 23. **SEED DATA MISSING TIMEZONE HANDLING** 🟡 LOW
- **Location**: `/home/sid/MediLink/backend/cmd/seed/main.go`, line 116, 126
- **Issue**: DOB strings are hardcoded without timezone:
  ```go
  dob: "1990-06-15",
  ```
- **Problem**: When encrypted and stored in `dob_enc` (BYTEA), timezone info is lost
- **Better**: Use timestamp with timezone: `1990-06-15T00:00:00+00:00`

---

## BEST PRACTICE VIOLATIONS

### 24. **NO REQUEST ID IN HEADERS** 🟡 LOW
- **Location**: `/home/sid/MediLink/backend/cmd/api/main.go`
- **Issue**: Audit logs track request_id but no middleware generates unique IDs for requests
- **Standard practice**: Add UUID to each request for tracing

---

### 25. **PROMETHEUS SCRAPE CONFIG NOT SECURED** 🟡 LOW  
- **Location**: `/home/sid/MediLink/infra/prometheus/prometheus.yml`, lines 6-9
- **Issue**: Prometheus scrapes `/metrics` endpoint without auth:
  ```yaml
  - job_name: 'MediLink-api'
    static_configs:
      - targets: ['api:8080']
    metrics_path: '/metrics'
  ```
- **Problem**: Anyone in Docker network can access metrics
- **Fix**: Add basic auth or network isolation

---

## SUMMARY TABLE

| # | Severity | Category | Issue | File | Line | Fix Time |
|---|----------|----------|-------|------|------|----------|
| 1 | 🔴 CRITICAL | Config | Missing GEMINI_MODEL load | config.go | 143 | 2 min |
| 2 | 🔴 CRITICAL | Config | Missing ELASTICSEARCH_URL defaults | config.go | N/A | 5 min |
| 3 | 🔴 CRITICAL | Security | Hardcoded dev credentials | docker-compose.yml | 10-150 | 10 min |
| 4 | 🔴 CRITICAL | Security | Zero-value encryption key | config.go | 180 | 5 min |
| 5 | 🟠 HIGH | Docker | Healthcheck race condition | docker-compose.yml | 66 | 3 min |
| 6 | 🟠 HIGH | Docker | Missing dependency in dev profile | docker-compose.dev.yml | 8 | 2 min |
| 7 | 🟠 HIGH | Config | Nginx subpath misconfiguration | nginx.conf | 23 | 5 min |
| 8 | 🟠 MEDIUM | Frontend | Hardcoded API port | next.config.ts | 8 | 3 min |
| 9 | 🟠 MEDIUM | Docker | Missing nginx healthcheck | docker-compose.yml | 185 | 5 min |
| 10 | 🟠 MEDIUM | Docker | Minio healthcheck curl issue | docker-compose.yml | 131 | 2 min |
| 11 | 🟠 MEDIUM | Database | Inefficient LOINC lookup | migrations/000005 | 98 | 10 min |
| 12 | 🟡 MEDIUM | Database | Audit log immutability | migrations/000001 | 99 | 5 min |
| 13 | 🟡 MEDIUM | Database | Missing refresh token defaults | migrations/000004 | 15 | 5 min |
| 14 | 🟡 MEDIUM | Code | Seed JSON injection risk | seed/main.go | 220 | 10 min |
| 15 | 🟡 MEDIUM | Config | Missing ELASTICSEARCH_URL in .env.example | .env.example | EOF | 1 min |
| 16 | 🟡 MEDIUM | Config | Port mapping inconsistency | Multiple | Various | 5 min |
| 17 | 🟡 LOW | Database | Missing index on created_at | migrations/000002 | N/A | 5 min |
| 18 | 🟡 LOW | Database | Missing FK on organization_id | migrations/000001 | 59 | 5 min |
| 19 | 🟡 LOW | Config | MINIO_ENDPOINT empty in dev | docker-compose.dev.yml | 15 | 2 min |
| 20 | 🟡 LOW | Docker | Missing cert support | Dockerfile | 31 | 2 min |
| 21 | 🟡 LOW | Docker | Healthcheck timeout | docker-compose.yml | 29 | 3 min |
| 22 | 🟡 LOW | TypeScript | skipLibCheck enabled | tsconfig.json | 6 | 1 min |
| 23 | 🟡 LOW | Data | Seed timezone handling | seed/main.go | 116 | 5 min |
| 24 | 🟡 LOW | Best Practice | No request ID tracking | cmd/api | N/A | 10 min |
| 25 | 🟡 LOW | Security | Unprotected prometheus | prometheus.yml | 6 | 5 min |

**TOTAL ESTIMATED FIX TIME**: ~2 hours
**CRITICAL ISSUES**: 4
**HIGH PRIORITY**: 3
**MEDIUM**: 9
**LOW**: 9

