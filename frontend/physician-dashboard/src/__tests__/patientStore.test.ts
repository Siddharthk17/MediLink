import { beforeEach } from 'vitest'
import { usePatientStore } from '@/store/patientStore'
import type { Patient } from '@medilink/shared'

const mockPatient: Patient = {
  resourceType: 'Patient',
  id: 'patient-1',
  meta: { versionId: '1', lastUpdated: '2024-01-01T00:00:00Z' },
  name: [{ family: 'Doe', given: ['John'], use: 'official' }],
  gender: 'male',
  birthDate: '1990-05-15',
}

const anotherPatient: Patient = {
  resourceType: 'Patient',
  id: 'patient-2',
  meta: { versionId: '1', lastUpdated: '2024-02-01T00:00:00Z' },
  name: [{ family: 'Smith', given: ['Jane'], use: 'official' }],
  gender: 'female',
  birthDate: '1985-11-20',
}

describe('patientStore', () => {
  beforeEach(() => {
    usePatientStore.getState().setActivePatient(null)
  })

  it('starts with null activePatient', () => {
    const state = usePatientStore.getState()
    expect(state.activePatient).toBeNull()
  })

  it('sets an active patient', () => {
    usePatientStore.getState().setActivePatient(mockPatient)
    const state = usePatientStore.getState()
    expect(state.activePatient).toEqual(mockPatient)
    expect(state.activePatient?.id).toBe('patient-1')
  })

  it('replaces the active patient with another', () => {
    usePatientStore.getState().setActivePatient(mockPatient)
    usePatientStore.getState().setActivePatient(anotherPatient)
    const state = usePatientStore.getState()
    expect(state.activePatient?.id).toBe('patient-2')
    expect(state.activePatient?.name?.[0].family).toBe('Smith')
  })

  it('clears the active patient by setting null', () => {
    usePatientStore.getState().setActivePatient(mockPatient)
    usePatientStore.getState().setActivePatient(null)
    expect(usePatientStore.getState().activePatient).toBeNull()
  })

  it('preserves full patient data including optional fields', () => {
    const patientWithDetails: Patient = {
      ...mockPatient,
      telecom: [{ system: 'phone', value: '555-0100', use: 'home' }],
      address: [{ city: 'Springfield', state: 'IL' }],
    }
    usePatientStore.getState().setActivePatient(patientWithDetails)
    const state = usePatientStore.getState()
    expect(state.activePatient?.telecom?.[0].value).toBe('555-0100')
    expect(state.activePatient?.address?.[0].city).toBe('Springfield')
  })
})
