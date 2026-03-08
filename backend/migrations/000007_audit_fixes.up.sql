-- Add missing index on (resource_type, resource_id) for FHIR resource lookup
CREATE INDEX IF NOT EXISTS idx_fhir_resources_type_id 
  ON fhir_resources (resource_type, resource_id) 
  WHERE deleted_at IS NULL;

-- Add missing index on resource_id alone
CREATE INDEX IF NOT EXISTS idx_fhir_resources_resource_id 
  ON fhir_resources (resource_id) 
  WHERE deleted_at IS NULL;

-- Add missing columns to consents table
ALTER TABLE consents ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';
ALTER TABLE consents ADD COLUMN IF NOT EXISTS revoked_reason TEXT;

-- Update existing consents: set status based on revoked_at
UPDATE consents SET status = 'revoked' WHERE revoked_at IS NOT NULL AND status = 'active';
