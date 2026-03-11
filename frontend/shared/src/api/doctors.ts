import { apiClient } from './client'

export interface DoctorSummary {
  id: string
  fullName: string
  specialization: string
  mciNumber?: string
}

export const doctorsAPI = {
  list: (specialization?: string) => {
    const params: Record<string, string> = {}
    if (specialization) params.specialization = specialization
    return apiClient.get<{ doctors: DoctorSummary[]; total: number }>('/doctors', { params })
  },
}
