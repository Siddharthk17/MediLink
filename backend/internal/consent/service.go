package consent

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ConsentService provides additional business logic on top of the engine.
type ConsentService struct {
	engine ConsentEngine
	db     *sqlx.DB
}

func NewConsentService(engine ConsentEngine, db *sqlx.DB) *ConsentService {
	return &ConsentService{engine: engine, db: db}
}

// ValidateProviderExists checks that the provider is a real physician user.
func (s *ConsentService) ValidateProviderExists(ctx context.Context, providerID string) error {
	var role string
	err := s.db.GetContext(ctx, &role, "SELECT role FROM users WHERE id = $1 AND status = 'active' AND deleted_at IS NULL", providerID)
	if err != nil {
		return fmt.Errorf("provider not found")
	}
	if role != "physician" {
		return fmt.Errorf("provider must be a physician")
	}
	return nil
}
