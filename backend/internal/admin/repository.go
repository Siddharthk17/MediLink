// Package admin provides the Admin Panel API for platform management.
package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// UserFilters defines filters for listing users.
type UserFilters struct {
	Role          string
	Status        string
	Query         string // full-text search on email_hash
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Sort          string // created_at, -created_at, last_login_at, -last_login_at
}

// UserSummary is a safe projection of user data — never includes password_hash, totp_secret, or raw email.
type UserSummary struct {
	ID            string     `json:"id" db:"id"`
	Role          string     `json:"role" db:"role"`
	Status        string     `json:"status" db:"status"`
	FullName      string     `json:"fullName"` // AES decrypted, not from DB directly
	FullNameEnc   []byte     `json:"-" db:"full_name_enc"`
	EmailHash     string     `json:"emailHash" db:"email_hash"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	LastLoginAt   *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	FHIRPatientID *string    `json:"fhirPatientId,omitempty" db:"fhir_patient_id"`
	TOTPEnabled   bool       `json:"totpEnabled" db:"totp_enabled"`
}

// DoctorSummary is a public-facing doctor profile for the directory.
type DoctorSummary struct {
	ID             string `json:"id" db:"id"`
	FullName       string `json:"fullName"`
	FullNameEnc    []byte `json:"-" db:"full_name_enc"`
	Specialization string `json:"specialization" db:"specialization"`
	MciNumber      string `json:"mciNumber,omitempty" db:"mci_number"`
}

// UserDetail extends UserSummary with additional fields for single-user view.
type UserDetail struct {
	UserSummary
	MciNumber      *string    `json:"mciNumber,omitempty" db:"mci_number"`
	Specialization *string    `json:"specialization,omitempty" db:"specialization"`
	TOTPVerified   bool       `json:"totpVerified" db:"totp_verified"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
}

// AuditLogFilters defines filters for querying audit logs.
type AuditLogFilters struct {
	ActorID        string
	ResourceType   string
	Action         string
	PatientRef     string
	From           *time.Time
	To             *time.Time
	BreakGlassOnly bool
	Sort           string // default: -created_at (newest first)
}

// AuditLogEntry represents a single audit log record.
type AuditLogEntry struct {
	ID           string    `json:"id" db:"id"`
	UserID       *string   `json:"userId,omitempty" db:"user_id"`
	UserRole     string    `json:"userRole" db:"user_role"`
	UserEmail    string    `json:"userEmail" db:"user_email"`
	ResourceType string    `json:"resourceType" db:"resource_type"`
	ResourceID   string    `json:"resourceId" db:"resource_id"`
	Action       string    `json:"action" db:"action"`
	PatientRef   string    `json:"patientRef,omitempty" db:"patient_ref"`
	Purpose      string    `json:"purpose" db:"purpose"`
	Success      bool      `json:"success" db:"success"`
	StatusCode   int       `json:"statusCode" db:"status_code"`
	IPAddress    string    `json:"ipAddress" db:"ip_address"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// AdminStats holds aggregated platform statistics.
type AdminStats struct {
	Users struct {
		Total       int `json:"total"`
		Patients    int `json:"patients"`
		Physicians  int `json:"physicians"`
		Admins      int `json:"admins"`
		Researchers int `json:"researchers"`
		Pending     int `json:"pending"`
	} `json:"users"`
	FHIRResources struct {
		Total  int            `json:"total"`
		ByType map[string]int `json:"byType"`
	} `json:"fhirResources"`
	Documents struct {
		TotalUploaded int `json:"totalUploaded"`
		Completed     int `json:"completed"`
		Failed        int `json:"failed"`
		Pending       int `json:"pending"`
	} `json:"documents"`
	DrugChecks struct {
		Total           int `json:"total"`
		Contraindicated int `json:"contraindicated"`
		Major           int `json:"major"`
		Moderate        int `json:"moderate"`
	} `json:"drugChecks"`
	Consents struct {
		ActiveGrants        int `json:"activeGrants"`
		RevokedThisMonth    int `json:"revokedThisMonth"`
		BreakGlassThisMonth int `json:"breakGlassThisMonth"`
	} `json:"consents"`
	Exports struct {
		Total     int `json:"total"`
		Completed int `json:"completed"`
		Pending   int `json:"pending"`
	} `json:"exports"`
}

// AdminRepository provides database queries for admin operations.
type AdminRepository struct {
	db *sqlx.DB
}

// NewAdminRepository creates a new AdminRepository.
func NewAdminRepository(db *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// safeUserColumns is the set of columns safe to select — never password_hash, totp_secret, raw email.
const safeUserColumns = `id, role, status, full_name_enc, email_hash, created_at, last_login_at, fhir_patient_id, totp_enabled`

const safeUserDetailColumns = `id, role, status, full_name_enc, email_hash, created_at, last_login_at,
	fhir_patient_id, totp_enabled, mci_number, specialization, totp_verified, updated_at`

// ListUsers returns a paginated, filtered list of users.
func (r *AdminRepository) ListUsers(ctx context.Context, filters UserFilters, count, offset int) ([]UserSummary, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, filters.Role)
		argIdx++
	}
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Query != "" {
		conditions = append(conditions, fmt.Sprintf("email_hash = $%d", argIdx))
		args = append(args, filters.Query)
		argIdx++
	}
	if filters.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filters.CreatedAfter)
		argIdx++
	}
	if filters.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filters.CreatedBefore)
		argIdx++
	}

	// Always exclude soft-deleted
	conditions = append(conditions, "deleted_at IS NULL")

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderBy := "ORDER BY created_at DESC"
	switch filters.Sort {
	case "created_at":
		orderBy = "ORDER BY created_at ASC"
	case "-created_at":
		orderBy = "ORDER BY created_at DESC"
	case "last_login_at":
		orderBy = "ORDER BY last_login_at ASC NULLS LAST"
	case "-last_login_at":
		orderBy = "ORDER BY last_login_at DESC NULLS LAST"
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Fetch page
	dataQuery := fmt.Sprintf("SELECT %s FROM users %s %s LIMIT $%d OFFSET $%d",
		safeUserColumns, where, orderBy, argIdx, argIdx+1)
	args = append(args, count, offset)

	var users []UserSummary
	if err := r.db.SelectContext(ctx, &users, dataQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// GetUser returns a single user with full details.
func (r *AdminRepository) GetUser(ctx context.Context, userID string) (*UserDetail, error) {
	query := fmt.Sprintf("SELECT %s FROM users WHERE id = $1 AND deleted_at IS NULL", safeUserDetailColumns)
	var user UserDetail
	if err := r.db.GetContext(ctx, &user, query, userID); err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// UpdateUserRole changes a user's role.
func (r *AdminRepository) UpdateUserRole(ctx context.Context, userID, newRole string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL",
		newRole, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// InviteResearcher creates a new user with researcher role and pending status.
func (r *AdminRepository) InviteResearcher(ctx context.Context, email, invitedBy string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, email_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, 'researcher', 'pending', NOW(), NOW())`,
		email, email, // email_hash should be SHA-256 in practice; service layer handles hashing
	)
	if err != nil {
		return fmt.Errorf("failed to invite researcher: %w", err)
	}
	return nil
}

// SuspendPhysician sets a user's status to suspended.
func (r *AdminRepository) SuspendPhysician(ctx context.Context, userID string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET status = 'suspended', updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL",
		userID)
	if err != nil {
		return fmt.Errorf("failed to suspend physician: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ReinstatePhysician sets a user's status to active.
func (r *AdminRepository) ReinstatePhysician(ctx context.Context, userID string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET status = 'active', updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL",
		userID)
	if err != nil {
		return fmt.Errorf("failed to reinstate physician: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ListDoctors returns active physicians for the public doctor directory.
func (r *AdminRepository) ListDoctors(ctx context.Context, specialization string) ([]DoctorSummary, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "role = 'physician'", "status = 'active'", "deleted_at IS NULL")

	if specialization != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(specialization) = LOWER($%d)", argIdx))
		args = append(args, specialization)
		argIdx++
	}
	_ = argIdx

	where := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf("SELECT id, full_name_enc, COALESCE(specialization, '') as specialization, COALESCE(mci_number, '') as mci_number FROM users %s ORDER BY created_at ASC", where)

	var doctors []DoctorSummary
	if err := r.db.SelectContext(ctx, &doctors, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list doctors: %w", err)
	}
	return doctors, nil
}

const auditLogColumns = `a.id, a.user_id, a.user_role, COALESCE(u.email, '') as user_email,
	a.resource_type, a.resource_id, a.action,
	a.patient_ref, a.purpose, a.success, a.status_code,
	COALESCE(a.ip_address::text, '') as ip_address, a.created_at`

// GetAuditLogs returns a paginated, filtered list of audit log entries.
func (r *AdminRepository) GetAuditLogs(ctx context.Context, filters AuditLogFilters, count, offset int) ([]AuditLogEntry, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filters.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("a.user_id = $%d", argIdx))
		args = append(args, filters.ActorID)
		argIdx++
	}
	if filters.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("a.resource_type = $%d", argIdx))
		args = append(args, filters.ResourceType)
		argIdx++
	}
	if filters.Action != "" {
		conditions = append(conditions, fmt.Sprintf("a.action = $%d", argIdx))
		args = append(args, filters.Action)
		argIdx++
	}
	if filters.PatientRef != "" {
		conditions = append(conditions, fmt.Sprintf("a.patient_ref = $%d", argIdx))
		args = append(args, filters.PatientRef)
		argIdx++
	}
	if filters.From != nil {
		conditions = append(conditions, fmt.Sprintf("a.created_at >= $%d", argIdx))
		args = append(args, *filters.From)
		argIdx++
	}
	if filters.To != nil {
		conditions = append(conditions, fmt.Sprintf("a.created_at <= $%d", argIdx))
		args = append(args, *filters.To)
		argIdx++
	}
	if filters.BreakGlassOnly {
		conditions = append(conditions, "a.action = 'break_glass'")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderBy := "ORDER BY a.created_at DESC"
	switch filters.Sort {
	case "created_at":
		orderBy = "ORDER BY a.created_at ASC"
	case "-created_at":
		orderBy = "ORDER BY a.created_at DESC"
	}

	fromClause := "audit_logs a LEFT JOIN users u ON a.user_id = u.id"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", fromClause, where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	dataQuery := fmt.Sprintf("SELECT %s FROM %s %s %s LIMIT $%d OFFSET $%d",
		auditLogColumns, fromClause, where, orderBy, argIdx, argIdx+1)
	args = append(args, count, offset)

	var entries []AuditLogEntry
	if err := r.db.SelectContext(ctx, &entries, dataQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return entries, total, nil
}

// GetPatientAuditLog returns audit log entries for a specific patient reference.
func (r *AdminRepository) GetPatientAuditLog(ctx context.Context, patientRef string, count, offset int) ([]AuditLogEntry, int, error) {
	return r.GetAuditLogs(ctx, AuditLogFilters{PatientRef: patientRef}, count, offset)
}

// GetActorAuditLog returns audit log entries for a specific actor.
func (r *AdminRepository) GetActorAuditLog(ctx context.Context, actorID string, count, offset int) ([]AuditLogEntry, int, error) {
	return r.GetAuditLogs(ctx, AuditLogFilters{ActorID: actorID}, count, offset)
}

// GetBreakGlassEvents returns audit log entries where action='break_glass'.
func (r *AdminRepository) GetBreakGlassEvents(ctx context.Context, count, offset int) ([]AuditLogEntry, int, error) {
	return r.GetAuditLogs(ctx, AuditLogFilters{BreakGlassOnly: true}, count, offset)
}

// GetStats returns aggregated platform statistics.
func (r *AdminRepository) GetStats(ctx context.Context) (*AdminStats, error) {
	stats := &AdminStats{}
	stats.FHIRResources.ByType = make(map[string]int)

	// User stats
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE role = 'patient'),
			COUNT(*) FILTER (WHERE role = 'physician'),
			COUNT(*) FILTER (WHERE role = 'admin'),
			COUNT(*) FILTER (WHERE role = 'researcher'),
			COUNT(*) FILTER (WHERE status = 'pending')
		FROM users WHERE deleted_at IS NULL
	`).Scan(
		&stats.Users.Total,
		&stats.Users.Patients,
		&stats.Users.Physicians,
		&stats.Users.Admins,
		&stats.Users.Researchers,
		&stats.Users.Pending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// FHIR resource stats — total
	_ = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM fhir_resources WHERE deleted_at IS NULL",
	).Scan(&stats.FHIRResources.Total)

	// FHIR resource stats — by type
	rows, err := r.db.QueryContext(ctx,
		"SELECT resource_type, COUNT(*) FROM fhir_resources WHERE deleted_at IS NULL GROUP BY resource_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rt string
			var cnt int
			if rows.Scan(&rt, &cnt) == nil {
				stats.FHIRResources.ByType[rt] = cnt
			}
		}
	}

	// Document stats
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status = 'failed'),
			COUNT(*) FILTER (WHERE status = 'pending')
		FROM document_jobs
	`).Scan(
		&stats.Documents.TotalUploaded,
		&stats.Documents.Completed,
		&stats.Documents.Failed,
		&stats.Documents.Pending,
	)

	// Drug interaction stats
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE severity = 'contraindicated'),
			COUNT(*) FILTER (WHERE severity = 'major'),
			COUNT(*) FILTER (WHERE severity = 'moderate')
		FROM drug_interactions
	`).Scan(
		&stats.DrugChecks.Total,
		&stats.DrugChecks.Contraindicated,
		&stats.DrugChecks.Major,
		&stats.DrugChecks.Moderate,
	)

	// Consent stats
	_ = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM consents WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())",
	).Scan(&stats.Consents.ActiveGrants)

	_ = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM consents WHERE revoked_at IS NOT NULL AND revoked_at >= date_trunc('month', NOW())",
	).Scan(&stats.Consents.RevokedThisMonth)

	_ = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM audit_logs WHERE action = 'break_glass' AND created_at >= date_trunc('month', NOW())",
	).Scan(&stats.Consents.BreakGlassThisMonth)

	// Export stats
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed'),
			COUNT(*) FILTER (WHERE status = 'pending')
		FROM research_exports
	`).Scan(
		&stats.Exports.Total,
		&stats.Exports.Completed,
		&stats.Exports.Pending,
	)

	return stats, nil
}
