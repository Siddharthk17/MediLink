-- GIN index on JSONB data — CRITICAL for performance.
CREATE INDEX IF NOT EXISTS idx_fhir_resources_data_gin
    ON fhir_resources USING GIN (data);

-- Resource type filter — used on every single query
CREATE INDEX idx_fhir_resources_type
    ON fhir_resources (resource_type)
    WHERE deleted_at IS NULL;

-- Patient reference — used for patient timeline queries
CREATE INDEX idx_fhir_resources_patient_ref
    ON fhir_resources (patient_ref)
    WHERE deleted_at IS NULL;

-- Composite: type + patient — most common query pattern
CREATE INDEX idx_fhir_resources_type_patient
    ON fhir_resources (resource_type, patient_ref)
    WHERE deleted_at IS NULL;

-- History lookups
CREATE INDEX idx_fhir_history_resource
    ON fhir_resource_history (resource_id, version_id DESC);

-- Audit log queries — by user, by resource, by time
CREATE INDEX idx_audit_logs_user
    ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource
    ON audit_logs (resource_id, created_at DESC);
CREATE INDEX idx_audit_logs_patient
    ON audit_logs (patient_ref, created_at DESC);

-- User email hash lookup
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_role ON users (role) WHERE deleted_at IS NULL;

-- Consent lookups
CREATE INDEX idx_consents_patient ON consents (patient_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_consents_provider ON consents (provider_id) WHERE revoked_at IS NULL;
