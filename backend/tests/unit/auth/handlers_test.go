package auth_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/notifications"

	"github.com/Siddharthk17/MediLink/internal/auth"

	"crypto/sha256"

	"encoding/hex"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Fake SQL driver (avoids external mock-db dependency)
// ---------------------------------------------------------------------------

var registerDriverOnce sync.Once

func newFakeDB() *sqlx.DB {
	registerDriverOnce.Do(func() { sql.Register("fakedb", &fakeDBDriver{}) })
	db, _ := sql.Open("fakedb", "test")
	return sqlx.NewDb(db, "fakedb")
}

type fakeDBDriver struct{}

func (d *fakeDBDriver) Open(string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return &fakeStmt{}, nil }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)            { return &fakeTx{}, nil }

type fakeTx struct{}

func (t *fakeTx) Commit() error   { return nil }
func (t *fakeTx) Rollback() error { return nil }

type fakeStmt struct{}

func (s *fakeStmt) Close() error                               { return nil }
func (s *fakeStmt) NumInput() int                              { return -1 }
func (s *fakeStmt) Exec([]driver.Value) (driver.Result, error) { return fakeResult{}, nil }
func (s *fakeStmt) Query([]driver.Value) (driver.Rows, error)  { return &fakeRows{}, nil }

type fakeResult struct{}

func (r fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) { return 1, nil }

type fakeRows struct{}

func (r *fakeRows) Columns() []string              { return []string{"id"} }
func (r *fakeRows) Close() error                   { return nil }
func (r *fakeRows) Next([]driver.Value) error      { return io.EOF }

// ---------------------------------------------------------------------------
// Mock AuthRepository
// ---------------------------------------------------------------------------

type mockRepo struct {
	createUserFn               func(ctx context.Context, user *auth.User) error
	getUserByEmailHashFn       func(ctx context.Context, emailHash string) (*auth.User, error)
	getUserByIDFn              func(ctx context.Context, id uuid.UUID) (*auth.User, error)
	countRecentFailedAttemptsFn func(ctx context.Context, emailHash string, since time.Time) (int, error)
	countRecentIPAttemptsFn    func(ctx context.Context, ipAddress string, since time.Time) (int, error)
	getRefreshTokenFn          func(ctx context.Context, tokenHash string) (*auth.RefreshTokenRecord, error)
}

func (m *mockRepo) CreateUser(ctx context.Context, user *auth.User) error {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, user)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	return nil
}
func (m *mockRepo) GetUserByEmailHash(ctx context.Context, emailHash string) (*auth.User, error) {
	if m.getUserByEmailHashFn != nil {
		return m.getUserByEmailHashFn(ctx, emailHash)
	}
	return nil, nil
}
func (m *mockRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockRepo) UpdateUser(context.Context, *auth.User) error                          { return nil }
func (m *mockRepo) UpdateLastLogin(context.Context, uuid.UUID) error                 { return nil }
func (m *mockRepo) UpdateTOTPSecret(context.Context, uuid.UUID, []byte) error        { return nil }
func (m *mockRepo) EnableTOTP(context.Context, uuid.UUID) error                      { return nil }
func (m *mockRepo) DisableTOTP(context.Context, uuid.UUID) error                     { return nil }
func (m *mockRepo) UpdateStatus(context.Context, uuid.UUID, string) error            { return nil }
func (m *mockRepo) UpdatePassword(context.Context, uuid.UUID, string) error          { return nil }
func (m *mockRepo) ListPendingPhysicians(context.Context) ([]*auth.User, error)           { return nil, nil }
func (m *mockRepo) ListUsers(context.Context, string, string, int, int) ([]*auth.User, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) StoreRefreshToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time, _, _ string) error {
	return nil
}
func (m *mockRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*auth.RefreshTokenRecord, error) {
	if m.getRefreshTokenFn != nil {
		return m.getRefreshTokenFn(ctx, tokenHash)
	}
	return nil, nil
}
func (m *mockRepo) RevokeRefreshToken(context.Context, string, string) error    { return nil }
func (m *mockRepo) RevokeAllUserTokens(context.Context, uuid.UUID, string) error { return nil }
func (m *mockRepo) RecordLoginAttempt(context.Context, string, string, bool) error { return nil }
func (m *mockRepo) CountRecentFailedAttempts(ctx context.Context, emailHash string, since time.Time) (int, error) {
	if m.countRecentFailedAttemptsFn != nil {
		return m.countRecentFailedAttemptsFn(ctx, emailHash, since)
	}
	return 0, nil
}
func (m *mockRepo) CountRecentIPAttempts(ctx context.Context, ip string, since time.Time) (int, error) {
	if m.countRecentIPAttemptsFn != nil {
		return m.countRecentIPAttemptsFn(ctx, ip, since)
	}
	return 0, nil
}
func (m *mockRepo) ClearFailedAttempts(context.Context, string) error { return nil }

// ---------------------------------------------------------------------------
// Mock auth.JWTService
// ---------------------------------------------------------------------------

type mockJWT struct {
	generateAccessTokenFn  func(userID, role, orgID string, totpVerified bool) (string, string, error)
	generateRefreshTokenFn func(userID string) (string, string, error)
	validateAccessTokenFn  func(tokenString string) (*auth.AccessTokenClaims, error)
	validateRefreshTokenFn func(tokenString string) (*auth.RefreshTokenClaims, error)
	blacklistTokenFn       func(ctx context.Context, jti string, remaining time.Duration) error
	isBlacklistedFn        func(ctx context.Context, jti string) (bool, error)
}

func (m *mockJWT) GenerateAccessToken(userID, role, orgID string, totpVerified bool) (string, string, error) {
	if m.generateAccessTokenFn != nil {
		return m.generateAccessTokenFn(userID, role, orgID, totpVerified)
	}
	return "mock-access-token", "mock-jti", nil
}
func (m *mockJWT) GenerateRefreshToken(userID string) (string, string, error) {
	if m.generateRefreshTokenFn != nil {
		return m.generateRefreshTokenFn(userID)
	}
	return "mock-refresh-token", "mock-refresh-jti", nil
}
func (m *mockJWT) ValidateAccessToken(tokenString string) (*auth.AccessTokenClaims, error) {
	if m.validateAccessTokenFn != nil {
		return m.validateAccessTokenFn(tokenString)
	}
	return nil, nil
}
func (m *mockJWT) ValidateRefreshToken(tokenString string) (*auth.RefreshTokenClaims, error) {
	if m.validateRefreshTokenFn != nil {
		return m.validateRefreshTokenFn(tokenString)
	}
	return nil, nil
}
func (m *mockJWT) BlacklistToken(ctx context.Context, jti string, remaining time.Duration) error {
	if m.blacklistTokenFn != nil {
		return m.blacklistTokenFn(ctx, jti, remaining)
	}
	return nil
}
func (m *mockJWT) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if m.isBlacklistedFn != nil {
		return m.isBlacklistedFn(ctx, jti)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Mock auth.TOTPService
// ---------------------------------------------------------------------------

type mockTOTP struct {
	validateCodeFn func(secret, code string) bool
}

func (m *mockTOTP) GenerateSecret(string) (string, string, error) {
	return "JBSWY3DPEHPK3PXP", "data:image/png;base64,fake", nil
}
func (m *mockTOTP) ValidateCode(secret, code string) bool {
	if m.validateCodeFn != nil {
		return m.validateCodeFn(secret, code)
	}
	return true
}
func (m *mockTOTP) GenerateBackupCodes() ([]string, []string, error) {
	return []string{"CODE1", "CODE2"}, []string{"h1", "h2"}, nil
}
func (m *mockTOTP) ValidateBackupCode(string, []string) (int, bool) { return -1, false }

// ---------------------------------------------------------------------------
// Mock OTPService
// ---------------------------------------------------------------------------

type mockOTP struct{}

func (m *mockOTP) GenerateOTP() (string, string, error) { return "123456", "hash", nil }
func (m *mockOTP) HashOTP(code string) string            { return "hashed-" + code }

// ---------------------------------------------------------------------------
// Mock Encryptor (identity — returns input unchanged)
// ---------------------------------------------------------------------------

type mockEncryptor struct{}

func (m *mockEncryptor) Encrypt(p []byte) ([]byte, error)    { return p, nil }
func (m *mockEncryptor) Decrypt(c []byte) ([]byte, error)    { return c, nil }
func (m *mockEncryptor) EncryptString(s string) ([]byte, error) { return []byte(s), nil }
func (m *mockEncryptor) DecryptString(b []byte) (string, error) { return string(b), nil }

// ---------------------------------------------------------------------------
// Mock EmailService
// ---------------------------------------------------------------------------

type mockEmailSvc struct{}

func (m *mockEmailSvc) SendOTP(context.Context, string, string, string, string, string) error {
	return nil
}
func (m *mockEmailSvc) SendBreakGlassNotification(context.Context, notifications.BreakGlassNotification) error {
	return nil
}
func (m *mockEmailSvc) SendConsentGranted(context.Context, notifications.ConsentNotification) error {
	return nil
}
func (m *mockEmailSvc) SendConsentRevoked(context.Context, notifications.ConsentNotification) error {
	return nil
}
func (m *mockEmailSvc) SendWelcomePhysician(context.Context, string, string) error { return nil }
func (m *mockEmailSvc) SendPhysicianApproved(context.Context, string, string) error { return nil }
func (m *mockEmailSvc) SendAccountLocked(context.Context, string, string, time.Time) error {
	return nil
}

// ---------------------------------------------------------------------------
// Mock AuditLogger
// ---------------------------------------------------------------------------

type mockAuditLogger struct{}

func (m *mockAuditLogger) Log(context.Context, audit.AuditEntry) error { return nil }
func (m *mockAuditLogger) LogAsync(audit.AuditEntry)                    {}
func (m *mockAuditLogger) Close()                                       {}

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

type testHarness struct {
	repo    *mockRepo
	jwt     *mockJWT
	totp    *mockTOTP
	handler *auth.AuthHandler
	router  *gin.Engine
}

func setupTest() *testHarness {
	repo := &mockRepo{}
	jwtSvc := &mockJWT{}
	totpSvc := &mockTOTP{}

	svc := auth.NewAuthService(
		repo, jwtSvc, totpSvc, &mockOTP{},
		&mockEncryptor{}, &mockEmailSvc{}, &mockAuditLogger{},
		newFakeDB(), zerolog.Nop(),
	)
	handler := auth.NewAuthHandler(svc)

	return &testHarness{
		repo:    repo,
		jwt:     jwtSvc,
		totp:    totpSvc,
		handler: handler,
		router:  gin.New(),
	}
}

func jsonReq(method, path string, body interface{}) *http.Request {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// getDiagnostics extracts the first diagnostics string from an OperationOutcome.
func getDiagnostics(body []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	issues, _ := m["issue"].([]interface{})
	if len(issues) == 0 {
		return ""
	}
	first, _ := issues[0].(map[string]interface{})
	d, _ := first["diagnostics"].(string)
	return d
}

// ---------------------------------------------------------------------------
// 1. TestRegisterPhysician_201
// ---------------------------------------------------------------------------

func TestRegisterPhysician_201(t *testing.T) {
	h := setupTest()
	h.router.POST("/auth/register/physician", h.handler.RegisterPhysician)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/register/physician",
		auth.PhysicianRegistrationRequest{
			Email:     "drsmith@hospital.com",
			Password:  "Str0ng!Pass",
			FullName:  "Dr. Smith",
			MCINumber: "MCI12345",
		}))

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp auth.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp.Status)
	assert.NotEmpty(t, resp.UserID)
}

// ---------------------------------------------------------------------------
// 2. TestRegisterPhysician_409_Duplicate
// ---------------------------------------------------------------------------

func TestRegisterPhysician_409_Duplicate(t *testing.T) {
	h := setupTest()
	h.repo.getUserByEmailHashFn = func(_ context.Context, _ string) (*auth.User, error) {
		return &auth.User{ID: uuid.New(), Email: "drsmith@hospital.com"}, nil
	}
	h.router.POST("/auth/register/physician", h.handler.RegisterPhysician)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/register/physician",
		auth.PhysicianRegistrationRequest{
			Email:     "drsmith@hospital.com",
			Password:  "Str0ng!Pass",
			FullName:  "Dr. Smith",
			MCINumber: "MCI12345",
		}))

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// 3. TestRegisterPatient_201
// ---------------------------------------------------------------------------

func TestRegisterPatient_201(t *testing.T) {
	h := setupTest()
	h.router.POST("/auth/register/patient", h.handler.RegisterPatient)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/register/patient",
		auth.PatientRegistrationRequest{
			Email:       "jane@example.com",
			Password:    "Str0ng!Pass",
			FullName:    "Jane Patient",
			DateOfBirth: "1990-01-15",
			Gender:      "female",
		}))

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp auth.RegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "active", resp.Status)
	assert.NotEmpty(t, resp.UserID)
	assert.NotEmpty(t, resp.FHIRPatientID)
}

// ---------------------------------------------------------------------------
// 4. TestLogin_200_Patient
// ---------------------------------------------------------------------------

func TestLogin_200_Patient(t *testing.T) {
	h := setupTest()
	passHash, _ := auth.HashPassword("Str0ng!Pass")
	h.repo.getUserByEmailHashFn = func(_ context.Context, _ string) (*auth.User, error) {
		return &auth.User{
			ID: uuid.New(), Email: "patient@example.com",
			PasswordHash: passHash, Role: "patient", Status: "active",
		}, nil
	}
	h.router.POST("/auth/login", h.handler.Login)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "patient@example.com", Password: "Str0ng!Pass"}))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp auth.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "patient", resp.Role)
	assert.Equal(t, 7200, resp.ExpiresIn)
}

// ---------------------------------------------------------------------------
// 5. TestLogin_200_Physician_RequiresTOTP
// ---------------------------------------------------------------------------

func TestLogin_200_Physician_RequiresTOTP(t *testing.T) {
	h := setupTest()
	passHash, _ := auth.HashPassword("Str0ng!Pass")
	h.repo.getUserByEmailHashFn = func(_ context.Context, _ string) (*auth.User, error) {
		return &auth.User{
			ID: uuid.New(), Email: "dr@hospital.com",
			PasswordHash: passHash, Role: "physician", Status: "active",
			TOTPEnabled: true, TOTPSecret: []byte("secret"),
		}, nil
	}
	h.router.POST("/auth/login", h.handler.Login)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "dr@hospital.com", Password: "Str0ng!Pass"}))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp auth.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.RequiresTOTP)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Empty(t, resp.RefreshToken, "no refresh token until TOTP verified")
	assert.Equal(t, "physician", resp.Role)
}

// ---------------------------------------------------------------------------
// 6. TestLogin_401_WrongPassword
// ---------------------------------------------------------------------------

func TestLogin_401_WrongPassword(t *testing.T) {
	h := setupTest()
	passHash, _ := auth.HashPassword("Str0ng!Pass")
	h.repo.getUserByEmailHashFn = func(_ context.Context, _ string) (*auth.User, error) {
		return &auth.User{
			ID: uuid.New(), PasswordHash: passHash,
			Role: "patient", Status: "active",
		}, nil
	}
	h.router.POST("/auth/login", h.handler.Login)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "patient@example.com", Password: "Wr0ng!Pass"}))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// 7. TestLogin_401_PendingPhysician
// ---------------------------------------------------------------------------

func TestLogin_401_PendingPhysician(t *testing.T) {
	h := setupTest()
	passHash, _ := auth.HashPassword("Str0ng!Pass")
	h.repo.getUserByEmailHashFn = func(_ context.Context, _ string) (*auth.User, error) {
		return &auth.User{
			ID: uuid.New(), PasswordHash: passHash,
			Role: "physician", Status: "pending",
		}, nil
	}
	h.router.POST("/auth/login", h.handler.Login)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "dr@hospital.com", Password: "Str0ng!Pass"}))

	// "pending" maps to 403 via mapErrorToStatus
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// 8. TestLogin_429_TooManyAttempts
// ---------------------------------------------------------------------------

func TestLogin_429_TooManyAttempts(t *testing.T) {
	h := setupTest()
	h.repo.countRecentFailedAttemptsFn = func(context.Context, string, time.Time) (int, error) {
		return 5, nil // triggers lockout
	}
	h.router.POST("/auth/login", h.handler.Login)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "locked@example.com", Password: "Any!Pass123"}))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

// ---------------------------------------------------------------------------
// 9. TestLogin_401_NoUserEnumeration
// ---------------------------------------------------------------------------

func TestLogin_401_NoUserEnumeration(t *testing.T) {
	h := setupTest()
	passHash, _ := auth.HashPassword("Str0ng!Pass")
	h.router.POST("/auth/login", h.handler.Login)

	// Request 1: non-existent user
	h.repo.getUserByEmailHashFn = func(context.Context, string) (*auth.User, error) {
		return nil, nil
	}
	w1 := httptest.NewRecorder()
	h.router.ServeHTTP(w1, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "noone@example.com", Password: "Some!Pass1"}))
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Request 2: existing user, wrong password
	h.repo.getUserByEmailHashFn = func(context.Context, string) (*auth.User, error) {
		return &auth.User{
			ID: uuid.New(), PasswordHash: passHash,
			Role: "patient", Status: "active",
		}, nil
	}
	w2 := httptest.NewRecorder()
	h.router.ServeHTTP(w2, jsonReq(http.MethodPost, "/auth/login",
		auth.LoginRequest{Email: "real@example.com", Password: "Wr0ng!Pass"}))
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Both must return the same diagnostics message
	diag1 := getDiagnostics(w1.Body.Bytes())
	diag2 := getDiagnostics(w2.Body.Bytes())
	assert.Equal(t, diag1, diag2, "error messages must be identical to prevent enumeration")
	assert.Contains(t, diag1, "invalid credentials")
}

// ---------------------------------------------------------------------------
// 10. TestVerifyTOTP_200
// ---------------------------------------------------------------------------

func TestVerifyTOTP_200(t *testing.T) {
	h := setupTest()
	userID := uuid.New()

	h.repo.getUserByIDFn = func(_ context.Context, _ uuid.UUID) (*auth.User, error) {
		return &auth.User{
			ID: userID, Role: "physician", Status: "active",
			TOTPEnabled: true, TOTPSecret: []byte("the-secret"),
		}, nil
	}
	h.totp.validateCodeFn = func(_, code string) bool { return code == "123456" }

	h.router.POST("/auth/login/verify-totp", func(c *gin.Context) {
		c.Set(auth.ActorIDKey, userID)
		c.Set(auth.ActorRoleKey, "physician")
		c.Set(auth.RequestJTIKey, "old-jti")
		c.Next()
	}, h.handler.VerifyTOTP)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login/verify-totp",
		map[string]string{"code": "123456"}))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp auth.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "physician", resp.Role)
}

// ---------------------------------------------------------------------------
// 11. TestVerifyTOTP_401_Invalid
// ---------------------------------------------------------------------------

func TestVerifyTOTP_401_Invalid(t *testing.T) {
	h := setupTest()
	userID := uuid.New()

	h.repo.getUserByIDFn = func(_ context.Context, _ uuid.UUID) (*auth.User, error) {
		return &auth.User{
			ID: userID, Role: "physician", Status: "active",
			TOTPEnabled: true, TOTPSecret: []byte("the-secret"),
		}, nil
	}
	h.totp.validateCodeFn = func(string, string) bool { return false }

	h.router.POST("/auth/login/verify-totp", func(c *gin.Context) {
		c.Set(auth.ActorIDKey, userID)
		c.Set(auth.ActorRoleKey, "physician")
		c.Set(auth.RequestJTIKey, "old-jti")
		c.Next()
	}, h.handler.VerifyTOTP)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/login/verify-totp",
		map[string]string{"code": "000000"}))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// 12. TestLogout_204
// ---------------------------------------------------------------------------

func TestLogout_204(t *testing.T) {
	h := setupTest()

	h.router.POST("/auth/logout", func(c *gin.Context) {
		c.Set(auth.ActorIDKey, uuid.New())
		c.Set(auth.ActorRoleKey, "patient")
		c.Set(auth.RequestJTIKey, "some-jti")
		c.Next()
	}, h.handler.Logout)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/logout",
		map[string]string{"refreshToken": "rt"}))

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// 13. TestLogout_401_AlreadyLoggedOut
// ---------------------------------------------------------------------------

func TestLogout_401_AlreadyLoggedOut(t *testing.T) {
	h := setupTest()

	// Token validates structurally but is blacklisted
	h.jwt.validateAccessTokenFn = func(string) (*auth.AccessTokenClaims, error) {
		return &auth.AccessTokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "revoked-jti"},
			UserID:           uuid.New().String(),
			Role:             "patient",
			TOTPVerified:     true,
			TokenType:        "access",
		}, nil
	}
	h.jwt.isBlacklistedFn = func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	h.router.POST("/auth/logout", auth.AuthMiddleware(h.jwt), h.handler.Logout)

	req := jsonReq(http.MethodPost, "/auth/logout", map[string]string{})
	req.Header.Set("Authorization", "Bearer already-revoked-token")
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// 14. TestRefresh_200
// ---------------------------------------------------------------------------

func TestRefresh_200(t *testing.T) {
	h := setupTest()

	userID := uuid.New()
	refreshTok := "valid-refresh-token"
	tokHashBytes := sha256.Sum256([]byte(refreshTok))
	tokHash := hex.EncodeToString(tokHashBytes[:])

	h.jwt.validateRefreshTokenFn = func(string) (*auth.RefreshTokenClaims, error) {
		return &auth.RefreshTokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{ID: "rjti"},
			UserID:           userID.String(),
			TokenType:        "refresh",
		}, nil
	}
	h.repo.getRefreshTokenFn = func(_ context.Context, hash string) (*auth.RefreshTokenRecord, error) {
		if hash == tokHash {
			return &auth.RefreshTokenRecord{
				ID: uuid.New(), UserID: userID, TokenHash: hash,
				IssuedAt: time.Now().Add(-time.Hour),
				ExpiresAt: time.Now().Add(6 * 24 * time.Hour),
			}, nil
		}
		return nil, nil
	}
	h.repo.getUserByIDFn = func(_ context.Context, _ uuid.UUID) (*auth.User, error) {
		return &auth.User{
			ID: userID, Email: "p@example.com",
			Role: "patient", Status: "active",
		}, nil
	}

	h.router.POST("/auth/refresh", h.handler.RefreshToken)

	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, jsonReq(http.MethodPost, "/auth/refresh",
		map[string]string{"refreshToken": refreshTok}))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp auth.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "patient", resp.Role)
	assert.Equal(t, 7200, resp.ExpiresIn)
}
