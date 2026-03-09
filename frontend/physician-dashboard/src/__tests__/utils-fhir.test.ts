import { describe, it, expect } from 'vitest'
import {
  getPatientName,
  getPatientAge,
  getCodeDisplay,
  getObservationValue,
  getObservationStatus,
} from '@medilink/shared'

describe('FHIR Utils', () => {
  describe('getPatientName', () => {
    it('returns full name from name array', () => {
      const patient = { name: [{ given: ['Ravi'], family: 'Kumar' }] }
      expect(getPatientName(patient as any)).toBe('Ravi Kumar')
    })

    it('returns Unknown Patient for missing name', () => {
      expect(getPatientName({} as any)).toBe('Unknown Patient')
    })

    it('handles multiple given names', () => {
      const patient = { name: [{ given: ['Ravi', 'K'], family: 'Kumar' }] }
      expect(getPatientName(patient as any)).toBe('Ravi K Kumar')
    })

    it('returns text when available', () => {
      const patient = { name: [{ text: 'Dr. Ravi Kumar' }] }
      expect(getPatientName(patient as any)).toBe('Dr. Ravi Kumar')
    })
  })

  describe('getPatientAge', () => {
    it('returns age from birthDate', () => {
      const birthYear = new Date().getFullYear() - 30
      const patient = { birthDate: `${birthYear}-01-01` }
      const age = getPatientAge(patient as any)
      expect(age).toBeGreaterThanOrEqual(29)
      expect(age).toBeLessThanOrEqual(30)
    })

    it('returns null for missing birthDate', () => {
      expect(getPatientAge({} as any)).toBeNull()
    })
  })

  describe('getCodeDisplay', () => {
    it('returns display from coding', () => {
      const code = { coding: [{ system: 'http://loinc.org', code: '4548-4', display: 'HbA1c' }] }
      expect(getCodeDisplay(code)).toBe('HbA1c')
    })

    it('returns text if present', () => {
      const code = { text: 'Hemoglobin A1c' }
      expect(getCodeDisplay(code as any)).toBe('Hemoglobin A1c')
    })

    it('returns Unknown for undefined', () => {
      expect(getCodeDisplay(undefined)).toBe('Unknown')
    })

    it('falls back to code when no display', () => {
      const code = { coding: [{ system: 'http://loinc.org', code: '4548-4' }] }
      expect(getCodeDisplay(code)).toBe('4548-4')
    })
  })

  describe('getObservationValue', () => {
    it('returns formatted value with unit', () => {
      const obs = { valueQuantity: { value: 7.2, unit: '%' } }
      expect(getObservationValue(obs as any)).toContain('7.2')
    })

    it('returns No value for missing value', () => {
      expect(getObservationValue({} as any)).toBe('No value')
    })

    it('returns valueString when present', () => {
      const obs = { valueString: 'Positive' }
      expect(getObservationValue(obs as any)).toBe('Positive')
    })
  })

  describe('getObservationStatus', () => {
    it('capitalizes final status', () => {
      const obs = { status: 'final' }
      expect(getObservationStatus(obs as any)).toBe('Final')
    })

    it('capitalizes preliminary status', () => {
      const obs = { status: 'preliminary' }
      expect(getObservationStatus(obs as any)).toBe('Preliminary')
    })

    it('capitalizes amended status', () => {
      const obs = { status: 'amended' }
      expect(getObservationStatus(obs as any)).toBe('Amended')
    })
  })
})
