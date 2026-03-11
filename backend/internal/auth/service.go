package auth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/Siddharthk17/MediLink/internal/audit"
	"github.com/Siddharthk17/MediLink/internal/notifications"
	"github.com/Siddharthk17/MediLink/pkg/crypto"
)

const (
	totpMaxAttempts    = 5
	totpAttemptWindow  = 10 * time.Minute
	totpLockoutDuration = 30 * time.Minute
)

// Request / Response types

// PhysicianRegistrationRequest is the payload for physician sign-up.
type PhysicianRegistrationRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	FullName       string `json:"fullName" binding:"required"`
	Phone          string `json:"phone"`
	MCINumber      string `json:"mciNumber" binding:"required"`
	Specialization string `json:"specialization"`
	OrganizationID string `json:"organizationId"`
}

// PatientRegistrationRequest is the payload for patient sign-up.
type PatientRegistrationRequest struct {
	Email             string `json:"email" binding:"required,email"`
	Password          string `json:"password" binding:"required"`
	FullName          string `json:"fullName" binding:"required"`
	DateOfBirth       string `json:"dateOfBirth" binding:"required"`
	Gender            string `json:"gender" binding:"required"`
	Phone             string `json:"phone"`
	PreferredLanguage string `json:"preferredLanguage"`
}

// LoginRequest is the payload for authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken,omitempty"`
	ExpiresIn        int    `json:"expiresIn"`
	Role             string `json:"role"`
	RequiresTOTP     bool   `json:"requiresTOTP,omitempty"`
	RequiresMFASetup bool   `json:"requiresMFASetup,omitempty"`
}

// RegisterResponse is returned after user registration.
type RegisterResponse struct {
	UserID        string `json:"userId"`
	FHIRPatientID string `json:"fhirPatientId,omitempty"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// UserProfile is the public representation of a user.
type UserProfile struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	FullName       string `json:"fullName"`
	Phone          string `json:"phone,omitempty"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	TOTPEnabled    bool   `json:"totpEnabled"`
	FHIRPatientID  string `json:"fhirPatientId,omitempty"`
	Specialization string `json:"specialization,omitempty"`
	MCINumber      string `json:"mciNumber,omitempty"`
	CreatedAt      string `json:"createdAt"`
	LastLoginAt    string `json:"lastLoginAt,omitempty"`
}

// TOTPSetupResponse is returned when initiating TOTP setup.
type TOTPSetupResponse struct {
	Secret string `json:"secret"`
	QRCode string `json:"qrCode"`
}

// TOTPSetupCompleteResponse is returned after TOTP verification succeeds.
type TOTPSetupCompleteResponse struct {
	BackupCodes []string `json:"backupCodes"`
	Message     string   `json:"message"`
}

// AuthService

// AuthService is the business-logic layer for authentication.
type AuthService struct {
	repo        AuthRepository
	jwtSvc      JWTService
	totpSvc     TOTPService
	otpSvc      OTPService
	encryptor   crypto.Encryptor
	emailSvc    notifications.EmailService
	auditLogger audit.AuditLogger
	db          *sqlx.DB
	logger      zerolog.Logger
	redisClient *redis.Client
}

// NewAuthService creates a new AuthService with all dependencies.
func NewAuthService(
	repo AuthRepository,
	jwtSvc JWTService,
	totpSvc TOTPService,
	otpSvc OTPService,
	encryptor crypto.Encryptor,
	emailSvc notifications.EmailService,
	auditLogger audit.AuditLogger,
	db *sqlx.DB,
	logger zerolog.Logger,
	redisClient ...*redis.Client,
) *AuthService {
	svc := &AuthService{
		repo: repo, jwtSvc: jwtSvc, totpSvc: totpSvc, otpSvc: otpSvc,
		encryptor: encryptor, emailSvc: emailSvc, auditLogger: auditLogger,
		db: db, logger: logger,
	}
	if len(redisClient) > 0 {
		svc.redisClient = redisClient[0]
	}
	return svc
}

// Registration

// RegisterPhysician creates a physician account in "pending" status.
func (s *AuthService) RegisterPhysician(ctx context.Context, req PhysicianRegistrationRequest) (*RegisterResponse, error) {
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.MCINumber == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	emailPrefix := strings.Split(req.Email, "@")[0]
	if violations := ValidatePasswordStrength(req.Password, emailPrefix); len(violations) > 0 {
		return nil, fmt.Errorf("password validation failed: %s", strings.Join(violations, "; "))
	}

	emailHash := HashEmail(req.Email)
	existing, err := s.repo.GetUserByEmailHash(ctx, emailHash)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to check email uniqueness")
		return nil, fmt.Errorf("internal error")
	}
	if existing != nil {
		return nil, fmt.Errorf("duplicate: account already exists")
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	emailEnc, err := s.encryptor.EncryptString(req.Email)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	fullNameEnc, err := s.encryptor.EncryptString(req.FullName)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	var phoneEnc []byte
	if req.Phone != "" {
		phoneEnc, err = s.encryptor.EncryptString(req.Phone)
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
	}

	user := &User{
		Email:        req.Email,
		EmailHash:    emailHash,
		EmailEnc:     emailEnc,
		PasswordHash: passwordHash,
		Role:         "physician",
		Status:       "pending",
		FullNameEnc:  fullNameEnc,
		PhoneEnc:     phoneEnc,
	}
	if req.MCINumber != "" {
		user.MCINumber = &req.MCINumber
	}
	if req.Specialization != "" {
		user.Specialization = &req.Specialization
	}
	if req.OrganizationID != "" {
		orgID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("invalid organization ID")
		}
		user.OrganizationID = &orgID
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error().Err(err).Msg("failed to create physician")
		return nil, fmt.Errorf("internal error")
	}

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &user.ID,
		UserRole:     "physician",
		ResourceType: "User",
		ResourceID:   user.ID.String(),
		Action:       "create",
		Success:      true,
		StatusCode:   201,
	})

	go func() {
		if err := s.emailSvc.SendWelcomePhysician(context.Background(), req.Email, req.FullName); err != nil {
			s.logger.Error().Err(err).Msg("failed to send welcome email")
		}
	}()

	return &RegisterResponse{
		UserID:  user.ID.String(),
		Status:  "pending",
		Message: "Physician registration submitted. Your account is pending admin approval.",
	}, nil
}

// RegisterPatient creates a patient account in "active" status.
func (s *AuthService) RegisterPatient(ctx context.Context, req PatientRegistrationRequest) (*RegisterResponse, error) {
	if req.Email == "" || req.Password == "" || req.FullName == "" || req.DateOfBirth == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	emailPrefix := strings.Split(req.Email, "@")[0]
	if violations := ValidatePasswordStrength(req.Password, emailPrefix); len(violations) > 0 {
		return nil, fmt.Errorf("password validation failed: %s", strings.Join(violations, "; "))
	}

	emailHash := HashEmail(req.Email)
	existing, err := s.repo.GetUserByEmailHash(ctx, emailHash)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to check email uniqueness")
		return nil, fmt.Errorf("internal error")
	}
	if existing != nil {
		return nil, fmt.Errorf("duplicate: account already exists")
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	emailEnc, err := s.encryptor.EncryptString(req.Email)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	fullNameEnc, err := s.encryptor.EncryptString(req.FullName)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	dobEnc, err := s.encryptor.EncryptString(req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	var phoneEnc []byte
	if req.Phone != "" {
		phoneEnc, err = s.encryptor.EncryptString(req.Phone)
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
	}

	user := &User{
		Email:        req.Email,
		EmailHash:    emailHash,
		EmailEnc:     emailEnc,
		PasswordHash: passwordHash,
		Role:         "patient",
		Status:       "active",
		FullNameEnc:  fullNameEnc,
		PhoneEnc:     phoneEnc,
		DobEnc:       dobEnc,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error().Err(err).Msg("failed to create patient")
		return nil, fmt.Errorf("internal error")
	}

	// Create FHIR Patient resource
	patientFHIRID := uuid.New().String()
	fhirData := map[string]interface{}{
		"resourceType": "Patient",
		"id":           patientFHIRID,
		"name":         []map[string]interface{}{{"text": req.FullName, "use": "official"}},
		"gender":       req.Gender,
		"birthDate":    req.DateOfBirth,
		"active":       true,
	}
	dataJSON, err := json.Marshal(fhirData)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal FHIR patient data")
		return nil, fmt.Errorf("internal error")
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO fhir_resources (resource_type, resource_id, data, status, created_by)
		 VALUES ('Patient', $1, $2, 'active', $3)`,
		patientFHIRID, dataJSON, user.ID,
	)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create FHIR patient resource")
	}

	_, _ = s.db.ExecContext(ctx,
		`UPDATE users SET fhir_patient_id = $1 WHERE id = $2`, patientFHIRID, user.ID)

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &user.ID,
		UserRole:     "patient",
		ResourceType: "User",
		ResourceID:   user.ID.String(),
		Action:       "create",
		Success:      true,
		StatusCode:   201,
	})

	return &RegisterResponse{
		UserID:        user.ID.String(),
		FHIRPatientID: patientFHIRID,
		Status:        "active",
		Message:       "Patient registration successful.",
	}, nil
}

// Login / TOTP / Refresh / Logout

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	emailHash := HashEmail(req.Email)

	// Email-based rate limit
	failedCount, err := s.repo.CountRecentFailedAttempts(ctx, emailHash, time.Now().Add(-15*time.Minute))
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to count login attempts")
		return nil, fmt.Errorf("internal error")
	}
	if failedCount >= 5 {
		return nil, fmt.Errorf("too many failed login attempts, please try again later")
	}

	// IP-based rate limit
	ipCount, err := s.repo.CountRecentIPAttempts(ctx, ipAddress, time.Now().Add(-15*time.Minute))
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to count IP login attempts")
		return nil, fmt.Errorf("internal error")
	}
	if ipCount >= 10 {
		return nil, fmt.Errorf("too many login attempts from this IP, please try again later")
	}

	user, err := s.repo.GetUserByEmailHash(ctx, emailHash)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to look up user")
		return nil, fmt.Errorf("internal error")
	}
	if user == nil {
		_ = s.repo.RecordLoginAttempt(ctx, emailHash, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Status checks — pending/suspended get specific messages; deleted looks like not-found
	switch user.Status {
	case "pending":
		return nil, fmt.Errorf("account is pending approval")
	case "suspended":
		return nil, fmt.Errorf("account has been suspended")
	}
	if user.DeletedAt != nil {
		_ = s.repo.RecordLoginAttempt(ctx, emailHash, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	if !VerifyPassword(user.PasswordHash, req.Password) {
		_ = s.repo.RecordLoginAttempt(ctx, emailHash, ipAddress, false)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Success path
	_ = s.repo.ClearFailedAttempts(ctx, emailHash)
	_ = s.repo.RecordLoginAttempt(ctx, emailHash, ipAddress, true)
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	orgID := ""
	if user.OrganizationID != nil {
		orgID = user.OrganizationID.String()
	}

	var resp LoginResponse
	resp.Role = user.Role
	resp.ExpiresIn = 7200

	switch {
	case user.Role == "patient":
		accessToken, _, err := s.jwtSvc.GenerateAccessToken(user.ID.String(), user.Role, orgID, true)
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
		resp.AccessToken = accessToken

		refreshToken, _, err := s.jwtSvc.GenerateRefreshToken(user.ID.String())
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
		resp.RefreshToken = refreshToken
		_ = s.repo.StoreRefreshToken(ctx, user.ID, hashToken(refreshToken),
			time.Now().Add(7*24*time.Hour), ipAddress, userAgent)

	case user.TOTPEnabled:
		// Physician/admin with TOTP — partial token, must verify
		accessToken, _, err := s.jwtSvc.GenerateAccessToken(user.ID.String(), user.Role, orgID, false)
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
		resp.AccessToken = accessToken
		resp.RequiresTOTP = true

	default:
		// Physician/admin without TOTP — full access, prompt setup
		accessToken, _, err := s.jwtSvc.GenerateAccessToken(user.ID.String(), user.Role, orgID, true)
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
		resp.AccessToken = accessToken
		resp.RequiresMFASetup = true

		refreshToken, _, err := s.jwtSvc.GenerateRefreshToken(user.ID.String())
		if err != nil {
			return nil, fmt.Errorf("internal error")
		}
		resp.RefreshToken = refreshToken
		_ = s.repo.StoreRefreshToken(ctx, user.ID, hashToken(refreshToken),
			time.Now().Add(7*24*time.Hour), ipAddress, userAgent)
	}

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:        &user.ID,
		UserRole:      user.Role,
		UserEmailHash: emailHash,
		ResourceType:  "Session",
		ResourceID:    user.ID.String(),
		Action:        "login",
		Success:       true,
		StatusCode:    200,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
	})

	return &resp, nil
}

// VerifyTOTP verifies a TOTP code and issues full tokens.
func (s *AuthService) VerifyTOTP(ctx context.Context, userID uuid.UUID, code, oldJTI, ipAddress, userAgent string) (*LoginResponse, error) {
	// Check lockout before doing any work
	if err := s.checkTOTPLockout(ctx, userID); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.TOTPEnabled || len(user.TOTPSecret) == 0 {
		return nil, fmt.Errorf("TOTP is not enabled for this account")
	}

	secret, err := s.encryptor.DecryptString(user.TOTPSecret)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to decrypt TOTP secret")
		return nil, fmt.Errorf("internal error")
	}

	if !s.totpSvc.ValidateCode(secret, code) {
		s.recordTOTPFailure(ctx, userID)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Successful verification — clear attempt counter
	s.clearTOTPAttempts(ctx, userID)

	// Blacklist the pre-MFA access token
	if oldJTI != "" {
		_ = s.jwtSvc.BlacklistToken(ctx, oldJTI, 2*time.Hour)
	}

	orgID := ""
	if user.OrganizationID != nil {
		orgID = user.OrganizationID.String()
	}

	accessToken, _, err := s.jwtSvc.GenerateAccessToken(user.ID.String(), user.Role, orgID, true)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	refreshToken, _, err := s.jwtSvc.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	_ = s.repo.StoreRefreshToken(ctx, user.ID, hashToken(refreshToken),
		time.Now().Add(7*24*time.Hour), ipAddress, userAgent)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    7200,
		Role:         user.Role,
	}, nil
}

// checkTOTPLockout returns an error if the user is currently locked out.
func (s *AuthService) checkTOTPLockout(ctx context.Context, userID uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}
	lockoutKey := fmt.Sprintf("totp:lockout:%s", userID.String())
	ttl, err := s.redisClient.TTL(ctx, lockoutKey).Result()
	if err != nil {
		s.logger.Warn().Err(err).Str("userID", userID.String()).Msg("failed to check TOTP lockout in Redis")
		return nil
	}
	if ttl > 0 {
		remaining := int(ttl.Seconds())
		s.logger.Warn().Str("userID", userID.String()).Int("retryAfter", remaining).Msg("TOTP account locked")
		return fmt.Errorf("too many failed TOTP attempts, account temporarily locked — retry after %d seconds", remaining)
	}
	return nil
}

// recordTOTPFailure increments the failed-attempt counter and triggers lockout if threshold is exceeded.
func (s *AuthService) recordTOTPFailure(ctx context.Context, userID uuid.UUID) {
	if s.redisClient == nil {
		return
	}
	attemptsKey := fmt.Sprintf("totp:attempts:%s", userID.String())

	count, err := s.redisClient.Incr(ctx, attemptsKey).Result()
	if err != nil {
		s.logger.Warn().Err(err).Str("userID", userID.String()).Msg("failed to increment TOTP attempts in Redis")
		return
	}

	// Set expiry on first attempt
	if count == 1 {
		if err := s.redisClient.Expire(ctx, attemptsKey, totpAttemptWindow).Err(); err != nil {
			s.logger.Warn().Err(err).Str("userID", userID.String()).Msg("failed to set expiry on TOTP attempts key")
		}
	}

	s.logger.Info().Str("userID", userID.String()).Int64("attempts", count).Msg("failed TOTP attempt")

	if count >= int64(totpMaxAttempts) {
		lockoutKey := fmt.Sprintf("totp:lockout:%s", userID.String())
		if err := s.redisClient.Set(ctx, lockoutKey, "1", totpLockoutDuration).Err(); err != nil {
			s.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to set TOTP lockout in Redis")
			return
		}
		// Clean up attempts key since lockout is now active
		if err := s.redisClient.Del(ctx, attemptsKey).Err(); err != nil {
			s.logger.Warn().Err(err).Str("userID", userID.String()).Msg("failed to clean up TOTP attempts key after lockout")
		}
		s.logger.Warn().Str("userID", userID.String()).Msg("TOTP account locked due to too many failed attempts")
	}
}

// clearTOTPAttempts removes the failed-attempt counter after a successful verification.
func (s *AuthService) clearTOTPAttempts(ctx context.Context, userID uuid.UUID) {
	if s.redisClient == nil {
		return
	}
	attemptsKey := fmt.Sprintf("totp:attempts:%s", userID.String())
	if err := s.redisClient.Del(ctx, attemptsKey).Err(); err != nil {
		s.logger.Warn().Err(err).Str("userID", userID.String()).Msg("failed to clear TOTP attempts in Redis")
	}
}

// RefreshToken rotates tokens using a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString, ipAddress, userAgent string) (*LoginResponse, error) {
	if _, err := s.jwtSvc.ValidateRefreshToken(refreshTokenString); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	tokenHash := hashToken(refreshTokenString)
	rt, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil || rt == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if rt.RevokedAt != nil {
		_ = s.repo.RevokeAllUserTokens(ctx, rt.UserID, "suspected_token_reuse")
		return nil, fmt.Errorf("invalid credentials")
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, fmt.Errorf("invalid credentials")
	}

	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if user.Status != "active" || user.DeletedAt != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	_ = s.repo.RevokeRefreshToken(ctx, tokenHash, "rotation")

	orgID := ""
	if user.OrganizationID != nil {
		orgID = user.OrganizationID.String()
	}

	accessToken, _, err := s.jwtSvc.GenerateAccessToken(user.ID.String(), user.Role, orgID, true)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	newRefreshToken, _, err := s.jwtSvc.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}
	_ = s.repo.StoreRefreshToken(ctx, user.ID, hashToken(newRefreshToken),
		time.Now().Add(7*24*time.Hour), ipAddress, userAgent)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    7200,
		Role:         user.Role,
	}, nil
}

// Logout blacklists the access token and revokes the refresh token.
func (s *AuthService) Logout(ctx context.Context, jti, refreshTokenString string) error {
	if jti != "" {
		_ = s.jwtSvc.BlacklistToken(ctx, jti, 2*time.Hour)
	}
	if refreshTokenString != "" {
		_ = s.repo.RevokeRefreshToken(ctx, hashToken(refreshTokenString), "logout")
	}
	return nil
}

// Profile / TOTP setup

// GetMe returns the authenticated user's profile with decrypted PII.
func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	fullName := ""
	if len(user.FullNameEnc) > 0 {
		var err error
		fullName, err = s.encryptor.DecryptString(user.FullNameEnc)
		if err != nil {
			s.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to decrypt user full name")
		}
	}
	phone := ""
	if len(user.PhoneEnc) > 0 {
		var err error
		phone, err = s.encryptor.DecryptString(user.PhoneEnc)
		if err != nil {
			s.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to decrypt user phone")
		}
	}

	profile := &UserProfile{
		ID:          user.ID.String(),
		Email:       user.Email,
		FullName:    fullName,
		Phone:       phone,
		Role:        user.Role,
		Status:      user.Status,
		TOTPEnabled: user.TOTPEnabled,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
	if user.FHIRPatientID != nil {
		profile.FHIRPatientID = *user.FHIRPatientID
	}
	if user.Specialization != nil {
		profile.Specialization = *user.Specialization
	}
	if user.MCINumber != nil {
		profile.MCINumber = *user.MCINumber
	}
	if user.LastLoginAt != nil {
		profile.LastLoginAt = user.LastLoginAt.Format(time.RFC3339)
	}
	return profile, nil
}

// SetupTOTP generates a TOTP secret and QR code for the user.
func (s *AuthService) SetupTOTP(ctx context.Context, userID uuid.UUID, email string) (*TOTPSetupResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.Role != "physician" && user.Role != "admin" {
		return nil, fmt.Errorf("TOTP is only available for physicians and admins")
	}
	if user.TOTPEnabled {
		return nil, fmt.Errorf("TOTP is already enabled")
	}

	if email == "" {
		email = user.Email
	}

	secret, qrCode, err := s.totpSvc.GenerateSecret(email)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	encSecret, err := s.encryptor.EncryptString(secret)
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	if err := s.repo.UpdateTOTPSecret(ctx, userID, encSecret); err != nil {
		return nil, fmt.Errorf("internal error")
	}

	return &TOTPSetupResponse{Secret: secret, QRCode: qrCode}, nil
}

// VerifyTOTPSetup confirms the TOTP code is valid and enables MFA.
func (s *AuthService) VerifyTOTPSetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPSetupCompleteResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if len(user.TOTPSecret) == 0 {
		return nil, fmt.Errorf("TOTP has not been set up yet")
	}

	secret, err := s.encryptor.DecryptString(user.TOTPSecret)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to decrypt TOTP secret")
		return nil, fmt.Errorf("internal error")
	}

	if !s.totpSvc.ValidateCode(secret, code) {
		return nil, fmt.Errorf("invalid TOTP code")
	}

	if err := s.repo.EnableTOTP(ctx, userID); err != nil {
		return nil, fmt.Errorf("internal error")
	}

	codes, hashes, err := s.totpSvc.GenerateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("internal error")
	}

	if err := s.repo.StoreBackupCodes(ctx, userID, hashes); err != nil {
		s.logger.Error().Err(err).Msg("failed to store backup codes")
		return nil, fmt.Errorf("internal error")
	}

	return &TOTPSetupCompleteResponse{
		BackupCodes: codes,
		Message:     "TOTP has been enabled. Save your backup codes securely — they will not be shown again.",
	}, nil
}

// Password / Admin actions

// ChangePassword verifies the old password, validates the new one, and updates.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	if !VerifyPassword(user.PasswordHash, oldPassword) {
		return fmt.Errorf("invalid credentials")
	}

	emailPrefix := strings.Split(user.Email, "@")[0]
	if violations := ValidatePasswordStrength(newPassword, emailPrefix); len(violations) > 0 {
		return fmt.Errorf("password validation failed: %s", strings.Join(violations, "; "))
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("internal error")
	}

	if err := s.repo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("internal error")
	}

	_ = s.repo.RevokeAllUserTokens(ctx, userID, "password_changed")

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &user.ID,
		UserRole:     user.Role,
		ResourceType: "User",
		ResourceID:   user.ID.String(),
		Action:       "password_change",
		Success:      true,
		StatusCode:   200,
	})

	return nil
}

// ApprovePhysician transitions a physician from pending to active.
func (s *AuthService) ApprovePhysician(ctx context.Context, userID, adminID uuid.UUID) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}
	if user.Status != "pending" {
		return fmt.Errorf("user is not in pending status")
	}

	if err := s.repo.UpdateStatus(ctx, userID, "active"); err != nil {
		return fmt.Errorf("internal error")
	}

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &adminID,
		UserRole:     "admin",
		ResourceType: "User",
		ResourceID:   userID.String(),
		Action:       "approve_physician",
		Success:      true,
		StatusCode:   200,
	})

	fullName := ""
	if len(user.FullNameEnc) > 0 {
		fullName, _ = s.encryptor.DecryptString(user.FullNameEnc)
	}
	go func() {
		if err := s.emailSvc.SendPhysicianApproved(context.Background(), user.Email, fullName); err != nil {
			s.logger.Error().Err(err).Msg("failed to send physician approval email")
		}
	}()

	return nil
}

// SuspendPhysician suspends a physician and revokes all tokens.
func (s *AuthService) SuspendPhysician(ctx context.Context, userID, adminID uuid.UUID, reason string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	if err := s.repo.UpdateStatus(ctx, userID, "suspended"); err != nil {
		return fmt.Errorf("internal error")
	}
	_ = s.repo.RevokeAllUserTokens(ctx, userID, "account_suspended")

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &adminID,
		UserRole:     "admin",
		ResourceType: "User",
		ResourceID:   userID.String(),
		Action:       "suspend_physician",
		Success:      true,
		StatusCode:   200,
		ErrorMessage: reason,
	})

	return nil
}

// ReinstatePhysician reactivates a previously suspended physician.
func (s *AuthService) ReinstatePhysician(ctx context.Context, userID, adminID uuid.UUID) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	if err := s.repo.UpdateStatus(ctx, userID, "active"); err != nil {
		return fmt.Errorf("internal error")
	}

	s.auditLogger.LogAsync(audit.AuditEntry{
		UserID:       &adminID,
		UserRole:     "admin",
		ResourceType: "User",
		ResourceID:   userID.String(),
		Action:       "reinstate_physician",
		Success:      true,
		StatusCode:   200,
	})

	return nil
}

// Helpers

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
