import { apiClient } from './client'
import type { ConsentedPatient, ConsentGrant, AccessLogEntry } from '../types/api'

export const consentAPI = {
  getMyPatients: () =>
    apiClient.get<{ patients: ConsentedPatient[] }>('/consent/my-patients'),

  getMyGrants: () =>
    apiClient.get<{ consents: ConsentGrant[]; total: number }>('/consent/my-grants'),

  getAccessLog: () =>
    apiClient.get<{ patientId: string; entries: AccessLogEntry[] }>('/consent/access-log'),

  grantConsent: (data: { providerId: string; scope: string[]; expiresAt?: string; purpose?: string }) =>
    apiClient.post('/consent/grant', data),

  revokeConsent: (consentId: string) =>
    apiClient.delete(`/consent/${consentId}/revoke`),

  acceptConsent: (consentId: string) =>
    apiClient.put(`/consent/${consentId}/accept`),

  declineConsent: (consentId: string, reason?: string) =>
    apiClient.put(`/consent/${consentId}/decline`, { reason }),

  getPendingRequests: () =>
    apiClient.get<{ requests: ConsentedPatient[]; total: number }>('/consent/pending-requests'),

  breakGlass: (data: { patientId: string; reason: string }) =>
    apiClient.post('/consent/break-glass', data),
}
