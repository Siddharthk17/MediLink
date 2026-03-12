# MediLink Security Model

This document describes MediLink's security architecture — how authentication works, how patient data is protected, and what controls are in place to prevent unauthorized access.

---

## Table of Contents

- [Authentication](#authentication)
- [Authorization & Consent](#authorization--consent)
- [Data Encryption](#data-encryption)
- [Rate Limiting](#rate-limiting)
- [Audit Logging](#audit-logging)
- [HTTP Security](#http-security)
- [Infrastructure Security](#infrastructure-security)

---

## Authentication

### JWT Tokens

MediLink uses JSON Web Tokens (JWT) signed with HS256 for authentication.

| Token Type | Lifetime | Purpose |
|---|---|---|
| Access Token | 2 hours | Authenticates API requests |
| Refresh Token | 7 days | Issues new access tokens without re-login |

**Token lifecycle:**

1. User logs in with email + password
2. Server validates credentials and issues access + refresh tokens
3. Access token is sent in every API request (`Authorization: Bearer <token>`)
4. When the access token expires, the client uses the refresh token to get a new pair
5. On logout, the access token JTI is blacklisted in Redis and the refresh token is revoked in PostgreSQL

### Refresh Token Rotation

Every time a refresh token is used, it is revoked and a new one is issued. This is called **rotation**.

**Reuse detection:** If a revoked refresh token is used again (indicating it may have been stolen), the server revokes **all** refresh tokens for that user. This forces a full re-login on every device.

```
Normal flow:
  Token A → use → revoke A, issue Token B → use → revoke B, issue Token C

Theft detection:
  Token A → use → revoke A, issue Token B
  Attacker uses stolen Token A again → REVOKE ALL tokens for this user
```

### Multi-Factor Authentication (MFA)

Physicians and admins can enable TOTP-based two-factor authentication.

**How it works:**

1. Physician calls `POST /auth/totp/setup` → receives QR code and secret
2. Physician scans QR code with an authenticator app (Google Authenticator, Authy, etc.)
3. Physician verifies by entering the 6-digit code → `POST /auth/totp/verify-setup`
4. Server stores encrypted TOTP secret, enables MFA, and returns 10 backup codes

**Login with MFA enabled:**

1. Email + password → server returns a **partial** access token + `requiresTOTP: true`
2. Client sends TOTP code with the partial token → `POST /auth/login/verify-totp`
3. Server validates the code, blacklists the partial token, issues **full** tokens

**TOTP Lockout:** After 5 failed TOTP attempts within 10 minutes, the account is locked for 30 minutes. The lockout is tracked in Redis.

**Backup Codes:** 10 single-use backup codes are generated at TOTP setup. They are bcrypt-hashed before storage. Each can be used once in place of a TOTP code.

### Password Security

| Control | Detail |
|---|---|
| Hashing | bcrypt with cost factor 12 |
| Minimum length | 8 characters |
| Complexity | Must include uppercase, lowercase, digit, and special character |
| Blocklist | Common passwords are rejected |
| Email check | Password cannot contain the email prefix |
| On change | All refresh tokens are revoked (forces re-login everywhere) |

---

## Authorization & Consent

### Role-Based Access Control

MediLink has four roles:

| Role | What They Can Do |
|---|---|
| **patient** | View own health records, manage consent grants, view access log |
| **physician** | View consented patient records, create/update FHIR resources, prescribe, upload labs |
| **admin** | All of the above, plus user management, audit logs, physician approval, system admin |
| **researcher** | Request de-identified data exports |

Roles are checked in middleware before the request reaches the handler. Some endpoints are restricted to specific roles (e.g., only physicians can create prescriptions, only admins can approve physicians).

### Consent-Based Access Control

**This is the core security mechanism.** Every time a physician reads patient data, the consent middleware checks whether the patient has granted that physician access.

**How consent works:**

1. Patient grants consent to a specific physician → consent starts as **pending**
2. Physician accepts the consent → status changes to **active**
3. Consent can optionally have:
   - A **scope** (which resource types the physician can access)
   - An **expiration date**
4. When the physician makes a FHIR read request, the consent middleware:
   - Checks Redis cache first (fast path)
   - On cache miss, queries PostgreSQL
   - Verifies the consent is active, not expired, and covers the requested resource type
   - Caches the result in Redis
5. Patient can revoke consent at any time → cache is immediately invalidated

**Consent enforcement matrix:**

| Actor | Search Endpoints | Read by ID |
|---|---|---|
| Patient | Auto-scoped to own data | Can only read own resources (verified by patient_ref) |
| Physician | Must provide `patient` parameter, consent checked | Consent checked against resource's patient_ref |
| Admin | Bypass (access logged) | Bypass (access logged) |

**What is NOT consent-gated:**
- Practitioner and Organization resources (public reference data)
- Write operations (POST, PUT, DELETE) — these are role-gated instead

### Break-Glass Emergency Access

For emergency situations where a physician needs immediate access without waiting for consent.

**Controls:**
- Reason must be at least 20 characters (prevents casual use)
- Maximum 3 break-glass accesses per physician per 24 hours (Redis counter)
- Creates a temporary consent that expires in 24 hours
- Patient is notified by email immediately
- Fully logged in the audit trail
- Admin can review all break-glass events via `/admin/audit-logs/break-glass`

---

## Data Encryption

### PII Encryption at Rest

Patient personally identifiable information is encrypted before storage using AES-256-GCM.

| Field | Encryption | Lookup |
|---|---|---|
| Email | AES-256-GCM encrypted | SHA-256 hash stored separately for lookups |
| Full Name | AES-256-GCM encrypted | — |
| Phone Number | AES-256-GCM encrypted | — |
| Date of Birth | AES-256-GCM encrypted | — |
| TOTP Secret | AES-256-GCM encrypted | — |

**Why two representations for email?**

The encrypted email cannot be searched (encryption is non-deterministic — same plaintext produces different ciphertext each time). The SHA-256 hash is deterministic and allows lookup by email without exposing the plaintext in the database.

**Key management:**

The encryption key is a 256-bit (32-byte) key provided via the `ENCRYPTION_KEY` environment variable (64 hex characters). In production, this should come from a secrets manager, not from a file.

### Data at Rest

- PostgreSQL data is stored in Docker volumes
- MinIO objects (uploaded documents) are stored in Docker volumes
- In production, use encrypted volumes and enable PostgreSQL's `ssl` mode

### Data in Transit

- Internal service communication is unencrypted (within Docker network)
- In production, enable TLS at the nginx layer for external traffic

---

## Rate Limiting

MediLink uses Redis-backed rate limiting with different limits per context.

### Login Rate Limits

| Limit | Threshold | Window |
|---|---|---|
| Per email | 5 failed attempts | 15 minutes |
| Per IP | 10 failed attempts | 15 minutes |
| TOTP | 5 failed attempts | 10 minutes (then 30-min lockout) |

### API Rate Limits

| Role | Limit | Window |
|---|---|---|
| Patient | 100 requests | 1 minute |
| Physician | 200 requests | 1 minute |
| Admin | 500 requests | 1 minute |
| Auth endpoints | 10 requests | 1 minute |

### Break-Glass Rate Limit

Maximum 3 emergency access requests per physician per 24 hours.

### Fail-Open Policy

If Redis is unavailable, the rate limiter **allows** the request through. This is intentional for a healthcare application — blocking a physician from accessing patient data because of a Redis outage could have worse consequences than allowing unthrottled access temporarily.

---

## Audit Logging

Every significant action in MediLink is recorded in an append-only audit log.

### What is Logged

| Event | Details Captured |
|---|---|
| Login (success/failure) | User ID, email hash, IP, user agent |
| Logout | User ID, JTI |
| Password change | User ID |
| FHIR resource read | Actor, resource type, resource ID, patient ref |
| FHIR resource create/update/delete | Actor, resource type, resource ID, patient ref |
| Consent grant/accept/decline/revoke | Actor, consent ID, patient ID, provider ID |
| Break-glass access | Physician ID, patient ref, reason |
| Admin actions | Admin ID, target user, action |
| Document upload/processing | Actor, job ID, patient ref |

### Audit Entry Structure

```json
{
  "id": "uuid",
  "userId": "actor-uuid",
  "userRole": "physician",
  "userEmailHash": "sha256-hash",
  "resourceType": "Observation",
  "resourceId": "resource-uuid",
  "patientRef": "Patient/patient-fhir-id",
  "action": "read",
  "purpose": "treatment",
  "success": true,
  "statusCode": 200,
  "ipAddress": "192.168.1.1",
  "userAgent": "Mozilla/5.0...",
  "createdAt": "2026-03-12T06:00:00Z"
}
```

### Implementation

Audit entries are batched and written asynchronously to avoid impacting request latency. The `audit_logs` table has no UPDATE or DELETE operations — entries are immutable once written.

---

## HTTP Security

### Security Headers

Every response includes:

| Header | Value | Purpose |
|---|---|---|
| `X-Frame-Options` | `SAMEORIGIN` | Prevents clickjacking |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME sniffing |
| `X-XSS-Protection` | `1; mode=block` | XSS filter |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer leakage |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Disables sensitive browser APIs |
| `X-Request-ID` | UUID | Request tracing |

### CORS

The API uses origin-based CORS allowlisting. Only the configured frontend origins are allowed. Credentials (cookies) are supported.

### Input Validation

- All FHIR resources are validated before storage (required fields, valid types, reference integrity)
- All SQL queries use parameterized placeholders (no string concatenation)
- Request body size limits are enforced
- File uploads are validated by content type

### XSS Prevention

- No `dangerouslySetInnerHTML` in any production frontend code
- All user input is rendered through React's built-in XSS protection
- API responses use proper Content-Type headers

---

## Infrastructure Security

### Docker Network Isolation

All services communicate over a Docker bridge network. Only the following ports are exposed to the host:

| Port | Service | Sensitivity |
|---|---|---|
| 8180 | nginx | Public entry point |
| 8580 | Go API | Direct API access |
| 8581 | Asynqmon | Admin tool |
| 5532 | PostgreSQL | Database |
| 6479 | Redis | Cache |
| 9280 | Elasticsearch | Search |
| 9050, 9051 | MinIO | Object storage |
| 9190 | Prometheus | Metrics |
| 3150 | Grafana | Monitoring |

**Production recommendation:** Restrict all ports except 8180 (nginx) to `127.0.0.1` or remove them entirely.

### Secrets Management

In the development `docker-compose.yml`, secrets are inline for convenience. In production:

1. Use Docker secrets or a secrets manager (Vault, AWS SSM, etc.)
2. Never commit real secrets to version control
3. Generate unique values for `JWT_SECRET` and `ENCRYPTION_KEY`
4. Rotate secrets periodically

### Container Security

- Backend containers run on minimal base images (`debian:bookworm-slim` for API/Worker, `node:22-alpine` for frontends)
- Frontend containers run as non-root user (`nextjs`, UID 1001)
- No unnecessary packages installed
- Multi-stage builds minimize image size
