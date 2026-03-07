-- ============================================================
-- REFRESH TOKENS TABLE
-- Stores issued refresh tokens for 7-day session management.
-- When a refresh token is used, it is immediately invalidated
-- and a new one issued (token rotation).
-- ============================================================
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL,  -- SHA-256 of the token
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,  -- issued_at + 7 days
    revoked_at      TIMESTAMPTZ,           -- NULL = still valid
    revoked_reason  VARCHAR(64),           -- 'logout', 'rotation', 'security'
    ip_address      INET,
    user_agent      TEXT,
    CONSTRAINT uq_token_hash UNIQUE (token_hash)
);

CREATE INDEX idx_refresh_tokens_user
    ON refresh_tokens (user_id)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_refresh_tokens_hash
    ON refresh_tokens (token_hash)
    WHERE revoked_at IS NULL;

-- ============================================================
-- EMAIL OTP TABLE
-- One-time passwords for patient registration and login.
-- Each OTP expires in 10 minutes and can be used only once.
-- ============================================================
CREATE TABLE email_otps (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email_hash      VARCHAR(64) NOT NULL,
    otp_hash        VARCHAR(64) NOT NULL,  -- SHA-256 of the 6-digit OTP
    purpose         VARCHAR(32) NOT NULL,  -- 'registration', 'login', 'password_reset'
    expires_at      TIMESTAMPTZ NOT NULL,  -- NOW() + 10 minutes
    used_at         TIMESTAMPTZ,           -- NULL = not yet used
    attempts        INTEGER NOT NULL DEFAULT 0,  -- max 5 attempts
    ip_address      INET,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_otps_hash
    ON email_otps (email_hash, purpose)
    WHERE used_at IS NULL;

-- ============================================================
-- RATE LIMIT TABLE
-- Tracks login attempts for lockout enforcement.
-- Redis handles request-level rate limiting.
-- This table handles account-level lockout (different concern).
-- ============================================================
CREATE TABLE login_attempts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email_hash      VARCHAR(64) NOT NULL,
    ip_address      INET NOT NULL,
    attempted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success         BOOLEAN NOT NULL
);

CREATE INDEX idx_login_attempts_email
    ON login_attempts (email_hash, attempted_at DESC);

CREATE INDEX idx_login_attempts_ip
    ON login_attempts (ip_address, attempted_at DESC);

-- ============================================================
-- SCHEMA FIXES FOR WEEK 3
-- ============================================================

-- totp_secret needs to store AES-256-GCM encrypted bytes
ALTER TABLE users ALTER COLUMN totp_secret TYPE BYTEA USING totp_secret::BYTEA;

-- Add email_enc for encrypted email (display purposes)
-- Plaintext email column will no longer be written to
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_enc BYTEA;

-- Drop the unique constraint on patient_id+provider_id so revoked
-- consents don't block new grants for the same pair.
ALTER TABLE consents DROP CONSTRAINT IF EXISTS uq_patient_provider;

-- Add a partial unique index: only one active consent per patient-provider pair
CREATE UNIQUE INDEX IF NOT EXISTS uq_active_consent_patient_provider
    ON consents (patient_id, provider_id)
    WHERE revoked_at IS NULL;
