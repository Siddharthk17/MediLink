-- ============================================================
-- NOTIFICATION PREFERENCES
-- One row per user. Created with defaults on first access.
-- ============================================================
CREATE TABLE notification_preferences (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Email channel preferences
    email_document_complete  BOOLEAN NOT NULL DEFAULT true,
    email_document_failed    BOOLEAN NOT NULL DEFAULT true,
    email_consent_granted    BOOLEAN NOT NULL DEFAULT true,
    email_consent_revoked    BOOLEAN NOT NULL DEFAULT true,
    -- These two cannot be disabled by the user (enforced in service layer):
    email_break_glass        BOOLEAN NOT NULL DEFAULT true,
    email_account_locked     BOOLEAN NOT NULL DEFAULT true,

    -- Push channel preferences
    push_enabled             BOOLEAN NOT NULL DEFAULT true,
    push_document_complete   BOOLEAN NOT NULL DEFAULT true,
    push_new_prescription    BOOLEAN NOT NULL DEFAULT true,
    push_lab_result_ready    BOOLEAN NOT NULL DEFAULT true,
    push_consent_request     BOOLEAN NOT NULL DEFAULT false,
    push_critical_lab        BOOLEAN NOT NULL DEFAULT true,

    -- Device token for FCM
    fcm_token                TEXT,
    fcm_token_updated_at     TIMESTAMPTZ,

    -- Notification language (en, hi, mr)
    preferred_language       VARCHAR(5) NOT NULL DEFAULT 'en',

    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_notif_prefs_user UNIQUE (user_id)
);

CREATE INDEX idx_notif_prefs_user ON notification_preferences (user_id);

-- ============================================================
-- RESEARCH EXPORT JOBS
-- Tracks anonymized FHIR data export requests.
-- ============================================================
CREATE TABLE research_exports (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requested_by     UUID NOT NULL REFERENCES users(id),

    -- What to export
    resource_types   TEXT[]   NOT NULL,
    date_from        TIMESTAMPTZ,
    date_to          TIMESTAMPTZ,
    filters          JSONB,
    k_value          INTEGER  NOT NULL DEFAULT 5,

    -- Job status
    status           VARCHAR(32) NOT NULL DEFAULT 'pending',
                     -- pending | processing | completed | failed

    -- Results
    record_count     INTEGER,            -- before suppression
    exported_count   INTEGER,            -- after suppression
    suppressed_count INTEGER,            -- records removed for k-anonymity
    minio_bucket     TEXT,
    minio_key        TEXT,               -- manifest JSON key
    file_size        BIGINT,

    -- Audit fields
    purpose          TEXT NOT NULL,      -- researcher must state purpose (min 20 chars)
    export_salt      VARCHAR(64),        -- random salt for pseudonymization

    -- Admin approval (optional — can be made required per policy)
    approved_by      UUID REFERENCES users(id),
    approved_at      TIMESTAMPTZ,

    error_message    TEXT,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ
);

CREATE INDEX idx_research_exports_requester
    ON research_exports (requested_by, created_at DESC);
CREATE INDEX idx_research_exports_status
    ON research_exports (status) WHERE status != 'completed';

-- ============================================================
-- SEARCH QUERY LOG
-- Analytics for search usage.
-- ============================================================
CREATE TABLE search_queries (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id       UUID NOT NULL,
    actor_role     VARCHAR(32) NOT NULL,
    query_text     TEXT NOT NULL,
    resource_types TEXT[],
    results_count  INTEGER,
    duration_ms    INTEGER,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_search_queries_actor
    ON search_queries (actor_id, created_at DESC);

-- ============================================================
-- Add researcher role to users check constraint
-- ============================================================
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('patient', 'physician', 'admin', 'researcher'));
