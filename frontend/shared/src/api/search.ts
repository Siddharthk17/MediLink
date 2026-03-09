import { apiClient } from './client'
import type { FHIRBundle } from '../types/fhir'

export const searchAPI = {
  unifiedSearch: (query: string, params?: Record<string, string>) =>
    apiClient.get<FHIRBundle>('/search', { params: { q: query, ...params } }),
}
