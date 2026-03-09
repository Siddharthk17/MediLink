export interface AuthUser {
  id: string
  role: 'physician' | 'patient' | 'admin' | 'researcher'
  status: 'active' | 'pending' | 'suspended'
  fullName: string
  email?: string
  phone?: string
  fhirPatientId?: string
  totpEnabled: boolean
  specialization?: string
  mciNumber?: string
}

export interface TokenPair {
  accessToken: string
  refreshToken: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken?: string
  expiresIn: number
  role: string
  requiresTOTP?: boolean
  requiresMFASetup?: boolean
}

export interface RegisterResponse {
  userId: string
  fhirPatientId?: string
  status: string
  message: string
}

export interface ConsentedPatient {
  patient: {
    id: string
    fhirId: string
    fullName: string
    gender?: string
    birthDate?: string
  }
  consent: {
    id: string
    status: 'active' | 'revoked' | 'expired'
    scope: string[]
    grantedAt: string
    expiresAt?: string
  }
}

export interface DocumentJob {
  jobId: string
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'needs-manual-review'
  observationsCreated?: number
  loincMapped?: number
  ocrConfidence?: number
  llmProvider?: string
  errorMessage?: string
  uploadedAt: string
  completedAt?: string
  fhirReportId?: string
  estimatedProcessingTime?: string
}

export interface NotificationPreferences {
  emailDocumentComplete: boolean
  emailDocumentFailed: boolean
  emailConsentGranted: boolean
  emailConsentRevoked: boolean
  emailBreakGlass: boolean
  emailAccountLocked: boolean
  pushEnabled: boolean
  pushDocumentComplete: boolean
  pushNewPrescription: boolean
  pushLabResultReady: boolean
  pushConsentRequest: boolean
  pushCriticalLab: boolean
  preferredLanguage: 'en' | 'hi' | 'mr'
}

export interface AdminStats {
  users: {
    total: number
    patients: number
    physicians: number
    admins: number
    researchers: number
    pending: number
  }
  fhirResources: { total: number; byType: Record<string, number> }
  documents: { totalUploaded: number; completed: number; failed: number; pending: number }
  drugChecks: { total: number; contraindicated: number; major: number; moderate: number }
  consents: { activeGrants: number; revokedThisMonth: number; breakGlassThisMonth: number }
  exports: { total: number; completed: number; pending: number }
}

export interface APIError {
  resourceType: 'OperationOutcome'
  issue: Array<{
    severity: 'fatal' | 'error' | 'warning' | 'information'
    code: string
    details?: { text: string }
    diagnostics?: string
  }>
}
