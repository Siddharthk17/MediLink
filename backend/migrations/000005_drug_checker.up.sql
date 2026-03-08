-- 000005_drug_checker.up.sql

-- ============================================================
-- DRUG INTERACTION CACHE
-- Stores OpenFDA drug interaction lookup results permanently.
-- The same drug pair is never queried from OpenFDA twice.
-- drug_a and drug_b are always stored in alphabetical order
-- to ensure (Metformin, Warfarin) and (Warfarin, Metformin)
-- hit the same cache entry.
-- ============================================================
CREATE TABLE drug_interactions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drug_a_rxnorm   VARCHAR(32) NOT NULL,
    drug_b_rxnorm   VARCHAR(32) NOT NULL,
    drug_a_name     VARCHAR(255) NOT NULL,
    drug_b_name     VARCHAR(255) NOT NULL,
    severity        VARCHAR(32) NOT NULL,
    description     TEXT NOT NULL,
    mechanism       TEXT,
    clinical_effect TEXT,
    management      TEXT,
    source          VARCHAR(64) NOT NULL DEFAULT 'openfda',
    source_url      TEXT,
    last_verified   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_drug_pair UNIQUE (drug_a_rxnorm, drug_b_rxnorm)
);

CREATE INDEX idx_drug_interactions_pair
    ON drug_interactions (drug_a_rxnorm, drug_b_rxnorm);

CREATE INDEX idx_drug_interactions_drug_a ON drug_interactions (drug_a_rxnorm);
CREATE INDEX idx_drug_interactions_drug_b ON drug_interactions (drug_b_rxnorm);

-- ============================================================
-- DRUG CLASS TABLE
-- Maps RxNorm codes to drug classes.
-- Used for allergy cross-reactivity checking.
-- ============================================================
CREATE TABLE drug_classes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rxnorm_code     VARCHAR(32) NOT NULL UNIQUE,
    drug_name       VARCHAR(255) NOT NULL,
    drug_class      VARCHAR(128) NOT NULL,
    drug_subclass   VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_drug_classes_rxnorm ON drug_classes (rxnorm_code);
CREATE INDEX idx_drug_classes_class ON drug_classes (drug_class);

-- ============================================================
-- DOCUMENT JOBS TABLE
-- Tracks every file upload and its processing status.
-- ============================================================
CREATE TABLE document_jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id      UUID NOT NULL REFERENCES users(id),
    patient_fhir_id VARCHAR(64) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    minio_bucket    VARCHAR(128) NOT NULL,
    minio_key       VARCHAR(512) NOT NULL,
    file_size       BIGINT NOT NULL,
    content_type    VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    asynq_task_id   VARCHAR(128),
    fhir_report_id  VARCHAR(64),
    observations_created INTEGER,
    loinc_mapped    INTEGER,
    ocr_confidence  FLOAT,
    llm_provider    VARCHAR(32),
    error_message   TEXT,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_started_at TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    uploaded_by     UUID NOT NULL REFERENCES users(id)
);

CREATE INDEX idx_document_jobs_patient ON document_jobs (patient_fhir_id, uploaded_at DESC);
CREATE INDEX idx_document_jobs_status ON document_jobs (status) WHERE status != 'completed';

-- ============================================================
-- LOINC MAPPING TABLE
-- Maps common Indian lab test names to LOINC codes.
-- ============================================================
CREATE TABLE loinc_mappings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    test_name       VARCHAR(255) NOT NULL,
    test_name_lower VARCHAR(255) NOT NULL,
    loinc_code      VARCHAR(32) NOT NULL,
    loinc_display   VARCHAR(255) NOT NULL,
    unit            VARCHAR(32),
    normal_low      FLOAT,
    normal_high     FLOAT,
    category        VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loinc_test_name ON loinc_mappings (test_name_lower);
CREATE INDEX idx_loinc_test_name_gin ON loinc_mappings USING GIN (to_tsvector('english', test_name));
