# 🔍 COMPREHENSIVE CROSS-REFERENCE AUDIT REPORT
# Backend Routes vs Frontend API Calls - MediLink

**Audit Date**: 2024  
**Status**: ⚠️ CRITICAL ISSUES FOUND  
**Backend File**: `/backend/cmd/api/main.go` (967 lines)  
**Frontend API Files**: 
- ✅ `/frontend/shared/src/api/auth.ts`
- ✅ `/frontend/shared/src/api/consent.ts`
- ✅ `/frontend/shared/src/api/fhir.ts`
- ✅ `/frontend/shared/src/api/clinical.ts`
- ✅ `/frontend/shared/src/api/notifications.ts`
- ✅ `/frontend/shared/src/api/search.ts`
- ✅ `/frontend/shared/src/api/client.ts`
- ❌ `/frontend/shared/src/api/admin.ts` (MISSING)
- ❌ `/frontend/shared/src/api/documents.ts` (MISSING)

---

## 🔴 CRITICAL ISSUES (BREAKING)

### 1. Drug Check API - Request Type Mismatch (CRITICAL)

**Location**: 
- Backend: `internal/clinical/types.go` + `internal/clinical/handlers.go`
- Frontend: `frontend/shared/src/api/clinical.ts`

**Issue**: Frontend sends wrong patient ID format

```
Backend expects:  patientId: "abc123" (plain FHIR ID)
Frontend sends:   patientId: "Patient/abc123" (full reference)
Result: Handler tries to create "Patient/Patient/abc123" - BROKEN
```

**Fix Required**:
```typescript
// BEFORE (WRONG):
checkDrugInteractions: (patientId: string, rxnormCode: string) =>
  apiClient.post<DrugCheckResult>('/clinical/drug-check', {
    patientId: `Patient/${patientId}`,  // ❌ WRONG
    newMedication: { rxnormCode },
  }),

// AFTER (CORRECT):
checkDrugInteractions: (patientId: string, rxnormCode: string) =>
  apiClient.post<DrugCheckResult>('/clinical/drug-check', {
    patientId: patientId,  // ✅ Just the ID
    newMedication: { rxnormCode },
  }),
```

---

### 2. Drug Check API - Response Type Mismatch (CRITICAL)

**Backend sends**:
```go
{
  "newMedication": { "rxnormCode": "...", "name": "..." },
  "interactions": [
    {
      "drugA": { "rxnormCode": "...", "name": "..." },
      "drugB": { "rxnormCode": "...", "name": "..." },  // ← Existing med
      "severity": "major",
      "description": "...",
      "mechanism": "...",
      "clinicalEffect": "...",
      "management": "...",
      "source": "openfda",
      "cached": false
    }
  ],
  "allergyConflicts": [
    {
      "allergen": { "rxnormCode": "...", "name": "..." },
      "newMedication": { "rxnormCode": "...", "name": "..." },
      "severity": "major",
      "mechanism": "...",
      "drugClass": "...",
      "reaction": "..."
    }
  ],
  "highestSeverity": "major",
  "hasContraindication": false,
  "checkComplete": true
}
```

**Frontend expects**:
```typescript
{
  "newMedication": { "rxnormCode": "...", "name": "..." },
  "interactions": [
    {
      "existingMedication": { "rxnormCode": "...", "name": "..." },  // ❌ Field name wrong
      "severity": "major",
      "description": "...",
      "mechanism": "...",
      "clinicalEffect": "...",
      "management": "..."
      // ❌ Missing: drugA, source, cached
    }
  ],
  "allergyConflicts": [
    {
      "allergyCode": "...",  // ❌ Wrong: should be allergen.rxnormCode
      "allergyDisplay": "...",  // ❌ Wrong: should be allergen.name
      "severity": "major",
      "crossReactive": false  // ❌ Wrong: backend has mechanism, not this
      // ❌ Missing: allergen, newMedication, drugClass, reaction
    }
  ],
  "overallSeverity": "major",  // ❌ Wrong field name: backend is "highestSeverity"
  "checkComplete": true
  // ❌ Missing: hasContraindication, checkError
}
```

**Mismatches**:
| Backend Field | Frontend Field | Status |
|---|---|---|
| `interactions[].drugA` | ❌ Missing | WRONG |
| `interactions[].drugB` | `existingMedication` | NAME MISMATCH |
| `interactions[].source` | ❌ Missing | WRONG |
| `interactions[].cached` | ❌ Missing | WRONG |
| `allergyConflicts[].allergen` | `allergyCode` + `allergyDisplay` | STRUCTURE MISMATCH |
| `allergyConflicts[].newMedication` | ❌ Missing | WRONG |
| `allergyConflicts[].drugClass` | ❌ Missing | WRONG |
| `allergyConflicts[].reaction` | ❌ Missing | WRONG |
| `allergyConflicts[].mechanism` | `crossReactive` | NAME MISMATCH |
| `highestSeverity` | `overallSeverity` | NAME MISMATCH |
| `hasContraindication` | ❌ Missing | WRONG |
| `checkError` | ❌ Missing | WRONG |

**Fix Required**: Update TypeScript types in `frontend/shared/src/types/fhir.ts`:
```typescript
export interface DrugCheckResult {
  newMedication: { rxnormCode: string; name: string }
  interactions: Array<{
    drugA: { rxnormCode: string; name: string }  // ✅ ADD
    drugB: { rxnormCode: string; name: string }  // ✅ Changed from existingMedication
    severity: InteractionSeverity
    description: string
    mechanism?: string
    clinicalEffect?: string
    management?: string
    source: string  // ✅ ADD
    cached: boolean  // ✅ ADD
  }>
  allergyConflicts: Array<{
    allergen: { rxnormCode: string; name: string }  // ✅ CHANGE from allergyCode/allergyDisplay
    newMedication: { rxnormCode: string; name: string }  // ✅ ADD
    severity: InteractionSeverity
    mechanism: string  // ✅ CHANGE from crossReactive
    drugClass?: string  // ✅ ADD
    reaction?: string  // ✅ ADD
  }>
  highestSeverity: InteractionSeverity  // ✅ CHANGE from overallSeverity
  hasContraindication: boolean  // ✅ ADD
  checkComplete: boolean
  checkError?: string  // ✅ ADD
}
```

---

### 3. Drug Check Acknowledge - Request Type Mismatch (CRITICAL)

**Location**: `frontend/shared/src/api/clinical.ts` line 11-16

**Issue**: Multiple field mismatches

```typescript
// CURRENT (WRONG):
acknowledgeDrugInteraction: (patientId: string, rxnormCode: string, reason: string) =>
  apiClient.post('/clinical/drug-check/acknowledge', {
    patientId: `Patient/${patientId}`,  // ❌ Should be just ID, not "Patient/{id}"
    rxnormCode,  // ❌ Backend expects: newMedication
    reason,  // ❌ Backend expects: reason (OK), but...
    // ❌ Missing: conflictingMedications array
  }),

// Backend expects:
{
  "patientId": "abc123",  // Just FHIR ID
  "newMedication": "rxnorm123",  // ← Wrong field name in frontend
  "conflictingMedications": ["rxnorm456", "rxnorm789"],  // ← MISSING
  "reason": "Lorem ipsum dolor sit amet consectetur..."  // Min 20 chars
}
```

**Fix Required**:
```typescript
acknowledgeDrugInteraction: (patientId: string, rxnormCode: string, conflictingCodes: string[], reason: string) =>
  apiClient.post('/clinical/drug-check/acknowledge', {
    patientId,  // ✅ Just the ID
    newMedication: rxnormCode,  // ✅ Renamed
    conflictingMedications: conflictingCodes,  // ✅ Added
    reason,  // ✅ Keep (validate min 20 chars on frontend)
  }),
```

---

### 4. Missing Frontend API Modules (CRITICAL)

**Not Found**:
- ❌ `frontend/shared/src/api/admin.ts` (16 backend routes not accessible)
- ❌ `frontend/shared/src/api/documents.ts` (4 backend routes not accessible)
- ❌ `frontend/shared/src/api/research.ts` (4 backend routes not accessible)

**Impact**:
- No admin panel functionality
- No document upload/processing
- No research export capability

**Required Files**:

`frontend/shared/src/api/admin.ts`:
```typescript
import { apiClient } from './client'

export const adminAPI = {
  listUsers: () => apiClient.get('/admin/users'),
  getUser: (userId: string) => apiClient.get(`/admin/users/${userId}`),
  updateUserRole: (userId: string, role: string) => apiClient.put(`/admin/users/${userId}/role`, { role }),
  approvePhysician: (userId: string) => apiClient.post(`/admin/physicians/${userId}/approve`),
  suspendPhysician: (userId: string, reason: string) => apiClient.post(`/admin/physicians/${userId}/suspend`, { reason }),
  reinstatePhysician: (userId: string) => apiClient.post(`/admin/physicians/${userId}/reinstate`),
  inviteResearcher: (email: string) => apiClient.post('/admin/researchers/invite', { email }),
  getAuditLogs: (params?: any) => apiClient.get('/admin/audit-logs', { params }),
  getPatientAuditLog: (patientId: string) => apiClient.get(`/admin/audit-logs/patient/${patientId}`),
  getActorAuditLog: (actorId: string) => apiClient.get(`/admin/audit-logs/actor/${actorId}`),
  getBreakGlassEvents: () => apiClient.get('/admin/audit-logs/break-glass'),
  getStats: () => apiClient.get('/admin/stats'),
  getSystemHealth: () => apiClient.get('/admin/system/health'),
  triggerReindex: () => apiClient.post('/admin/search/reindex'),
  triggerTokenCleanup: () => apiClient.post('/admin/tasks/cleanup-tokens'),
}
```

`frontend/shared/src/api/documents.ts`:
```typescript
import { apiClient } from './client'
import type { DocumentJob } from '../types/api'

export const documentsAPI = {
  uploadDocument: (file: File, patientFhirId: string) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('patientFhirId', patientFhirId)
    return apiClient.post<DocumentJob>('/documents/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  getJobStatus: (jobId: string) => apiClient.get<DocumentJob>(`/documents/jobs/${jobId}`),
  listJobs: () => apiClient.get<{ jobs: DocumentJob[] }>('/documents/jobs'),
  deleteJob: (jobId: string) => apiClient.delete(`/documents/jobs/${jobId}`),
}
```

---

### 5. Auth Endpoints Not Called by Frontend (HIGH)

**Backend Routes** → **Frontend Status**:
- `POST /auth/register/patient` → ❌ NOT CALLED
- `POST /auth/password/change` → ❌ NOT CALLED

**Impact**: Patient registration flow not exposed to frontend; password change not available

---

## ⚠️ HIGH PRIORITY ISSUES

### 6. Consent Routes Not Called

**Backend Routes** → **Frontend Status**:
- `GET /consent/my-grants` → ❌ NOT CALLED
- `GET /consent/:consentId` → ❌ NOT CALLED
- `GET /consent/access-log` → ❌ NOT CALLED

**File**: `frontend/shared/src/api/consent.ts`

**Fix**:
```typescript
export const consentAPI = {
  // ... existing ...
  getMyGrants: () =>
    apiClient.get<{ consents: any[] }>('/consent/my-grants'),

  getConsent: (consentId: string) =>
    apiClient.get(`/consent/${consentId}`),

  getAccessLog: () =>
    apiClient.get('/consent/access-log'),
}
```

---

### 7. Clinical Routes Not Called

**Backend Route** → **Frontend Status**:
- `GET /clinical/drug-check/history/:patientId` → ❌ NOT CALLED

**File**: `frontend/shared/src/api/clinical.ts`

**Fix**:
```typescript
export const clinicalAPI = {
  // ... existing ...
  getDrugCheckHistory: (patientId: string) =>
    apiClient.get(`/clinical/drug-check/history/${patientId}`),
}
```

---

### 8. Notifications Routes Not Called

**Backend Routes** → **Frontend Status**:
- `POST /notifications/fcm-token` → ❌ NOT CALLED
- `DELETE /notifications/fcm-token` → ❌ NOT CALLED

**File**: `frontend/shared/src/api/notifications.ts`

**Fix**:
```typescript
export const notificationsAPI = {
  // ... existing ...
  registerFCMToken: (token: string) =>
    apiClient.post('/notifications/fcm-token', { token }),

  revokeFCMToken: () =>
    apiClient.delete('/notifications/fcm-token'),
}
```

---

## ✅ CORRECT IMPLEMENTATIONS

### Auth Routes (8/10)
- ✅ POST /auth/register/physician
- ✅ POST /auth/login
- ✅ POST /auth/login/verify-totp
- ✅ POST /auth/logout
- ✅ POST /auth/refresh
- ✅ POST /auth/totp/setup
- ✅ POST /auth/totp/verify-setup
- ✅ GET /auth/me

### Consent Routes (3/7)
- ✅ POST /consent/grant
- ✅ DELETE /consent/:consentId/revoke
- ✅ GET /consent/my-patients
- ✅ POST /consent/break-glass

### FHIR Routes (8/50+)
- ✅ GET /fhir/R4/Patient
- ✅ GET /fhir/R4/Patient/:id
- ✅ GET /fhir/R4/Patient/:id/$timeline
- ✅ GET /fhir/R4/Observation/$lab-trends
- ✅ POST /fhir/R4/:resourceType (generic)
- ✅ GET /fhir/R4/:resourceType (search)
- ✅ GET /fhir/R4/:resourceType/:id

### Notifications Routes (2/4)
- ✅ GET /notifications/preferences
- ✅ PUT /notifications/preferences

### Other Routes (2/2)
- ✅ GET /search (unified search)

---

## 📊 AUDIT STATISTICS

| Category | Total Backend Routes | Frontend Calls | Match % |
|---|---|---|---|
| Auth | 10 | 8 | 80% |
| Consent | 7 | 4 | 57% |
| Clinical | 3 | 2 | 67% |
| FHIR | 50+ | 7 | 14% |
| Documents | 4 | 0 | 0% ❌ |
| Admin | 16 | 0 | 0% ❌ |
| Notifications | 4 | 2 | 50% |
| Research | 4 | 0 | 0% ❌ |
| Search | 1 | 1 | 100% |
| **TOTAL** | **~99** | **~24** | **~24%** ⚠️ |

---

## 🔧 RECOMMENDED FIXES (Priority Order)

### Priority 1: CRITICAL (Fix Now)
1. ✅ Fix drug check request: Send plain ID, not "Patient/{id}"
2. ✅ Fix drug check response: Align all field names with backend
3. ✅ Fix acknowledge request: Add conflictingMedications array
4. ✅ Create admin.ts module
5. ✅ Create documents.ts module

### Priority 2: HIGH (Fix This Sprint)
6. ✅ Add `/auth/register/patient` call to frontend
7. ✅ Add `/auth/password/change` call to frontend
8. ✅ Add missing consent routes (my-grants, get, access-log)
9. ✅ Add `/clinical/drug-check/history` call
10. ✅ Add FCM token routes

### Priority 3: MEDIUM (Fix Next Sprint)
11. ✅ Add FHIR history/versioning routes
12. ✅ Add research export routes
13. ✅ Implement direct Patient CRUD (currently uses generic)

---

## FILES AFFECTED

### Backend (Read-Only)
- `backend/cmd/api/main.go` - Routes defined (967 lines)
- `backend/internal/clinical/types.go` - Drug check types
- `backend/internal/clinical/handlers.go` - Drug check handlers
- `backend/internal/consent/engine.go` - Consent types
- `backend/internal/auth/handlers.go` - Auth types

### Frontend (Needs Changes)
- `frontend/shared/src/api/clinical.ts` - 🔴 CRITICAL FIXES NEEDED
- `frontend/shared/src/api/consent.ts` - ⚠️ Add missing routes
- `frontend/shared/src/api/auth.ts` - ⚠️ Add missing routes
- `frontend/shared/src/api/notifications.ts` - ⚠️ Add missing routes
- `frontend/shared/src/types/fhir.ts` - 🔴 CRITICAL TYPE FIXES NEEDED
- `frontend/shared/src/types/api.ts` - ✅ Generally OK
- `frontend/shared/src/api/admin.ts` - ❌ NEEDS CREATION
- `frontend/shared/src/api/documents.ts` - ❌ NEEDS CREATION
- `frontend/physician-dashboard/next.config.ts` - ✅ CORRECT

---

## VERIFICATION CHECKLIST

- [ ] Drug check request/response types fixed
- [ ] admin.ts created with 16 endpoints
- [ ] documents.ts created with 4 endpoints
- [ ] research.ts created (if needed) with 4 endpoints
- [ ] All missing routes added to respective modules
- [ ] TypeScript types aligned with backend Go structs
- [ ] Request/response validation updated
- [ ] Frontend tests updated
- [ ] Backend tests updated
- [ ] API documentation updated

---

**Audit Completed**: Cross-reference verification of 99+ backend routes against frontend API calls  
**Critical Issues Found**: 5  
**High Priority Issues**: 8  
**Missing Modules**: 3  
**Recommendation**: Address Priority 1 issues immediately before deployment
