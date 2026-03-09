import { apiClient } from './client'
import type { LoginResponse, RegisterResponse } from '../types/api'

export const authAPI = {
  loginPhysician: (email: string, password: string) =>
    apiClient.post<LoginResponse>('/auth/login', { email, password }),

  registerPhysician: (data: {
    email: string
    password: string
    fullName: string
    phone?: string
    mciNumber: string
    specialization?: string
    organizationId?: string
  }) => apiClient.post<RegisterResponse>('/auth/register/physician', data),

  verifyTOTP: (code: string) =>
    apiClient.post<LoginResponse>('/auth/login/verify-totp', { code }),

  logout: () =>
    apiClient.post('/auth/logout'),

  refresh: (refreshToken: string) =>
    apiClient.post<{ accessToken: string; refreshToken: string }>('/auth/refresh', { refreshToken }),

  setupTOTP: () =>
    apiClient.post<{ secret: string; qrCode: string }>('/auth/totp/setup'),

  verifyTOTPSetup: (code: string) =>
    apiClient.post<{ backupCodes: string[]; message: string }>('/auth/totp/verify-setup', { code }),

  getMe: () =>
    apiClient.get<{
      id: string
      role: string
      fullName: string
      status: string
      totpEnabled: boolean
      fhirPatientId?: string
      specialization?: string
      mciNumber?: string
      phone?: string
      email: string
    }>('/auth/me'),
}
