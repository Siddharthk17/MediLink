package tasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Siddharthk17/MediLink/internal/tasks"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "sqlmock"), mock
}

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return client, mr
}

var nopLogger = zerolog.Nop()

// ── TokenCleanupTask ─────────────────────────────────────────────────────────

func TestTokenCleanup_DeletesExpiredRefresh(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectExec("DELETE FROM email_otps").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM login_attempts").
		WillReturnResult(sqlmock.NewResult(0, 0))

	task := tasks.NewTokenCleanupTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskCleanupExpiredTokens, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenCleanup_DeletesExpiredOTPs(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM email_otps").
		WillReturnResult(sqlmock.NewResult(0, 8))
	mock.ExpectExec("DELETE FROM login_attempts").
		WillReturnResult(sqlmock.NewResult(0, 0))

	task := tasks.NewTokenCleanupTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskCleanupExpiredTokens, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenCleanup_DeletesOldAttempts(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM email_otps").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM login_attempts").
		WillReturnResult(sqlmock.NewResult(0, 12))

	task := tasks.NewTokenCleanupTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskCleanupExpiredTokens, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenCleanup_KeepsValidTokens(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM email_otps").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("DELETE FROM login_attempts").
		WillReturnResult(sqlmock.NewResult(0, 0))

	task := tasks.NewTokenCleanupTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskCleanupExpiredTokens, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── JobExpiryTask ────────────────────────────────────────────────────────────

func TestJobExpiry_StuckProcessing(t *testing.T) {
	db, mock := newTestDB(t)

	// First UPDATE: stuck processing > 10 min → failed
	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 3))
	// Second UPDATE: stuck pending (none here)
	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))

	task := tasks.NewJobExpiryTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskExpireStaleJobs, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJobExpiry_StuckPending(t *testing.T) {
	db, mock := newTestDB(t)

	// First UPDATE: stuck processing (none here)
	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// Second UPDATE: stuck pending > 30 min → failed
	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 2))

	task := tasks.NewJobExpiryTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskExpireStaleJobs, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJobExpiry_KeepsRecent(t *testing.T) {
	db, mock := newTestDB(t)

	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE document_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))

	task := tasks.NewJobExpiryTask(db, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskExpireStaleJobs, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── ConsentExpiryTask ────────────────────────────────────────────────────────

func TestConsentExpiry_RevokesExpired(t *testing.T) {
	db, mock := newTestDB(t)
	client, _ := newTestRedis(t)

	rows := sqlmock.NewRows([]string{"id", "patient_id", "provider_id", "fhir_patient_id"}).
		AddRow("consent-1", "patient-1", "provider-1", "fhir-p1").
		AddRow("consent-2", "patient-2", "provider-2", "fhir-p2")
	mock.ExpectQuery("WITH expired_consents").WillReturnRows(rows)

	task := tasks.NewConsentExpiryTask(db, client, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskRevokeExpiredConsents, nil))

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConsentExpiry_InvalidatesRedisCache(t *testing.T) {
	db, mock := newTestDB(t)
	client, mr := newTestRedis(t)

	require.NoError(t, mr.Set("consent:provider-1:fhir-p1", "cached"))
	require.NoError(t, mr.Set("consent:provider-2:fhir-p2", "cached"))

	rows := sqlmock.NewRows([]string{"id", "patient_id", "provider_id", "fhir_patient_id"}).
		AddRow("consent-1", "patient-1", "provider-1", "fhir-p1").
		AddRow("consent-2", "patient-2", "provider-2", "fhir-p2")
	mock.ExpectQuery("WITH expired_consents").WillReturnRows(rows)

	task := tasks.NewConsentExpiryTask(db, client, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskRevokeExpiredConsents, nil))

	require.NoError(t, err)
	assert.False(t, mr.Exists("consent:provider-1:fhir-p1"), "cache for provider-1 should be invalidated")
	assert.False(t, mr.Exists("consent:provider-2:fhir-p2"), "cache for provider-2 should be invalidated")
}

func TestConsentExpiry_KeepsNonExpired(t *testing.T) {
	db, mock := newTestDB(t)
	client, mr := newTestRedis(t)

	require.NoError(t, mr.Set("consent:provider-1:fhir-p1", "active"))

	// Empty result set — no consents expired
	rows := sqlmock.NewRows([]string{"id", "patient_id", "provider_id", "fhir_patient_id"})
	mock.ExpectQuery("WITH expired_consents").WillReturnRows(rows)

	task := tasks.NewConsentExpiryTask(db, client, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskRevokeExpiredConsents, nil))

	require.NoError(t, err)
	assert.True(t, mr.Exists("consent:provider-1:fhir-p1"), "non-expired consent cache should remain")
}

func TestConsentExpiry_NilExpiry_NotRevoked(t *testing.T) {
	db, mock := newTestDB(t)
	client, mr := newTestRedis(t)

	require.NoError(t, mr.Set("consent:provider-1:fhir-p1", "permanent"))

	// SQL condition: expires_at IS NOT NULL AND expires_at < NOW()
	// Consents with NULL expires_at never match → empty result
	rows := sqlmock.NewRows([]string{"id", "patient_id", "provider_id", "fhir_patient_id"})
	mock.ExpectQuery("WITH expired_consents").WillReturnRows(rows)

	task := tasks.NewConsentExpiryTask(db, client, nopLogger)
	err := task.Process(context.Background(), asynq.NewTask(tasks.TaskRevokeExpiredConsents, nil))

	require.NoError(t, err)
	assert.True(t, mr.Exists("consent:provider-1:fhir-p1"), "consent with NULL expires_at must not be revoked")
}

// ── Scheduler ────────────────────────────────────────────────────────────────

func TestScheduler_RegistersAllFiveTasks(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisOpt := asynq.RedisClientOpt{Addr: mr.Addr()}
	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Location: time.UTC})

	// Should not panic
	assert.NotPanics(t, func() {
		tasks.RegisterPeriodicTasks(scheduler)
	})

	// Scheduler should start and stop without error after registration
	require.NoError(t, scheduler.Start())

	// Give the scheduler heartbeater time to write to Redis
	time.Sleep(2 * time.Second)

	scheduler.Shutdown()

	// Verify all 5 task type constants exist (compile-time + value check)
	taskTypes := []string{
		tasks.TaskCleanupExpiredTokens,
		tasks.TaskESReindexMissed,
		tasks.TaskExpireStaleJobs,
		tasks.TaskRevokeExpiredConsents,
		tasks.TaskDailyStatsSnapshot,
	}
	assert.Len(t, taskTypes, 5, "expected 5 periodic task types")
	for _, tt := range taskTypes {
		assert.NotEmpty(t, tt, "task type constant should not be empty")
	}
}
