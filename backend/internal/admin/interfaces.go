package admin

import "context"

// Service defines the business logic interface for admin operations.
type Service interface {
	ListUsers(ctx context.Context, filters UserFilters, count, offset int) ([]UserSummary, int, error)
	GetUser(ctx context.Context, userID string) (*UserDetail, error)
	UpdateUserRole(ctx context.Context, userID, newRole string) error
	InviteResearcher(ctx context.Context, email, invitedBy string) error
	SuspendPhysician(ctx context.Context, userID string) error
	ReinstatePhysician(ctx context.Context, userID string) error
	GetAuditLogs(ctx context.Context, filters AuditLogFilters, count, offset int) ([]AuditLogEntry, int, error)
	GetPatientAuditLog(ctx context.Context, patientRef string, count, offset int) ([]AuditLogEntry, int, error)
	GetActorAuditLog(ctx context.Context, actorID string, count, offset int) ([]AuditLogEntry, int, error)
	GetBreakGlassEvents(ctx context.Context, count, offset int) ([]AuditLogEntry, int, error)
	GetStats(ctx context.Context) (*AdminStats, error)
	Ping(ctx context.Context) error
}

// DuplicateError indicates a uniqueness constraint violation.
type DuplicateError struct {
	Message string
}

func (e *DuplicateError) Error() string {
	return e.Message
}
