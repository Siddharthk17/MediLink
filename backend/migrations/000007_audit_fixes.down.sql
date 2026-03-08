DROP INDEX IF EXISTS idx_fhir_resources_type_id;
DROP INDEX IF EXISTS idx_fhir_resources_resource_id;
ALTER TABLE consents DROP COLUMN IF EXISTS status;
ALTER TABLE consents DROP COLUMN IF EXISTS revoked_reason;
