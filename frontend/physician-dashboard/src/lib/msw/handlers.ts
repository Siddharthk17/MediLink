import { http, HttpResponse } from 'msw'
import type { ConsentedPatient } from '@medilink/shared'

const API_BASE = 'http://localhost:8580'

const MOCK_PATIENTS: ConsentedPatient[] = [
  {
    patient: { id: 'u-p1', fhirId: 'p1', fullName: 'Ravi Kumar', gender: 'male', birthDate: '1985-06-15' },
    consent: { id: 'c1', status: 'active', scope: ['read', 'write'], grantedAt: '2024-01-15T10:00:00Z', expiresAt: '2025-01-15T10:00:00Z' },
  },
  {
    patient: { id: 'u-p2', fhirId: 'p2', fullName: 'Priya Singh', gender: 'female', birthDate: '1990-11-22' },
    consent: { id: 'c2', status: 'active', scope: ['read', 'emergency'], grantedAt: '2024-02-20T14:30:00Z' },
  },
  {
    patient: { id: 'u-p3', fhirId: 'p3', fullName: 'Amit Patel', gender: 'male', birthDate: '1978-03-08' },
    consent: { id: 'c3', status: 'active', scope: ['read', 'write', 'emergency'], grantedAt: '2024-03-10T09:00:00Z', expiresAt: '2025-03-10T09:00:00Z' },
  },
  {
    patient: { id: 'u-p4', fhirId: 'p4', fullName: 'Neha Sharma', gender: 'female', birthDate: '1995-07-19' },
    consent: { id: 'c4', status: 'expired', scope: ['read'], grantedAt: '2023-04-05T11:30:00Z', expiresAt: '2024-04-05T11:30:00Z' },
  },
]

const MOCK_FHIR_PATIENTS: Record<string, object> = {
  p1: {
    resourceType: 'Patient', id: 'p1',
    name: [{ given: ['Ravi'], family: 'Kumar' }],
    gender: 'male', birthDate: '1985-06-15',
    telecom: [{ system: 'phone', value: '+91-9876543210' }, { system: 'email', value: 'ravi.kumar@email.com' }],
    address: [{ city: 'Mumbai', state: 'Maharashtra', country: 'IN' }],
  },
  p2: {
    resourceType: 'Patient', id: 'p2',
    name: [{ given: ['Priya'], family: 'Singh' }],
    gender: 'female', birthDate: '1990-11-22',
    telecom: [{ system: 'phone', value: '+91-9123456789' }],
    address: [{ city: 'Delhi', state: 'Delhi', country: 'IN' }],
  },
  p3: {
    resourceType: 'Patient', id: 'p3',
    name: [{ given: ['Amit'], family: 'Patel' }],
    gender: 'male', birthDate: '1978-03-08',
    telecom: [{ system: 'phone', value: '+91-9988776655' }],
    address: [{ city: 'Ahmedabad', state: 'Gujarat', country: 'IN' }],
  },
  p4: {
    resourceType: 'Patient', id: 'p4',
    name: [{ given: ['Neha'], family: 'Sharma' }],
    gender: 'female', birthDate: '1995-07-19',
    telecom: [{ system: 'phone', value: '+91-9011223344' }],
    address: [{ city: 'Bangalore', state: 'Karnataka', country: 'IN' }],
  },
}

function createHandlers(base: string) {
  return [
    // Auth
    http.post(`${base}/auth/login/physician`, async ({ request }) => {
      const body = await request.json() as { email: string; password: string }
      if (body.email === 'doctor@medilink.in' && body.password === 'password123') {
        return HttpResponse.json({
          accessToken: 'mock-access-token',
          refreshToken: 'mock-refresh-token',
          user: {
            id: 'user-1',
            email: 'doctor@medilink.in',
            role: 'physician',
            firstName: 'Dr. Arjun',
            lastName: 'Mehta',
            totpEnabled: false,
          },
        })
      }
      if (body.email === 'admin@medilink.in' && body.password === 'password123') {
        return HttpResponse.json({
          accessToken: 'mock-admin-token',
          refreshToken: 'mock-admin-refresh',
          user: {
            id: 'user-3',
            email: 'admin@medilink.in',
            role: 'admin',
            firstName: 'Admin',
            lastName: 'User',
            totpEnabled: false,
          },
        })
      }
      if (body.email === 'totp@medilink.in') {
        return HttpResponse.json({ totpRequired: true, partialToken: 'partial-totp-token' })
      }
      return HttpResponse.json({ error: 'Invalid credentials' }, { status: 401 })
    }),

    http.post(`${base}/auth/login/verify-totp`, async ({ request }) => {
      const body = await request.json() as { code: string }
      if (body.code === '123456') {
        return HttpResponse.json({
          accessToken: 'mock-totp-token',
          refreshToken: 'mock-totp-refresh',
          user: {
            id: 'user-2',
            email: 'totp@medilink.in',
            role: 'physician',
            firstName: 'Dr. Priya',
            lastName: 'Sharma',
            totpEnabled: true,
          },
        })
      }
      return HttpResponse.json({ error: 'Invalid TOTP code' }, { status: 401 })
    }),

    http.post(`${base}/auth/refresh`, () => {
      return HttpResponse.json({
        accessToken: 'new-access-token',
        refreshToken: 'new-refresh-token',
      })
    }),

    // Consent — matches consentAPI.getMyPatients() → GET /consent/my-patients
    http.get(`${base}/consent/my-patients`, () => {
      return HttpResponse.json({ patients: MOCK_PATIENTS })
    }),

    // FHIR Patient — matches fhirAPI.getPatient() → GET /fhir/R4/Patient/:id
    http.get(`${base}/fhir/R4/Patient/:id`, ({ params }) => {
      const patient = MOCK_FHIR_PATIENTS[params.id as string]
      if (patient) return HttpResponse.json(patient)
      return HttpResponse.json({
        ...MOCK_FHIR_PATIENTS.p1,
        id: params.id,
      })
    }),

    // Patient search — matches fhirAPI.searchPatients()
    http.get(`${base}/fhir/R4/Patient`, () => {
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: Object.keys(MOCK_FHIR_PATIENTS).length,
        entry: Object.values(MOCK_FHIR_PATIENTS).map((r) => ({ resource: r })),
      })
    }),

    // Timeline — matches fhirAPI.getTimeline() → GET /fhir/R4/Patient/:id/$timeline
    http.get(`${base}/fhir/R4/Patient/:patientId/\\$timeline`, ({ params }) => {
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: 3,
        entry: [
          {
            resource: {
              resourceType: 'Encounter',
              id: 'enc-1',
              status: 'finished',
              class: { code: 'AMB', display: 'Ambulatory' },
              period: { start: '2024-06-10T09:00:00Z', end: '2024-06-10T09:30:00Z' },
              type: [{ coding: [{ display: 'Follow-up visit' }], text: 'Follow-up visit' }],
            },
          },
          {
            resource: {
              resourceType: 'Condition',
              id: 'cond-1',
              clinicalStatus: { coding: [{ code: 'active' }] },
              code: { coding: [{ system: 'http://snomed.info/sct', code: '73211009', display: 'Diabetes mellitus' }], text: 'Type 2 Diabetes' },
              onsetDateTime: '2022-03-15T00:00:00Z',
            },
          },
          {
            resource: {
              resourceType: 'MedicationRequest',
              id: 'medr-1',
              status: 'active',
              intent: 'order',
              medicationCodeableConcept: { coding: [{ display: 'Metformin 500mg' }], text: 'Metformin 500mg' },
              authoredOn: '2024-06-10T09:15:00Z',
            },
          },
        ],
      })
    }),

    // Lab trends — matches fhirAPI.getLabTrends() → GET /fhir/R4/Observation/$lab-trends
    http.get(`${base}/fhir/R4/Observation/\\$lab-trends`, () => {
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: 4,
        entry: [
          {
            resource: {
              resourceType: 'Observation', id: 'obs-1', status: 'final',
              code: { coding: [{ system: 'http://loinc.org', code: '4548-4', display: 'HbA1c' }] },
              valueQuantity: { value: 8.1, unit: '%' },
              effectiveDateTime: '2024-01-15T10:00:00Z',
              referenceRange: [{ low: { value: 4.0 }, high: { value: 5.6 } }],
            },
          },
          {
            resource: {
              resourceType: 'Observation', id: 'obs-2', status: 'final',
              code: { coding: [{ system: 'http://loinc.org', code: '4548-4', display: 'HbA1c' }] },
              valueQuantity: { value: 7.5, unit: '%' },
              effectiveDateTime: '2024-04-15T10:00:00Z',
              referenceRange: [{ low: { value: 4.0 }, high: { value: 5.6 } }],
            },
          },
          {
            resource: {
              resourceType: 'Observation', id: 'obs-3', status: 'final',
              code: { coding: [{ system: 'http://loinc.org', code: '4548-4', display: 'HbA1c' }] },
              valueQuantity: { value: 7.0, unit: '%' },
              effectiveDateTime: '2024-07-15T10:00:00Z',
              referenceRange: [{ low: { value: 4.0 }, high: { value: 5.6 } }],
            },
          },
          {
            resource: {
              resourceType: 'Observation', id: 'obs-4', status: 'final',
              code: { coding: [{ system: 'http://loinc.org', code: '4548-4', display: 'HbA1c' }] },
              valueQuantity: { value: 6.8, unit: '%' },
              effectiveDateTime: '2024-10-15T10:00:00Z',
              referenceRange: [{ low: { value: 4.0 }, high: { value: 5.6 } }],
            },
          },
        ],
      })
    }),

    // Drug interactions — matches clinicalAPI.checkDrugInteractions()
    http.post(`${base}/clinical/drug-check`, async ({ request }) => {
      const body = await request.json() as { newMedication: { rxnormCode: string } }
      const rxnormCode = body.newMedication?.rxnormCode
      if (rxnormCode === '855332') {
        return HttpResponse.json({
          overallSeverity: 'severe',
          newMedication: { name: 'Warfarin 5mg', rxnormCode: '855332' },
          interactions: [
            {
              existingMedication: { name: 'Aspirin 75mg', rxnormCode: '198464' },
              severity: 'severe',
              description: 'Increased risk of bleeding when combined with antiplatelet agents',
              mechanism: 'Both affect platelet function and coagulation cascade',
              clinicalEffect: 'Major bleeding risk including GI and intracranial hemorrhage',
              management: 'Monitor INR closely. Consider alternative antiplatelet if possible.',
            },
          ],
          allergyConflicts: [],
        })
      }
      if (rxnormCode === '197806') {
        return HttpResponse.json({
          overallSeverity: 'contraindicated',
          newMedication: { name: 'Ibuprofen 400mg', rxnormCode: '197806' },
          interactions: [],
          allergyConflicts: [
            { allergyDisplay: 'Ibuprofen', severity: 'high' },
          ],
        })
      }
      return HttpResponse.json({
        overallSeverity: 'none',
        newMedication: { name: 'Medication', rxnormCode },
        interactions: [],
        allergyConflicts: [],
      })
    }),

    http.post(`${base}/clinical/drug-check/acknowledge`, () => {
      return HttpResponse.json({ acknowledged: true })
    }),

    // Document upload
    http.post(`${base}/documents/upload`, () => {
      return HttpResponse.json({
        id: 'job-' + Date.now(),
        status: 'processing',
        originalFilename: 'blood_test.pdf',
        uploadedAt: new Date().toISOString(),
      })
    }),

    http.get(`${base}/documents/jobs`, () => {
      return HttpResponse.json([
        {
          id: 'job-1',
          status: 'completed',
          originalFilename: 'blood_test_report.pdf',
          uploadedAt: '2024-01-10T08:00:00Z',
          observationsCreated: 12,
          loincMapped: 10,
          fhirReportId: 'report-1',
        },
        {
          id: 'job-2',
          status: 'completed',
          originalFilename: 'lipid_panel.pdf',
          uploadedAt: '2024-03-15T14:30:00Z',
          observationsCreated: 6,
          loincMapped: 6,
          fhirReportId: 'report-2',
        },
      ])
    }),

    // Search
    http.get(`${base}/search`, ({ request }) => {
      const url = new URL(request.url)
      const q = (url.searchParams.get('q') || '').toLowerCase()
      const matchingPatients = Object.values(MOCK_FHIR_PATIENTS).filter((p: any) =>
        JSON.stringify(p).toLowerCase().includes(q)
      )
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: matchingPatients.length,
        entry: matchingPatients.map((r) => ({ resource: r })),
      })
    }),

    // Admin
    http.get(`${base}/admin/stats`, () => {
      return HttpResponse.json({
        totalUsers: 15,
        totalPatients: 120,
        totalObservations: 3500,
        totalDocuments: 89,
        activeConsents: 98,
        systemUptime: '99.9%',
      })
    }),

    http.get(`${base}/admin/users`, () => {
      return HttpResponse.json([
        { id: 'u1', email: 'admin@medilink.in', role: 'admin', firstName: 'Admin', lastName: 'User', isActive: true, createdAt: '2024-01-01T00:00:00Z', lastLoginAt: '2024-06-15T08:00:00Z' },
        { id: 'u2', email: 'doctor@medilink.in', role: 'physician', firstName: 'Dr. Arjun', lastName: 'Mehta', isActive: true, createdAt: '2024-01-05T00:00:00Z', lastLoginAt: '2024-06-15T09:30:00Z' },
        { id: 'u3', email: 'priya@medilink.in', role: 'physician', firstName: 'Dr. Priya', lastName: 'Sharma', isActive: true, createdAt: '2024-02-10T00:00:00Z', lastLoginAt: '2024-06-14T16:00:00Z' },
        { id: 'u4', email: 'nurse@medilink.in', role: 'nurse', firstName: 'Sunita', lastName: 'Rao', isActive: false, createdAt: '2024-03-01T00:00:00Z' },
      ])
    }),

    http.get(`${base}/admin/audit-logs`, () => {
      return HttpResponse.json({
        logs: [
          { id: 'log-1', userId: 'u1', userEmail: 'admin@medilink.in', action: 'create_user', resourceType: 'User', resourceId: 'u3', details: 'Created physician account for Dr. Priya', ipAddress: '10.0.0.1', createdAt: '2024-06-15T10:00:00Z' },
          { id: 'log-2', userId: 'u2', userEmail: 'doctor@medilink.in', action: 'read_patient', resourceType: 'Patient', resourceId: 'p1', details: 'Viewed patient record', ipAddress: '10.0.0.5', createdAt: '2024-06-15T09:45:00Z' },
          { id: 'log-3', userId: 'u2', userEmail: 'doctor@medilink.in', action: 'create_prescription', resourceType: 'MedicationRequest', resourceId: 'medr-1', details: 'Prescribed Metformin 500mg', ipAddress: '10.0.0.5', createdAt: '2024-06-15T09:30:00Z' },
        ],
        total: 3,
      })
    }),

    // FHIR resource creation (for prescriptions)
    http.post(`${base}/fhir/R4/:resourceType`, () => {
      return HttpResponse.json({ id: 'new-resource-1', resourceType: 'MedicationRequest' })
    }),

    // FHIR resource search (AllergyIntolerance, etc.)
    http.get(`${base}/fhir/R4/AllergyIntolerance`, () => {
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: 1,
        entry: [
          {
            resource: {
              resourceType: 'AllergyIntolerance',
              id: 'allergy-1',
              code: { coding: [{ system: 'http://snomed.info/sct', code: '387207008', display: 'Ibuprofen' }], text: 'Ibuprofen' },
              criticality: 'high',
            },
          },
        ],
      })
    }),

    // Generic FHIR resource search fallback
    http.get(`${base}/fhir/R4/:resourceType`, () => {
      return HttpResponse.json({
        resourceType: 'Bundle',
        type: 'searchset',
        total: 0,
        entry: [],
      })
    }),

    // Notifications preferences
    http.get(`${base}/notifications/preferences`, () => {
      return HttpResponse.json({
        drugInteractions: true,
        labResults: true,
        documents: true,
        consents: true,
        system: false,
      })
    }),
  ]
}

export const handlers = [
  ...createHandlers(API_BASE),
  ...createHandlers('/api'),
]
