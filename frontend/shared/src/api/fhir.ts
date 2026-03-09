import { apiClient } from './client'
import type { FHIRBundle, Patient, FHIRResource } from '../types/fhir'

export const fhirAPI = {
  getPatient: (id: string) =>
    apiClient.get<Patient>(`/fhir/R4/Patient/${id}`),

  searchPatients: (params: Record<string, string>) =>
    apiClient.get<FHIRBundle>('/fhir/R4/Patient', { params }),

  getTimeline: (patientId: string, params?: Record<string, string>) =>
    apiClient.get<FHIRBundle>(`/fhir/R4/Patient/${patientId}/$timeline`, { params }),

  getLabTrends: (patientId: string, loincCode: string) =>
    apiClient.get<FHIRBundle>('/fhir/R4/Observation/$lab-trends', {
      params: { patient: `Patient/${patientId}`, code: loincCode },
    }),

  createResource: (resourceType: string, data: unknown) =>
    apiClient.post<FHIRResource>(`/fhir/R4/${resourceType}`, data),

  getResource: (resourceType: string, id: string) =>
    apiClient.get<FHIRResource>(`/fhir/R4/${resourceType}/${id}`),

  searchResources: (resourceType: string, params: Record<string, string>) =>
    apiClient.get<FHIRBundle>(`/fhir/R4/${resourceType}`, { params }),
}
