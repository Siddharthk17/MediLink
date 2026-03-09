// Package audit provides immutable audit logging for FHIR resource access.
package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	UserID        *uuid.UUID
	UserRole      string
	UserEmailHash string
	ResourceType  string
	ResourceID    string
	Action        string // create, read, update, delete, search
	PatientRef    string
	Purpose       string // treatment, research, admin, break-glass
	Success       bool
	StatusCode    int
	ErrorMessage  string
	IPAddress     string
	UserAgent     string
	RequestID     string
}

// AuditLogger defines the audit logging interface.
type AuditLogger interface {
	// Log writes an audit entry synchronously — used for security-critical events.
	Log(ctx context.Context, entry AuditEntry) error
	// LogAsync writes to a buffered channel to avoid blocking the request.
	LogAsync(entry AuditEntry)
	// Close flushes remaining entries and shuts down the async worker.
	Close()
}

// PostgresAuditLogger implements AuditLogger backed by PostgreSQL.
type PostgresAuditLogger struct {
	db     *sqlx.DB
	ch     chan AuditEntry
	done   chan struct{}
}

// NewPostgresAuditLogger creates a new audit logger with a buffered async channel.
func NewPostgresAuditLogger(db *sqlx.DB) *PostgresAuditLogger {
	al := &PostgresAuditLogger{
		db:   db,
		ch:   make(chan AuditEntry, 1000),
		done: make(chan struct{}),
	}
	go al.worker()
	return al
}

const insertAuditSQL = `
INSERT INTO audit_logs (
	user_id, user_role, user_email_hash,
	resource_type, resource_id, action,
	patient_ref, purpose,
	success, status_code, error_message,
	ip_address, user_agent, request_id
) VALUES (
	$1, $2, $3,
	$4, $5, $6,
	$7, $8,
	$9, $10, $11,
	$12, $13, $14
)`

// nullableIP converts an empty IP string to a sql.NullString so Postgres
// receives NULL instead of an invalid empty-string INET value.
func nullableIP(ip string) interface{} {
	if ip == "" {
		return sql.NullString{}
	}
	return ip
}

// Log writes an audit entry synchronously.
func (a *PostgresAuditLogger) Log(ctx context.Context, entry AuditEntry) error {
	_, err := a.db.ExecContext(ctx, insertAuditSQL,
		entry.UserID, entry.UserRole, entry.UserEmailHash,
		entry.ResourceType, entry.ResourceID, entry.Action,
		entry.PatientRef, entry.Purpose,
		entry.Success, entry.StatusCode, entry.ErrorMessage,
		nullableIP(entry.IPAddress), entry.UserAgent, entry.RequestID,
	)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}
	return nil
}

// LogAsync enqueues an audit entry for asynchronous writing.
func (a *PostgresAuditLogger) LogAsync(entry AuditEntry) {
	select {
	case a.ch <- entry:
	default:
		log.Error().Str("action", entry.Action).Msg("audit log channel full, dropping entry")
	}
}

// Close flushes remaining entries and shuts down the async worker.
func (a *PostgresAuditLogger) Close() {
	close(a.ch)
	<-a.done
}

// worker processes buffered audit entries, flushing every 100ms or at 50 entries.
func (a *PostgresAuditLogger) worker() {
	defer close(a.done)

	batch := make([]AuditEntry, 0, 50)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-a.ch:
			if !ok {
				// Channel closed, flush remaining
				if len(batch) > 0 {
					a.flushBatch(batch)
				}
				return
			}
			batch = append(batch, entry)
			if len(batch) >= 50 {
				a.flushBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				a.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (a *PostgresAuditLogger) flushBatch(entries []AuditEntry) {
	for _, entry := range entries {
		_, err := a.db.Exec(insertAuditSQL,
			entry.UserID, entry.UserRole, entry.UserEmailHash,
			entry.ResourceType, entry.ResourceID, entry.Action,
			entry.PatientRef, entry.Purpose,
			entry.Success, entry.StatusCode, entry.ErrorMessage,
			nullableIP(entry.IPAddress), entry.UserAgent, entry.RequestID,
		)
		if err != nil {
			log.Error().Err(err).Str("action", entry.Action).Msg("failed to flush audit entry")
		}
	}
}
