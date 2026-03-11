export interface FHIRResource {
  resourceType: string
  id: string
  meta: {
    versionId: string
    lastUpdated: string
  }
}

export interface HumanName {
  family?: string
  given?: string[]
  text?: string
  use?: 'official' | 'nickname' | 'anonymous'
}

export interface CodeableConcept {
  coding?: Array<{ system: string; code: string; display?: string }>
  text?: string
}

export interface Quantity {
  value?: number
  unit?: string
  system?: string
  code?: string
}

export interface Reference {
  reference: string
  display?: string
}

export interface Patient extends FHIRResource {
  resourceType: 'Patient'
  name?: HumanName[]
  gender?: 'male' | 'female' | 'other' | 'unknown'
  birthDate?: string
  telecom?: Array<{ system: string; value: string; use?: string }>
  address?: Array<{ text?: string; city?: string; state?: string }>
}

export interface Practitioner extends FHIRResource {
  resourceType: 'Practitioner'
  name?: HumanName[]
  qualification?: Array<{ code: CodeableConcept; identifier?: unknown[] }>
}

export interface Encounter extends FHIRResource {
  resourceType: 'Encounter'
  status: 'planned' | 'arrived' | 'triaged' | 'in-progress' | 'onleave' | 'finished' | 'cancelled'
  class: { code: string; display?: string }
  subject: Reference
  period?: { start?: string; end?: string }
}

export interface Condition extends FHIRResource {
  resourceType: 'Condition'
  clinicalStatus?: CodeableConcept
  verificationStatus?: CodeableConcept
  severity?: CodeableConcept
  code?: CodeableConcept
  subject: Reference
  onsetDateTime?: string
  recordedDate?: string
}

export interface MedicationRequest extends FHIRResource {
  resourceType: 'MedicationRequest'
  status: 'active' | 'on-hold' | 'cancelled' | 'completed' | 'entered-in-error' | 'stopped' | 'draft' | 'unknown'
  intent: 'proposal' | 'plan' | 'order'
  medicationCodeableConcept?: CodeableConcept
  subject: Reference
  authoredOn?: string
  dosageInstruction?: Array<{
    text?: string
    timing?: unknown
    doseAndRate?: Array<{ doseQuantity?: Quantity }>
  }>
}

export type InteractionSeverity = 'contraindicated' | 'major' | 'moderate' | 'minor' | 'unknown' | 'none'

export interface MedicationInfo {
  rxnormCode: string
  name: string
}

export interface DrugCheckResult {
  newMedication: MedicationInfo
  interactions: Array<{
    drugA: MedicationInfo
    drugB: MedicationInfo
    severity: InteractionSeverity
    description: string
    mechanism?: string
    clinicalEffect?: string
    management?: string
    source: string
    cached: boolean
  }>
  allergyConflicts: Array<{
    allergen: MedicationInfo
    newMedication: MedicationInfo
    severity: InteractionSeverity
    mechanism: string
    drugClass?: string
    reaction?: string
  }>
  highestSeverity: InteractionSeverity
  hasContraindication: boolean
  checkComplete: boolean
  checkError?: string
}

export type ObservationValue =
  | { valueQuantity: Quantity }
  | { valueString: string }
  | { valueBoolean: boolean }
  | { valueCodeableConcept: CodeableConcept }
  | { dataAbsentReason: CodeableConcept }

export interface Observation extends FHIRResource {
  resourceType: 'Observation'
  status: 'registered' | 'preliminary' | 'final' | 'amended' | 'corrected'
  code: CodeableConcept
  subject: Reference
  effectiveDateTime?: string
  issued?: string
  interpretation?: CodeableConcept[]
  referenceRange?: Array<{
    low?: Quantity
    high?: Quantity
    text?: string
  }>
  valueQuantity?: Quantity
  valueString?: string
  dataAbsentReason?: CodeableConcept
}

export interface DiagnosticReport extends FHIRResource {
  resourceType: 'DiagnosticReport'
  status: 'registered' | 'partial' | 'preliminary' | 'final'
  code: CodeableConcept
  subject: Reference
  effectiveDateTime?: string
  result?: Reference[]
  presentedForm?: Array<{
    url?: string
    contentType?: string
    title?: string
  }>
}

export interface AllergyIntolerance extends FHIRResource {
  resourceType: 'AllergyIntolerance'
  clinicalStatus?: CodeableConcept
  verificationStatus?: CodeableConcept
  criticality?: 'low' | 'high' | 'unable-to-assess'
  code?: CodeableConcept
  patient: Reference
  onsetDateTime?: string
  reaction?: Array<{
    substance?: CodeableConcept
    manifestation: CodeableConcept[]
    severity?: 'mild' | 'moderate' | 'severe'
  }>
}

export interface Immunization extends FHIRResource {
  resourceType: 'Immunization'
  status: 'completed' | 'entered-in-error' | 'not-done'
  vaccineCode: CodeableConcept
  patient: Reference
  occurrenceDateTime?: string
  primarySource?: boolean
  protocolApplied?: Array<{
    doseNumberPositiveInt?: number
    seriesDosesPositiveInt?: number
  }>
}

export interface FHIRBundle {
  resourceType: 'Bundle'
  type: 'searchset' | 'history'
  total: number
  entry?: Array<{
    resource: Patient | Practitioner | Encounter | Condition |
              MedicationRequest | Observation | DiagnosticReport |
              AllergyIntolerance | Immunization
    search?: { score?: number }
  }>
}

export interface TimelineBundle extends FHIRBundle {}

export type FHIRResourceType =
  | 'Patient' | 'Practitioner' | 'Organization' | 'Encounter'
  | 'Condition' | 'MedicationRequest' | 'Observation' | 'DiagnosticReport'
  | 'AllergyIntolerance' | 'Immunization'
