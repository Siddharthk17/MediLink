import { apiClient } from './client'
import type { ConsentedPatient } from '../types/api'

export const consentAPI = {
  getMyPatients: () =>
    apiClient.get<{ patients: ConsentedPatient[] }>('/consent/my-patients'),

  grantConsent: (data: { patientId: string; providerId: string; scope: string[]; expiresAt?: string; purpose?: string }) =>
    apiClient.post('/consent/grant', data),

  revokeConsent: (consentId: string) =>
    apiClient.delete(`/consent/${consentId}/revoke`),

  breakGlass: (data: { patientId: string; reason: string }) =>
    apiClient.post('/consent/break-glass', data),
}
