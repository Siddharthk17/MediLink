import type { FHIRResourceType } from '../types/fhir'

export const FHIR_RESOURCE_TYPES: FHIRResourceType[] = [
  'Patient', 'Practitioner', 'Organization', 'Encounter',
  'Condition', 'MedicationRequest', 'Observation', 'DiagnosticReport',
  'AllergyIntolerance', 'Immunization',
]

export const ENCOUNTER_STATUSES = [
  'planned', 'arrived', 'triaged', 'in-progress', 'onleave', 'finished', 'cancelled',
] as const

export const MEDICATION_STATUSES = [
  'active', 'on-hold', 'cancelled', 'completed', 'entered-in-error', 'stopped',
] as const

export const OBSERVATION_STATUSES = [
  'registered', 'preliminary', 'final', 'amended', 'corrected',
] as const
