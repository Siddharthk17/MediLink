-- Revert partial unique index and restore original constraint
DROP INDEX IF EXISTS uq_active_consent_patient_provider;
ALTER TABLE consents ADD CONSTRAINT uq_patient_provider UNIQUE (patient_id, provider_id);

-- Remove email_enc column
ALTER TABLE users DROP COLUMN IF EXISTS email_enc;

-- Revert totp_secret back to VARCHAR
ALTER TABLE users ALTER COLUMN totp_secret TYPE VARCHAR(64) USING totp_secret::VARCHAR(64);

-- Drop auth tables
DROP INDEX IF EXISTS idx_login_attempts_ip;
DROP INDEX IF EXISTS idx_login_attempts_email;
DROP TABLE IF EXISTS login_attempts;
DROP INDEX IF EXISTS idx_email_otps_hash;
DROP TABLE IF EXISTS email_otps;
DROP INDEX IF EXISTS idx_refresh_tokens_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user;
DROP TABLE IF EXISTS refresh_tokens;
