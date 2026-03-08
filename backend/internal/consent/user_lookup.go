package consent

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type sqlUserLookup struct {
	db *sqlx.DB
}

// NewSQLUserLookup returns a UserLookup backed by direct SQL queries.
func NewSQLUserLookup(db *sqlx.DB) UserLookup {
	return &sqlUserLookup{db: db}
}

func (s *sqlUserLookup) GetUserContactInfo(ctx context.Context, userID uuid.UUID) (string, string, error) {
	var info struct {
		Email string `db:"email"`
	}
	err := s.db.GetContext(ctx, &info, "SELECT email FROM users WHERE id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		return "", "", err
	}
	return info.Email, "", nil
}
