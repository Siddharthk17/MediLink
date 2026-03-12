# FHIR R4 Compliance

This document describes how MediLink implements the [HL7 FHIR R4](https://hl7.org/fhir/R4/) specification — which resource types are supported, what search parameters are available, and how data is stored and returned.

---

## Table of Contents

- [Overview](#overview)
- [Supported Resources](#supported-resources)
- [Resource Storage Model](#resource-storage-model)
- [Search Parameters by Resource](#search-parameters-by-resource)
- [FHIR Bundle Responses](#fhir-bundle-responses)
- [Resource Versioning & History](#resource-versioning--history)
- [Custom Operations](#custom-operations)
- [Validation Rules](#validation-rules)
- [Content Types](#content-types)
- [Reference Validation](#reference-validation)

---

## Overview

MediLink implements a subset of FHIR R4 focused on the clinical resources needed for a healthcare coordination platform. All resources are stored as JSONB in PostgreSQL with server-assigned IDs and metadata.

**What MediLink implements:**
- 10 clinical resource types
- CRUD operations (create, read, update, delete) for all resources
- Search with FHIR-standard query parameters
- Resource versioning and history
- FHIR Bundle responses for search and history
- Two custom operations (`$timeline` and `$lab-trends`)
- Cross-resource reference validation

**What MediLink does not implement:**
- FHIR XML format (JSON only)
- Capability statement / conformance endpoint
- Subscriptions
- Batch/Transaction bundles
- Contained resources
- Extensions beyond standard fields

---

## Supported Resources

| Resource | Description | Patient-Scoped |
|---|---|---|
| [Patient](https://hl7.org/fhir/R4/patient.html) | Demographics, identifiers, contact information | Self |
| [Practitioner](https://hl7.org/fhir/R4/practitioner.html) | Healthcare provider information | No |
| [Organization](https://hl7.org/fhir/R4/organization.html) | Healthcare organizations | No |
| [Encounter](https://hl7.org/fhir/R4/encounter.html) | Clinical visits and admissions | Yes |
| [Condition](https://hl7.org/fhir/R4/condition.html) | Diagnoses and health conditions | Yes |
| [MedicationRequest](https://hl7.org/fhir/R4/medicationrequest.html) | Prescriptions and medication orders | Yes |
| [Observation](https://hl7.org/fhir/R4/observation.html) | Lab results, vitals, clinical findings | Yes |
| [DiagnosticReport](https://hl7.org/fhir/R4/diagnosticreport.html) | Lab reports and diagnostic studies | Yes |
| [AllergyIntolerance](https://hl7.org/fhir/R4/allergyintolerance.html) | Allergies and adverse reactions | Yes |
| [Immunization](https://hl7.org/fhir/R4/immunization.html) | Vaccination records | Yes |

**Patient-Scoped** means the resource is linked to a specific patient via a `patient_ref` column. Consent-based access control is enforced for these resources.

---

## Resource Storage Model

All FHIR resources are stored in a single PostgreSQL table:

```sql
CREATE TABLE fhir_resources (
    id            UUID PRIMARY KEY,
    fhir_id       TEXT UNIQUE NOT NULL,
    resource_type TEXT NOT NULL,
    data          JSONB NOT NULL,
    patient_ref   TEXT,
    version       INTEGER DEFAULT 1,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    deleted_at    TIMESTAMPTZ
);
```

Key design decisions:
- **`fhir_id`** is the logical ID used in FHIR references (e.g., `Patient/abc-123`)
- **`data`** stores the full FHIR JSON resource
- **`patient_ref`** stores the patient FHIR ID for consent scoping (format: `Patient/abc-123`)
- **`version`** is incremented on each update
- **`deleted_at`** enables soft delete (resource is marked deleted, not removed)
- Patients are stored in a separate `patients` table with additional PII encryption

---

## Search Parameters by Resource

All search endpoints support pagination via `_count` (default: 20, max: 100) and `_offset` (default: 0).

### Patient

| Parameter | Type | Description |
|---|---|---|
| `family` | string | Family (last) name |
| `given` | string | Given (first) name |
| `name` | string | Any part of the name (family, given, text) |
| `birthdate` | date | Date of birth |
| `gender` | token | `male`, `female`, `other`, `unknown` |
| `identifier` | token | Identifier value (e.g., MRN) |
| `active` | boolean | Whether the patient record is active |

### Practitioner

| Parameter | Type | Description |
|---|---|---|
| `family` | string | Family name |
| `given` | string | Given name |
| `name` | string | Any part of the name |
| `identifier` | token | Identifier value (e.g., NPI) |
| `gender` | token | Gender |
| `active` | boolean | Active status |

### Organization

| Parameter | Type | Description |
|---|---|---|
| `name` | string | Organization name (partial match) |
| `identifier` | token | Identifier value |
| `type` | token | Organization type code |
| `active` | boolean | Active status |

### Encounter

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `status` | token | `planned`, `arrived`, `in-progress`, `finished`, etc. |
| `class` | token | `AMB`, `IMP`, `EMER`, etc. |
| `date` | date | Encounter start date (supports prefixes) |
| `practitioner` | reference | Practitioner reference |
| `organization` | reference | Service provider reference |

### Condition

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `clinical-status` | token | `active`, `recurrence`, `relapse`, `inactive`, `remission`, `resolved` |
| `verification-status` | token | `unconfirmed`, `provisional`, `differential`, `confirmed` |
| `code` | token | Condition code (e.g., ICD-10 or SNOMED CT code) |
| `onset-date` | date | Onset date (supports prefixes) |
| `encounter` | reference | Encounter reference |

### MedicationRequest

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `status` | token | `active`, `on-hold`, `cancelled`, `completed`, `stopped`, `draft`, `entered-in-error`, `unknown` |
| `medication` | token | Medication code |
| `intent` | token | `proposal`, `plan`, `order`, `original-order`, `reflex-order`, `filler-order`, `instance-order`, `option` |
| `encounter` | reference | Encounter reference |
| `authored-on` | date | Authored date (supports prefixes) |
| `requester` | reference | Prescribing practitioner reference |

### Observation

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `code` | token | Observation code (e.g., LOINC code) |
| `category` | token | Category code (e.g., `laboratory`, `vital-signs`) |
| `status` | token | `registered`, `preliminary`, `final`, `amended` |
| `date` | date | Effective date (supports prefixes) |
| `encounter` | reference | Encounter reference |
| `value-quantity` | quantity | Numeric value comparison (supports prefixes) |

### DiagnosticReport

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `status` | token | `registered`, `partial`, `preliminary`, `final` |
| `category` | token | Category code |
| `code` | token | Report code |
| `date` | date | Effective date (supports prefixes) |
| `encounter` | reference | Encounter reference |

### AllergyIntolerance

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `clinical-status` | token | `active`, `inactive`, `resolved` |
| `criticality` | token | `low`, `high`, `unable-to-assess` |
| `code` | token | Allergy/substance code |
| `category` | token | `food`, `medication`, `environment`, `biologic` |

### Immunization

| Parameter | Type | Description |
|---|---|---|
| `patient` | reference | Patient FHIR ID |
| `status` | token | `completed`, `entered-in-error`, `not-done` |
| `vaccine-code` | token | Vaccine code (e.g., CVX code) |
| `date` | date | Occurrence date (supports prefixes) |

### Date Prefixes

Date parameters support comparison prefixes:

| Prefix | Meaning | Example |
|---|---|---|
| `eq` | Equal (default) | `date=eq2024-01-15` |
| `gt` | Greater than | `date=gt2024-01-01` |
| `lt` | Less than | `date=lt2024-12-31` |
| `ge` | Greater or equal | `date=ge2024-06-01` |
| `le` | Less or equal | `date=le2024-06-30` |

---

## FHIR Bundle Responses

Search and history endpoints return FHIR Bundle resources:

```json
{
  "resourceType": "Bundle",
  "type": "searchset",
  "total": 42,
  "link": [
    {
      "relation": "self",
      "url": "https://medilink.dev/fhir/Observation?patient=Patient/abc-123&_count=20&_offset=0"
    },
    {
      "relation": "next",
      "url": "https://medilink.dev/fhir/Observation?patient=Patient/abc-123&_count=20&_offset=20"
    }
  ],
  "entry": [
    {
      "fullUrl": "https://medilink.dev/fhir/Observation/obs-uuid-1",
      "resource": {
        "resourceType": "Observation",
        "id": "obs-uuid-1",
        "status": "final",
        "code": { ... },
        "subject": { "reference": "Patient/abc-123" },
        "effectiveDateTime": "2024-03-15T10:30:00Z",
        "valueQuantity": { "value": 120, "unit": "mg/dL" }
      }
    }
  ]
}
```

History bundles use `"type": "history"` and include a `request` object in each entry showing the HTTP method.

---

## Resource Versioning & History

Every FHIR resource has a version number that starts at 1 and increments on each update.

### Reading a Specific Version

```
GET /fhir/Observation/obs-uuid-1/_history/3
```

Returns version 3 of the Observation.

### Reading Version History

```
GET /fhir/Observation/obs-uuid-1/_history
```

Returns a Bundle containing all versions, most recent first.

### Version Metadata

The `meta` field on each resource includes:

```json
{
  "meta": {
    "versionId": "3",
    "lastUpdated": "2024-03-15T10:30:00Z"
  }
}
```

---

## Custom Operations

### Patient Timeline — `$timeline`

```
GET /fhir/Patient/{fhirId}/$timeline?_count=50&_offset=0
```

Returns a chronological Bundle of all clinical resources linked to a patient, sorted by date (most recent first). This includes Encounters, Conditions, Observations, MedicationRequests, DiagnosticReports, AllergyIntolerances, and Immunizations.

Useful for displaying a patient's complete health timeline in the dashboard.

### Lab Trends — `$lab-trends`

```
GET /fhir/Observation/$lab-trends?patient={patientFhirId}&code={loincCode}
```

Returns all Observations for a patient with a specific LOINC code, sorted chronologically. Used to graph trends for a specific lab test over time (e.g., blood glucose readings across months).

---

## Validation Rules

Every resource is validated before storage. Common validation rules:

### All Resources
- `resourceType` must match the endpoint (e.g., `POST /fhir/Patient` requires `resourceType: "Patient"`)
- If `id` is provided in the body, it must match the URL parameter (for updates)
- All referenced resources must exist and be active (see [Reference Validation](#reference-validation))

### Patient
- `name` array with at least one entry containing `family` name is required
- `gender` is required and must be one of: `male`, `female`, `other`, `unknown`
- `birthDate` must be a valid date in `YYYY-MM-DD` format if provided

### Practitioner
- `name` array with at least one entry containing `family` name is required
- `gender` must be valid if provided

### Organization
- `name` is required

### Encounter
- `status` is required (valid: `planned`, `arrived`, `triaged`, `in-progress`, `onleave`, `finished`, `cancelled`, `entered-in-error`, `unknown`)
- `class` coding is required with `system` and `code`
- `subject` reference to a Patient is required
- `period.start` must be before `period.end` if both are provided

### Condition
- `subject` reference to a Patient is required
- `code` with at least one coding is required
- `clinicalStatus` is required with valid value

### MedicationRequest
- `status` is required
- `intent` is required
- `subject` reference to a Patient is required
- `medicationCodeableConcept` with at least one coding is required
- Drug interaction check is performed automatically — **contraindicated** interactions block creation

### Observation
- `status` is required
- `code` with at least one coding is required
- `subject` reference to a Patient is required

### DiagnosticReport
- `status` is required
- `code` with at least one coding is required

### AllergyIntolerance
- `patient` reference is required
- `code` with at least one coding is required

### Immunization
- `status` is required
- `vaccineCode` with at least one coding is required
- `patient` reference is required
- `occurrenceDateTime` is required

---

## Content Types

| Header | Value |
|---|---|
| Request `Content-Type` | `application/json` or `application/fhir+json` |
| Response `Content-Type` | `application/fhir+json; charset=utf-8` |

All responses use FHIR JSON format. XML is not supported.

---

## Reference Validation

When a resource references another resource (e.g., an Observation referencing a Patient), MediLink validates that:

1. The referenced resource exists in the database
2. The referenced resource is not deleted
3. The resource type matches the expected type (e.g., a `subject` reference must point to a Patient)

Invalid references return a `422 Unprocessable Entity` error with a FHIR OperationOutcome:

```json
{
  "resourceType": "OperationOutcome",
  "issue": [
    {
      "severity": "error",
      "code": "processing",
      "diagnostics": "Referenced resource Patient/nonexistent-id not found"
    }
  ]
}
```

References use the format `ResourceType/fhir-id` (e.g., `Patient/abc-123`, `Practitioner/dr-456`).
