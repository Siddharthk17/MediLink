package admin

import (
	"context"

	"github.com/rs/zerolog"
)

// AdminService provides business logic for admin operations.
type AdminService struct {
	repo   *AdminRepository
	logger zerolog.Logger
}

// NewAdminService creates a new AdminService.
func NewAdminService(repo *AdminRepository, logger zerolog.Logger) *AdminService {
	return &AdminService{
		repo:   repo,
		logger: logger.With().Str("component", "admin-service").Logger(),
	}
}

// ListUsers returns paginated users.
func (s *AdminService) ListUsers(ctx context.Context, filters UserFilters, count, offset int) ([]UserSummary, int, error) {
	users, total, err := s.repo.ListUsers(ctx, filters, count, offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list users")
		return nil, 0, err
	}
	return users, total, nil
}

// GetUser returns a single user by ID.
func (s *AdminService) GetUser(ctx context.Context, userID string) (*UserDetail, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("userId", userID).Msg("failed to get user")
		return nil, err
	}
	return user, nil
}

// UpdateUserRole changes a user's role.
func (s *AdminService) UpdateUserRole(ctx context.Context, userID, newRole string) error {
	validRoles := map[string]bool{
		"patient": true, "physician": true, "admin": true, "researcher": true,
	}
	if !validRoles[newRole] {
		return &ValidationError{Message: "invalid role: must be patient, physician, admin, or researcher"}
	}
	return s.repo.UpdateUserRole(ctx, userID, newRole)
}

// InviteResearcher creates a pending researcher invitation.
func (s *AdminService) InviteResearcher(ctx context.Context, email, invitedBy string) error {
	if email == "" {
		return &ValidationError{Message: "email is required"}
	}
	return s.repo.InviteResearcher(ctx, email, invitedBy)
}

// SuspendPhysician sets physician status to suspended.
func (s *AdminService) SuspendPhysician(ctx context.Context, userID string) error {
	return s.repo.SuspendPhysician(ctx, userID)
}

// ReinstatePhysician sets physician status to active.
func (s *AdminService) ReinstatePhysician(ctx context.Context, userID string) error {
	return s.repo.ReinstatePhysician(ctx, userID)
}

// ListDoctors returns active physicians for the directory.
func (s *AdminService) ListDoctors(ctx context.Context, specialization string) ([]DoctorSummary, error) {
	doctors, err := s.repo.ListDoctors(ctx, specialization)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list doctors")
		return nil, err
	}
	return doctors, nil
}

// GetAuditLogs returns paginated audit log entries.
func (s *AdminService) GetAuditLogs(ctx context.Context, filters AuditLogFilters, count, offset int) ([]AuditLogEntry, int, error) {
	return s.repo.GetAuditLogs(ctx, filters, count, offset)
}

// GetPatientAuditLog returns audit logs for a patient.
func (s *AdminService) GetPatientAuditLog(ctx context.Context, patientRef string, count, offset int) ([]AuditLogEntry, int, error) {
	return s.repo.GetPatientAuditLog(ctx, patientRef, count, offset)
}

// GetActorAuditLog returns audit logs for an actor.
func (s *AdminService) GetActorAuditLog(ctx context.Context, actorID string, count, offset int) ([]AuditLogEntry, int, error) {
	return s.repo.GetActorAuditLog(ctx, actorID, count, offset)
}

// GetBreakGlassEvents returns break-glass audit log entries.
func (s *AdminService) GetBreakGlassEvents(ctx context.Context, count, offset int) ([]AuditLogEntry, int, error) {
	return s.repo.GetBreakGlassEvents(ctx, count, offset)
}

// GetStats returns aggregated platform statistics.
func (s *AdminService) GetStats(ctx context.Context) (*AdminStats, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get platform stats")
		return nil, err
	}
	return stats, nil
}

// Ping checks database connectivity.
func (s *AdminService) Ping(ctx context.Context) error {
	return s.repo.db.PingContext(ctx)
}

// ValidationError is a domain error for invalid input.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
