package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// User represents a user record in the database.
type User struct {
	ID             uuid.UUID  `db:"id"`
	Email          string     `db:"email"`
	EmailHash      string     `db:"email_hash"`
	EmailEnc       []byte     `db:"email_enc"`
	PasswordHash   string     `db:"password_hash"`
	Role           string     `db:"role"`
	Status         string     `db:"status"`
	FullNameEnc    []byte     `db:"full_name_enc"`
	PhoneEnc       []byte     `db:"phone_enc"`
	DobEnc         []byte     `db:"dob_enc"`
	AddressEnc     []byte     `db:"address_enc"`
	MCINumber      *string    `db:"mci_number"`
	Specialization *string    `db:"specialization"`
	OrganizationID *uuid.UUID `db:"organization_id"`
	TOTPSecret     []byte     `db:"totp_secret"`
	TOTPEnabled    bool       `db:"totp_enabled"`
	TOTPVerified   bool       `db:"totp_verified"`
	FHIRPatientID  *string    `db:"fhir_patient_id"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
	LastLoginAt    *time.Time `db:"last_login_at"`
	DeletedAt      *time.Time `db:"deleted_at"`
}

// RefreshTokenRecord represents a refresh token stored in the database.
type RefreshTokenRecord struct {
	ID            uuid.UUID  `db:"id"`
	UserID        uuid.UUID  `db:"user_id"`
	TokenHash     string     `db:"token_hash"`
	IssuedAt      time.Time  `db:"issued_at"`
	ExpiresAt     time.Time  `db:"expires_at"`
	RevokedAt     *time.Time `db:"revoked_at"`
	RevokedReason *string    `db:"revoked_reason"`
	IPAddress     *string    `db:"ip_address"`
	UserAgent     *string    `db:"user_agent"`
}

// AuthRepository defines the data access interface for authentication.
type AuthRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmailHash(ctx context.Context, emailHash string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret []byte) error
	EnableTOTP(ctx context.Context, userID uuid.UUID) error
	DisableTOTP(ctx context.Context, userID uuid.UUID) error
	UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	ListPendingPhysicians(ctx context.Context) ([]*User, error)
	ListUsers(ctx context.Context, role, status string, limit, offset int) ([]*User, int, error)

	// Refresh tokens
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, ipAddress, userAgent string) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string, reason string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) error

	// Login attempts
	RecordLoginAttempt(ctx context.Context, emailHash, ipAddress string, success bool) error
	CountRecentFailedAttempts(ctx context.Context, emailHash string, since time.Time) (int, error)
	CountRecentIPAttempts(ctx context.Context, ipAddress string, since time.Time) (int, error)
	ClearFailedAttempts(ctx context.Context, emailHash string) error

	// Backup codes
	StoreBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error
	GetBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error)
	MarkBackupCodeUsed(ctx context.Context, userID uuid.UUID, codeHash string) error
	DeleteBackupCodes(ctx context.Context, userID uuid.UUID) error
}

// HashEmail returns the SHA-256 hex hash of a lowercased, trimmed email.
func HashEmail(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("%x", h)
}

type postgresAuthRepository struct {
	db *sqlx.DB
}

// NewAuthRepository creates a new PostgreSQL-backed auth repository.
func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &postgresAuthRepository{db: db}
}

func (r *postgresAuthRepository) CreateUser(ctx context.Context, user *User) error {
	query := `INSERT INTO users (email, email_hash, email_enc, password_hash, role, status,
              full_name_enc, phone_enc, dob_enc, address_enc, mci_number, specialization,
              organization_id, fhir_patient_id)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
              RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		user.Email, user.EmailHash, user.EmailEnc, user.PasswordHash, user.Role, user.Status,
		user.FullNameEnc, user.PhoneEnc, user.DobEnc, user.AddressEnc,
		user.MCINumber, user.Specialization, user.OrganizationID, user.FHIRPatientID,
	).Scan(&user.ID, &user.CreatedAt)
}

func (r *postgresAuthRepository) GetUserByEmailHash(ctx context.Context, emailHash string) (*User, error) {
	user := &User{}
	err := r.db.GetContext(ctx, user,
		`SELECT id, email, email_hash, email_enc, COALESCE(password_hash,'') as password_hash,
		 role, status, full_name_enc, phone_enc, dob_enc, address_enc,
		 mci_number, specialization, organization_id, totp_secret, totp_enabled,
		 totp_verified, fhir_patient_id, created_at, updated_at, last_login_at, deleted_at
		 FROM users WHERE email_hash = $1 AND deleted_at IS NULL`, emailHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email hash: %w", err)
	}
	return user, nil
}

func (r *postgresAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := r.db.GetContext(ctx, user,
		`SELECT id, email, email_hash, email_enc, COALESCE(password_hash,'') as password_hash,
		 role, status, full_name_enc, phone_enc, dob_enc, address_enc,
		 mci_number, specialization, organization_id, totp_secret, totp_enabled,
		 totp_verified, fhir_patient_id, created_at, updated_at, last_login_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (r *postgresAuthRepository) UpdateUser(ctx context.Context, user *User) error {
	query := `UPDATE users SET email = $1, email_hash = $2, email_enc = $3,
              full_name_enc = $4, phone_enc = $5, dob_enc = $6, address_enc = $7,
              mci_number = $8, specialization = $9, organization_id = $10,
              updated_at = NOW()
              WHERE id = $11 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query,
		user.Email, user.EmailHash, user.EmailEnc,
		user.FullNameEnc, user.PhoneEnc, user.DobEnc, user.AddressEnc,
		user.MCINumber, user.Specialization, user.OrganizationID,
		user.ID,
	)
	return err
}

func (r *postgresAuthRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *postgresAuthRepository) UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_secret = $1, updated_at = NOW() WHERE id = $2`, secret, userID)
	return err
}

func (r *postgresAuthRepository) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_enabled = TRUE, totp_verified = TRUE, updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *postgresAuthRepository) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET totp_enabled = FALSE, totp_verified = FALSE, totp_secret = NULL, updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *postgresAuthRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`, status, userID)
	return err
}

func (r *postgresAuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, passwordHash, userID)
	return err
}

func (r *postgresAuthRepository) ListPendingPhysicians(ctx context.Context) ([]*User, error) {
	var users []*User
	err := r.db.SelectContext(ctx, &users,
		`SELECT id, email, email_hash, email_enc, COALESCE(password_hash,'') as password_hash,
		 role, status, full_name_enc, phone_enc, dob_enc, address_enc,
		 mci_number, specialization, organization_id, totp_secret, totp_enabled,
		 totp_verified, fhir_patient_id, created_at, updated_at, last_login_at, deleted_at
		 FROM users WHERE role = 'physician' AND status = 'pending' AND deleted_at IS NULL
		 ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending physicians: %w", err)
	}
	return users, nil
}

func (r *postgresAuthRepository) ListUsers(ctx context.Context, role, status string, limit, offset int) ([]*User, int, error) {
	args := []interface{}{}
	conditions := []string{"deleted_at IS NULL"}
	argIdx := 1

	if role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", where)
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	dataQuery := fmt.Sprintf(
		`SELECT id, email, email_hash, email_enc, COALESCE(password_hash,'') as password_hash,
		 role, status, full_name_enc, phone_enc, dob_enc, address_enc,
		 mci_number, specialization, organization_id, totp_secret, totp_enabled,
		 totp_verified, fhir_patient_id, created_at, updated_at, last_login_at, deleted_at
		 FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	var users []*User
	err = r.db.SelectContext(ctx, &users, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (r *postgresAuthRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, ipAddress, userAgent string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, tokenHash, expiresAt, ipAddress, userAgent,
	)
	return err
}

func (r *postgresAuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error) {
	rt := &RefreshTokenRecord{}
	err := r.db.GetContext(ctx, rt,
		`SELECT * FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return rt, nil
}

func (r *postgresAuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW(), revoked_reason = $1
		 WHERE token_hash = $2 AND revoked_at IS NULL`,
		reason, tokenHash,
	)
	return err
}

func (r *postgresAuthRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW(), revoked_reason = $1
		 WHERE user_id = $2 AND revoked_at IS NULL`,
		reason, userID,
	)
	return err
}

func (r *postgresAuthRepository) RecordLoginAttempt(ctx context.Context, emailHash, ipAddress string, success bool) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO login_attempts (email_hash, ip_address, success) VALUES ($1, $2, $3)`,
		emailHash, ipAddress, success,
	)
	return err
}

func (r *postgresAuthRepository) CountRecentFailedAttempts(ctx context.Context, emailHash string, since time.Time) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM login_attempts
		 WHERE email_hash = $1 AND attempted_at > $2 AND success = FALSE`,
		emailHash, since,
	)
	return count, err
}

func (r *postgresAuthRepository) CountRecentIPAttempts(ctx context.Context, ipAddress string, since time.Time) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM login_attempts
		 WHERE ip_address = $1 AND attempted_at > $2 AND success = FALSE`,
		ipAddress, since,
	)
	return count, err
}

func (r *postgresAuthRepository) ClearFailedAttempts(ctx context.Context, emailHash string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM login_attempts WHERE email_hash = $1 AND success = FALSE`,
		emailHash,
	)
	return err
}

func (r *postgresAuthRepository) StoreBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete any existing codes first
	if _, err := tx.ExecContext(ctx, `DELETE FROM totp_backup_codes WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("failed to delete old backup codes: %w", err)
	}

	for _, hash := range codeHashes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO totp_backup_codes (user_id, code_hash) VALUES ($1, $2)`,
			userID, hash,
		); err != nil {
			return fmt.Errorf("failed to store backup code: %w", err)
		}
	}

	return tx.Commit()
}

func (r *postgresAuthRepository) GetBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var hashes []string
	err := r.db.SelectContext(ctx, &hashes,
		`SELECT code_hash FROM totp_backup_codes WHERE user_id = $1 AND used_at IS NULL ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup codes: %w", err)
	}
	return hashes, nil
}

func (r *postgresAuthRepository) MarkBackupCodeUsed(ctx context.Context, userID uuid.UUID, codeHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE totp_backup_codes SET used_at = NOW() WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL`,
		userID, codeHash,
	)
	return err
}

func (r *postgresAuthRepository) DeleteBackupCodes(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM totp_backup_codes WHERE user_id = $1`,
		userID,
	)
	return err
}
