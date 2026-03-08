-- 000005_drug_checker.down.sql
DROP INDEX IF EXISTS idx_loinc_test_name_gin;
DROP INDEX IF EXISTS idx_loinc_test_name;
DROP TABLE IF EXISTS loinc_mappings;
DROP INDEX IF EXISTS idx_document_jobs_status;
DROP INDEX IF EXISTS idx_document_jobs_patient;
DROP TABLE IF EXISTS document_jobs;
DROP INDEX IF EXISTS idx_drug_classes_class;
DROP INDEX IF EXISTS idx_drug_classes_rxnorm;
DROP TABLE IF EXISTS drug_classes;
DROP INDEX IF EXISTS idx_drug_interactions_drug_b;
DROP INDEX IF EXISTS idx_drug_interactions_drug_a;
DROP INDEX IF EXISTS idx_drug_interactions_pair;
DROP TABLE IF EXISTS drug_interactions;
