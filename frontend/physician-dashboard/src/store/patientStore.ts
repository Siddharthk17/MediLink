import { create } from 'zustand'
import type { Patient } from '@medilink/shared'

interface PatientState {
  activePatient: Patient | null
  setActivePatient: (patient: Patient | null) => void
}

export const usePatientStore = create<PatientState>((set) => ({
  activePatient: null,
  setActivePatient: (patient) => set({ activePatient: patient }),
}))
