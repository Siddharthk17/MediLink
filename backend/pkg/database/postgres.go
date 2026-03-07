// Package database provides PostgreSQL connection setup using GORM and sqlx.
package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresConnections holds both GORM and sqlx database connections.
type PostgresConnections struct {
	GORM *gorm.DB
	SQLX *sqlx.DB
}

// NewPostgresConnections creates new PostgreSQL connections using GORM and sqlx.
func NewPostgresConnections(dsn string, maxOpen, maxIdle, maxLifetimeSecs int) (*PostgresConnections, error) {
	// GORM connection
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres via GORM: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetimeSecs) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres via GORM: %w", err)
	}

	// sqlx connection
	sqlxDB, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres via sqlx: %w", err)
	}

	sqlxDB.SetMaxOpenConns(maxOpen)
	sqlxDB.SetMaxIdleConns(maxIdle)
	sqlxDB.SetConnMaxLifetime(time.Duration(maxLifetimeSecs) * time.Second)

	log.Info().Msg("PostgreSQL connections established (GORM + sqlx)")

	return &PostgresConnections{
		GORM: gormDB,
		SQLX: sqlxDB,
	}, nil
}

// Close closes both database connections.
func (p *PostgresConnections) Close() error {
	sqlDB, err := p.GORM.DB()
	if err == nil {
		sqlDB.Close()
	}
	p.SQLX.Close()
	log.Info().Msg("PostgreSQL connections closed")
	return nil
}

// HealthCheck verifies the database connection is alive.
func (p *PostgresConnections) HealthCheck() error {
	sqlDB, err := p.GORM.DB()
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return sqlDB.Ping()
}
