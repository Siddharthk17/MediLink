package consent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/consent"
)

// Mock: ConsentRepository

type mockConsentRepo struct {
	mu                sync.Mutex
	consents          map[uuid.UUID]*consent.Consent
	patientFHIRIDs    map[uuid.UUID]string
	fhirToUserID      map[string]uuid.UUID
	knownProviders    map[uuid.UUID]bool
	breakGlassCount   map[uuid.UUID]int
	breakGlassLimit   int
	checkConsentCalls int
}

func newMockRepo() *mockConsentRepo {
	return &mockConsentRepo{
		consents:        make(map[uuid.UUID]*consent.Consent),
		patientFHIRIDs:  make(map[uuid.UUID]string),
		fhirToUserID:    make(map[string]uuid.UUID),
		knownProviders:  make(map[uuid.UUID]bool),
		breakGlassCount: make(map[uuid.UUID]int),
		breakGlassLimit: 3,
	}
}

func (m *mockConsentRepo) addPatient(userID uuid.UUID, fhirID string) {
	m.patientFHIRIDs[userID] = fhirID
	m.fhirToUserID[fhirID] = userID
}

func (m *mockConsentRepo) Create(ctx context.Context, consent *consent.Consent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.knownProviders[consent.ProviderID] {
		return fmt.Errorf("provider not found: no user with id %s", consent.ProviderID)
	}

	if consent.Purpose == "break-glass" {
		if m.breakGlassCount[consent.ProviderID] >= m.breakGlassLimit {
			return fmt.Errorf("break-glass rate limit exceeded: maximum %d emergency accesses per 24 hours", m.breakGlassLimit)
		}
		m.breakGlassCount[consent.ProviderID]++
	}

	consent.ID = uuid.New()
	consent.GrantedAt = time.Now()
	stored := *consent
	m.consents[consent.ID] = &stored
	return nil
}

func (m *mockConsentRepo) GetByID(ctx context.Context, id uuid.UUID) (*consent.Consent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.consents[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *mockConsentRepo) Update(ctx context.Context, consent *consent.Consent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *consent
	m.consents[consent.ID] = &stored
	return nil
}

func (m *mockConsentRepo) GetActiveConsent(ctx context.Context, patientID, providerID uuid.UUID) (*consent.Consent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.consents {
		if c.PatientID == patientID && c.ProviderID == providerID && c.RevokedAt == nil {
			if c.ExpiresAt == nil || c.ExpiresAt.After(time.Now()) {
				cp := *c
				return &cp, nil
			}
		}
	}
	return nil, nil
}

func (m *mockConsentRepo) RevokeConsent(ctx context.Context, consentID uuid.UUID, revokedBy uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.consents[consentID]
	if !ok {
		return fmt.Errorf("consent not found")
	}
	now := time.Now()
	c.RevokedAt = &now
	c.RevokedBy = &revokedBy
	return nil
}

func (m *mockConsentRepo) GetPatientConsents(ctx context.Context, patientID uuid.UUID) ([]*consent.Consent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*consent.Consent
	for _, c := range m.consents {
		if c.PatientID == patientID {
			cp := *c
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockConsentRepo) GetPhysicianPatients(ctx context.Context, providerID uuid.UUID) ([]*consent.ConsentedPatient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*consent.ConsentedPatient
	for _, c := range m.consents {
		if c.ProviderID == providerID && c.RevokedAt == nil {
			if c.ExpiresAt == nil || c.ExpiresAt.After(time.Now()) {
				fhirID := m.patientFHIRIDs[c.PatientID]
				result = append(result, &consent.ConsentedPatient{
					PatientID:     c.PatientID,
					FHIRPatientID: fhirID,
					Scope:         c.Scope,
					GrantedAt:     c.GrantedAt,
					ConsentID:     c.ID,
				})
			}
		}
	}
	return result, nil
}

func (m *mockConsentRepo) GetPendingRequests(ctx context.Context, providerID uuid.UUID) ([]*consent.ConsentedPatient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*consent.ConsentedPatient
	for _, c := range m.consents {
		if c.ProviderID == providerID && c.Status == "pending" && c.RevokedAt == nil {
			fhirID := m.patientFHIRIDs[c.PatientID]
			result = append(result, &consent.ConsentedPatient{
				PatientID:     c.PatientID,
				FHIRPatientID: fhirID,
				Scope:         c.Scope,
				GrantedAt:     c.GrantedAt,
				ConsentID:     c.ID,
				Status:        c.Status,
			})
		}
	}
	return result, nil
}

func (m *mockConsentRepo) UpdateStatus(ctx context.Context, consentID uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.consents[consentID]
	if !ok {
		return fmt.Errorf("consent not found")
	}
	c.Status = status
	return nil
}

func (m *mockConsentRepo) CheckConsent(ctx context.Context, providerID uuid.UUID, patientFHIRID string) (bool, []string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkConsentCalls++

	patientUserID, ok := m.fhirToUserID[patientFHIRID]
	if !ok {
		return false, nil, nil
	}

	for _, c := range m.consents {
		if c.ProviderID == providerID && c.PatientID == patientUserID && c.RevokedAt == nil {
			if c.ExpiresAt == nil || c.ExpiresAt.After(time.Now()) {
				var scope []string
				if err := json.Unmarshal(c.Scope, &scope); err != nil {
					return false, nil, err
				}
				return true, scope, nil
			}
		}
	}
	return false, nil, nil
}

func (m *mockConsentRepo) GetPatientFHIRID(ctx context.Context, userID uuid.UUID) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fhirID, ok := m.patientFHIRIDs[userID]
	if !ok {
		return "", fmt.Errorf("patient not found")
	}
	return fhirID, nil
}

func (m *mockConsentRepo) GetUserIDByFHIRPatientID(ctx context.Context, fhirPatientID string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	userID, ok := m.fhirToUserID[fhirPatientID]
	if !ok {
		return uuid.Nil, fmt.Errorf("patient not found for FHIR ID %s", fhirPatientID)
	}
	return userID, nil
}

func (m *mockConsentRepo) getCheckConsentCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkConsentCalls
}

// Mock: AuditLogger

type mockAuditLogger struct {
	mu   sync.Mutex
	logs []audit.AuditEntry
}

func (m *mockAuditLogger) Log(_ context.Context, entry audit.AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, entry)
	return nil
}

func (m *mockAuditLogger) LogAsync(entry audit.AuditEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, entry)
}

func (m *mockAuditLogger) Close() {}

func (m *mockAuditLogger) entries() []audit.AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]audit.AuditEntry, len(m.logs))
	copy(out, m.logs)
	return out
}

// Helpers

func newTestEngine(repo *mockConsentRepo, cache *consent.ConsentCache) (consent.ConsentEngine, *mockAuditLogger) {
	al := &mockAuditLogger{}
	engine := consent.NewConsentEngine(repo, cache, al, zerolog.Nop(), nil, nil, nil)
	return engine, al
}

func setupCache(t *testing.T) *consent.ConsentCache {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return consent.NewConsentCache(rdb)
}

// 1. TestGrantConsent_201

func TestGrantConsent_201(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, auditLog := newTestEngine(repo, nil)
	ctx := context.Background()

	req := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"Observation", "MedicationRequest"},
		Purpose:        "treatment",
		Notes:          "routine care",
	}

	consent, err := engine.GrantConsent(ctx, req, patientID.String())
	require.NoError(t, err)
	require.NotNil(t, consent)

	assert.Equal(t, patientID, consent.PatientID)
	assert.Equal(t, providerID, consent.ProviderID)
	assert.Equal(t, "treatment", consent.Purpose)
	assert.NotEqual(t, uuid.Nil, consent.ID)

	var scope []string
	require.NoError(t, json.Unmarshal(consent.Scope, &scope))
	assert.ElementsMatch(t, []string{"Observation", "MedicationRequest"}, scope)

	entries := auditLog.entries()
	require.Len(t, entries, 1)
	assert.Equal(t, "request", entries[0].Action)
	assert.Equal(t, 201, entries[0].StatusCode)
	assert.True(t, entries[0].Success)
}

// 2. TestGrantConsent_400_SelfGrant

func TestGrantConsent_400_SelfGrant(t *testing.T) {
	repo := newMockRepo()
	userID := uuid.New()
	repo.knownProviders[userID] = true

	engine, _ := newTestEngine(repo, nil)

	req := consent.GrantConsentRequest{
		ProviderUserID: userID.String(),
		Scope:          []string{"*"},
	}

	consent, err := engine.GrantConsent(context.Background(), req, userID.String())
	assert.Nil(t, consent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot grant consent to yourself")
}

// 3. TestGrantConsent_404_PhysicianNotFound

func TestGrantConsent_404_PhysicianNotFound(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	unknownProvider := uuid.New()
	// Provider is intentionally NOT registered in knownProviders.

	engine, _ := newTestEngine(repo, nil)

	req := consent.GrantConsentRequest{
		ProviderUserID: unknownProvider.String(),
		Scope:          []string{"*"},
	}

	consent, err := engine.GrantConsent(context.Background(), req, patientID.String())
	assert.Nil(t, consent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// 4. TestRevokeConsent_200

func TestRevokeConsent_200(t *testing.T) {
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, auditLog := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant consent first.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}
	consent, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)
	require.NotNil(t, consent)

	// Populate the cache via a check.
	_, err = engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)

	// Revoke as the owning patient.
	err = engine.RevokeConsent(ctx, consent.ID.String(), patientID.String(), "patient")
	require.NoError(t, err)

	// Verify revoked in repo.
	revoked, err := repo.GetByID(ctx, consent.ID)
	require.NoError(t, err)
	require.NotNil(t, revoked)
	assert.NotNil(t, revoked.RevokedAt)

	// Verify audit trail contains the revoke action.
	entries := auditLog.entries()
	var found bool
	for _, e := range entries {
		if e.Action == "revoke" && e.StatusCode == 200 {
			found = true
			break
		}
	}
	assert.True(t, found, "audit log should contain a revoke entry with status 200")
}

// 5. TestRevokeConsent_403_WrongPatient

func TestRevokeConsent_403_WrongPatient(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	providerID := uuid.New()
	otherUser := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, nil)
	ctx := context.Background()

	// Grant consent as the patient.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}
	consent, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	// Attempt revoke as a different non-admin user.
	err = engine.RevokeConsent(ctx, consent.ID.String(), otherUser.String(), "patient")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only the patient")
}

// 6. TestCheckConsent_CacheHit

func TestCheckConsent_CacheHit(t *testing.T) {
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant consent.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}
	_, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	// First check — cache miss, hits repo.
	granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.True(t, granted)
	callsAfterFirst := repo.getCheckConsentCalls()
	assert.Equal(t, 1, callsAfterFirst)

	// Second check — should serve from cache (no additional repo call).
	granted, err = engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.True(t, granted)
	assert.Equal(t, callsAfterFirst, repo.getCheckConsentCalls(),
		"second CheckConsent should use cache, not call repo again")
}

// 7. TestCheckConsent_CacheMiss

func TestCheckConsent_CacheMiss(t *testing.T) {
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant consent.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"Observation", "Condition"},
	}
	_, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	// No checks yet — repo call count should be 0.
	assert.Equal(t, 0, repo.getCheckConsentCalls())

	// First check — cache miss, must query repo.
	granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.True(t, granted)
	assert.Equal(t, 1, repo.getCheckConsentCalls(), "cache miss should query the repo")

	// A second call should now hit cache and NOT increment repo calls.
	granted, err = engine.CheckConsent(ctx, providerID.String(), fhirID, "Condition")
	require.NoError(t, err)
	assert.True(t, granted)
	assert.Equal(t, 1, repo.getCheckConsentCalls(), "second call should hit cache, not repo")
}

// 8. TestCheckConsent_AfterRevoke

func TestCheckConsent_AfterRevoke(t *testing.T) {
	repo := newMockRepo()
	cache := setupCache(t)
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, cache)
	ctx := context.Background()

	// Grant consent.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}
	consent, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	// Check — should be granted.
	granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.True(t, granted)

	// Revoke — also invalidates cache.
	err = engine.RevokeConsent(ctx, consent.ID.String(), patientID.String(), "patient")
	require.NoError(t, err)

	// Check again — consent should now be denied.
	granted, err = engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
	require.NoError(t, err)
	assert.False(t, granted, "consent check should return false after revocation")
}

// 9. TestBreakGlass_201

func TestBreakGlass_201(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	physicianID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[physicianID] = true
	repo.addPatient(patientID, fhirID)

	engine, auditLog := newTestEngine(repo, nil)
	ctx := context.Background()

	req := consent.BreakGlassRequest{
		PatientFHIRID:   fhirID,
		Reason:          "Patient arrived unconscious in ER, need immediate access to medical history",
		ClinicalContext: "emergency",
	}

	consent, err := engine.RecordBreakGlass(ctx, req, physicianID.String())
	require.NoError(t, err)
	require.NotNil(t, consent)

	assert.Equal(t, patientID, consent.PatientID)
	assert.Equal(t, physicianID, consent.ProviderID)
	assert.Equal(t, "break-glass", consent.Purpose)
	assert.NotNil(t, consent.ExpiresAt)
	assert.True(t, consent.ExpiresAt.After(time.Now()))
	assert.True(t, consent.ExpiresAt.Before(time.Now().Add(25*time.Hour)))

	var scope []string
	require.NoError(t, json.Unmarshal(consent.Scope, &scope))
	assert.Equal(t, []string{"*"}, scope)

	// Verify audit entry.
	entries := auditLog.entries()
	var bgEntry *audit.AuditEntry
	for i := range entries {
		if entries[i].Action == "break-glass" {
			bgEntry = &entries[i]
			break
		}
	}
	require.NotNil(t, bgEntry)
	assert.Equal(t, 201, bgEntry.StatusCode)
	assert.Contains(t, bgEntry.PatientRef, fhirID)
}

// 10. TestBreakGlass_400_ShortReason

func TestBreakGlass_400_ShortReason(t *testing.T) {
	repo := newMockRepo()
	physicianID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[physicianID] = true
	repo.addPatient(uuid.New(), fhirID)

	engine, _ := newTestEngine(repo, nil)

	req := consent.BreakGlassRequest{
		PatientFHIRID: fhirID,
		Reason:        "short reason", // less than 20 characters
	}

	consent, err := engine.RecordBreakGlass(context.Background(), req, physicianID.String())
	assert.Nil(t, consent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 20 characters")
}

// 11. TestBreakGlass_429_RateLimited

func TestBreakGlass_429_RateLimited(t *testing.T) {
	repo := newMockRepo()
	repo.breakGlassLimit = 100 // disable repo-level limiting; test Redis rate limit
	physicianID := uuid.New()
	repo.knownProviders[physicianID] = true

	// Create 4 distinct patients.
	fhirIDs := make([]string, 4)
	for i := 0; i < 4; i++ {
		pid := uuid.New()
		fhirIDs[i] = uuid.New().String()
		repo.addPatient(pid, fhirIDs[i])
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	al := &mockAuditLogger{}
	engine := consent.NewConsentEngine(repo, nil, al, zerolog.Nop(), nil, nil, rdb)
	ctx := context.Background()
	validReason := "Patient is unconscious and requires emergency treatment access immediately"

	// First 3 break-glass requests should succeed.
	for i := 0; i < 3; i++ {
		req := consent.BreakGlassRequest{
			PatientFHIRID: fhirIDs[i],
			Reason:        validReason,
		}
		consent, err := engine.RecordBreakGlass(ctx, req, physicianID.String())
		require.NoError(t, err, "break-glass %d should succeed", i+1)
		require.NotNil(t, consent)
	}

	// 4th should be rate-limited.
	req := consent.BreakGlassRequest{
		PatientFHIRID: fhirIDs[3],
		Reason:        validReason,
	}
	consent, err := engine.RecordBreakGlass(ctx, req, physicianID.String())
	assert.Nil(t, consent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

// 12. TestConsentScope_FullAccess

func TestConsentScope_FullAccess(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, nil)
	ctx := context.Background()

	// Grant with wildcard scope.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"*"},
	}
	_, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	// Every patient-scoped resource type should be allowed.
	allTypes := []string{
		"Patient", "Observation", "DiagnosticReport",
		"MedicationRequest", "Condition", "Encounter",
		"AllergyIntolerance", "Immunization",
	}
	for _, rt := range allTypes {
		t.Run(rt, func(t *testing.T) {
			granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, rt)
			require.NoError(t, err)
			assert.True(t, granted, "scope [*] should allow %s", rt)
		})
	}
}

// 13. TestConsentScope_Limited

func TestConsentScope_Limited(t *testing.T) {
	repo := newMockRepo()
	patientID := uuid.New()
	providerID := uuid.New()
	fhirID := uuid.New().String()

	repo.knownProviders[providerID] = true
	repo.addPatient(patientID, fhirID)

	engine, _ := newTestEngine(repo, nil)
	ctx := context.Background()

	// Grant with a restricted scope — Observation only.
	grantReq := consent.GrantConsentRequest{
		ProviderUserID: providerID.String(),
		Scope:          []string{"Observation"},
	}
	_, err := engine.GrantConsent(ctx, grantReq, patientID.String())
	require.NoError(t, err)

	t.Run("Observation_Allowed", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Observation")
		require.NoError(t, err)
		assert.True(t, granted, "Observation should be allowed with scope [Observation]")
	})

	t.Run("MedicationRequest_Blocked", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "MedicationRequest")
		require.NoError(t, err)
		assert.False(t, granted, "MedicationRequest should be blocked with scope [Observation]")
	})

	t.Run("Patient_AlwaysAllowed", func(t *testing.T) {
		granted, err := engine.CheckConsent(ctx, providerID.String(), fhirID, "Patient")
		require.NoError(t, err)
		assert.True(t, granted, "Patient resource should always be allowed with any active consent")
	})
}
