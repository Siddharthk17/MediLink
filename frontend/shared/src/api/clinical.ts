import { apiClient } from './client'
import type { DrugCheckResult } from '../types/fhir'

export const clinicalAPI = {
  checkDrugInteractions: (patientId: string, rxnormCode: string, name?: string) =>
    apiClient.post<DrugCheckResult>('/clinical/drug-check', {
      patientId,
      newMedication: { rxnormCode, name: name || '' },
    }),

  acknowledgeDrugInteraction: (
    patientId: string,
    newMedication: string,
    conflictingMedications: string[],
    reason: string
  ) =>
    apiClient.post('/clinical/drug-check/acknowledge', {
      patientId,
      newMedication,
      conflictingMedications,
      reason,
    }),
}
