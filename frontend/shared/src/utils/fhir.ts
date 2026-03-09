import { differenceInYears } from 'date-fns'
import type { Patient, CodeableConcept, Observation, Condition, MedicationRequest } from '../types/fhir'

export function getPatientName(patient: Patient): string {
  if (!patient.name?.length) return 'Unknown Patient'
  const official = patient.name.find((n) => n.use === 'official') ?? patient.name[0]
  if (official.text) return official.text
  const given = official.given?.join(' ') ?? ''
  const family = official.family ?? ''
  return `${given} ${family}`.trim() || 'Unknown Patient'
}

export function getPatientFamilyName(patient: Patient): string {
  if (!patient.name?.length) return ''
  const official = patient.name.find((n) => n.use === 'official') ?? patient.name[0]
  return official.family ?? ''
}

export function getPatientAge(patient: Patient): number | null {
  if (!patient.birthDate) return null
  return differenceInYears(new Date(), new Date(patient.birthDate))
}

export function formatBirthDate(patient: Patient): string {
  if (!patient.birthDate) return 'Unknown'
  return new Date(patient.birthDate).toLocaleDateString('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function formatGender(gender?: string): string {
  if (!gender) return 'Unknown'
  return gender.charAt(0).toUpperCase() + gender.slice(1)
}

export function getCodeDisplay(concept?: CodeableConcept): string {
  if (!concept) return 'Unknown'
  if (concept.text) return concept.text
  const coding = concept.coding?.[0]
  return coding?.display ?? coding?.code ?? 'Unknown'
}

export function getObservationLOINC(obs: Observation): string | null {
  const coding = obs.code?.coding?.find((c) => c.system === 'http://loinc.org')
  return coding?.code ?? null
}

export function getObservationValue(obs: Observation): string {
  if (obs.valueQuantity) {
    const val = obs.valueQuantity.value
    const unit = obs.valueQuantity.unit ?? ''
    return val !== undefined ? `${val} ${unit}`.trim() : 'No value'
  }
  if (obs.valueString) return obs.valueString
  if (obs.dataAbsentReason) return getCodeDisplay(obs.dataAbsentReason)
  return 'No value'
}

export function getObservationInterpretation(obs: Observation): string | null {
  if (!obs.interpretation?.length) return null
  return getCodeDisplay(obs.interpretation[0])
}

export function getObservationStatus(obs: Observation): string {
  return obs.status.charAt(0).toUpperCase() + obs.status.slice(1)
}

export function getConditionDisplay(condition: Condition): string {
  return getCodeDisplay(condition.code)
}

export function getMedicationDisplay(medReq: MedicationRequest): string {
  return getCodeDisplay(medReq.medicationCodeableConcept)
}

const AVATAR_COLORS = [
  '#F43F5E', '#8B5CF6', '#3B82F6', '#10B981',
  '#F59E0B', '#EC4899', '#6366F1', '#14B8A6',
]

export function getPatientAvatarColor(patientId: string): string {
  let hash = 0
  for (let i = 0; i < patientId.length; i++) {
    hash = (hash * 31 + patientId.charCodeAt(i)) | 0
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
}

export function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase()
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase()
}

export function extractPatientId(reference: string): string {
  return reference.replace(/^Patient\//, '')
}

export function buildPatientRef(patientId: string): string {
  return patientId.startsWith('Patient/') ? patientId : `Patient/${patientId}`
}
