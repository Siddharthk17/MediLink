-- ============================================================
-- TOTP BACKUP CODES TABLE
-- Stores bcrypt-hashed backup codes for 2FA recovery.
-- Each user gets 10 codes at TOTP setup; each is single-use.
-- ============================================================
CREATE TABLE IF NOT EXISTS totp_backup_codes (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user
    ON totp_backup_codes (user_id)
    WHERE used_at IS NULL;
