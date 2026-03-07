-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- FHIR RESOURCES TABLE
-- Core table. Stores ALL FHIR resource types as JSONB.
-- One table for all resource types — this is how real FHIR
-- servers work. Resource type is a column, not a table name.
-- ============================================================
CREATE TABLE fhir_resources (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type   VARCHAR(64) NOT NULL,
    resource_id     VARCHAR(64) NOT NULL,
    version_id      INTEGER NOT NULL DEFAULT 1,
    data            JSONB NOT NULL,
    patient_ref     VARCHAR(128),
    status          VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    created_by      UUID,
    updated_by      UUID,

    CONSTRAINT uq_resource_version UNIQUE (resource_id, version_id)
);

-- Resource history table — immutable record of every version
-- NEVER delete from this table. EVER.
CREATE TABLE fhir_resource_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type   VARCHAR(64) NOT NULL,
    resource_id     VARCHAR(64) NOT NULL,
    version_id      INTEGER NOT NULL,
    data            JSONB NOT NULL,
    operation       VARCHAR(16) NOT NULL,
    operated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operated_by     UUID
);

-- ============================================================
-- USERS TABLE
-- ============================================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           VARCHAR(320) NOT NULL,
    email_hash      VARCHAR(64) NOT NULL,
    password_hash   VARCHAR(255),
    role            VARCHAR(32) NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',

    full_name_enc   BYTEA,
    phone_enc       BYTEA,
    dob_enc         BYTEA,
    address_enc     BYTEA,

    mci_number      VARCHAR(32),
    specialization  VARCHAR(128),
    organization_id UUID,

    totp_secret     VARCHAR(64),
    totp_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    totp_verified   BOOLEAN NOT NULL DEFAULT FALSE,

    fhir_patient_id VARCHAR(64),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT uq_email_hash UNIQUE (email_hash)
);

-- ============================================================
-- AUDIT LOGS TABLE
-- IMMUTABLE. Every access to every FHIR resource is logged here.
-- ============================================================
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID,
    user_role       VARCHAR(32),
    user_email_hash VARCHAR(64),
    resource_type   VARCHAR(64),
    resource_id     VARCHAR(64),
    action          VARCHAR(32) NOT NULL,
    patient_ref     VARCHAR(128),
    purpose         VARCHAR(32),
    success         BOOLEAN NOT NULL,
    status_code     INTEGER,
    error_message   TEXT,
    ip_address      INET,
    user_agent      TEXT,
    request_id      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- IMMUTABILITY RULE — enforced at database layer
CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

-- ============================================================
-- CONSENTS TABLE
-- ============================================================
CREATE TABLE consents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id      UUID NOT NULL REFERENCES users(id),
    provider_id     UUID NOT NULL REFERENCES users(id),
    scope           JSONB NOT NULL DEFAULT '["*"]',
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    revoked_by      UUID,
    purpose         VARCHAR(32) NOT NULL DEFAULT 'treatment',
    notes           TEXT,
    created_by      UUID NOT NULL,

    CONSTRAINT uq_patient_provider UNIQUE (patient_id, provider_id)
);

-- ============================================================
-- ORGANIZATIONS TABLE
-- ============================================================
CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fhir_org_id     VARCHAR(64) UNIQUE,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(64),
    registration_no VARCHAR(64),
    address_enc     BYTEA,
    phone_enc       BYTEA,
    status          VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
