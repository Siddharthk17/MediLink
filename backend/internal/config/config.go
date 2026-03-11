// Package config provides application configuration using Viper.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Encryption EncryptionConfig
	OpenFDA    OpenFDAConfig
	Gemini     GeminiConfig
	Resend     ResendConfig
	Sentry     SentryConfig
	App        AppConfig
	Storage    StorageConfig
	Firebase   FirebaseConfig
}

// StorageConfig holds MinIO object storage settings.
type StorageConfig struct {
	Endpoint  string `mapstructure:"MINIO_ENDPOINT"`
	AccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	SecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	Bucket    string `mapstructure:"MINIO_BUCKET"`
	UseSSL    bool   `mapstructure:"MINIO_USE_SSL"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         string `mapstructure:"PORT"`
	Mode         string `mapstructure:"GIN_MODE"`
	ReadTimeout  int    `mapstructure:"READ_TIMEOUT"`
	WriteTimeout int    `mapstructure:"WRITE_TIMEOUT"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN             string `mapstructure:"DATABASE_URL"`
	MaxOpenConns    int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime int    `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL string `mapstructure:"REDIS_URL"`
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret      string `mapstructure:"JWT_SECRET"`
	ExpiryHours int    `mapstructure:"JWT_EXPIRY_HOURS"`
	RefreshDays int    `mapstructure:"JWT_REFRESH_DAYS"`
}

// EncryptionConfig holds AES-256-GCM encryption settings.
type EncryptionConfig struct {
	Key string `mapstructure:"ENCRYPTION_KEY"`
}

// OpenFDAConfig holds OpenFDA API settings.
type OpenFDAConfig struct {
	BaseURL string `mapstructure:"OPENFDA_BASE_URL"`
	APIKey  string `mapstructure:"OPENFDA_API_KEY"`
}

// GeminiConfig holds Gemini API settings.
type GeminiConfig struct {
	APIKey string `mapstructure:"GEMINI_API_KEY"`
	Model  string `mapstructure:"GEMINI_MODEL"`
}

// FirebaseConfig holds Firebase Admin SDK settings.
type FirebaseConfig struct {
	ServiceAccountPath string `mapstructure:"FIREBASE_SERVICE_ACCOUNT_PATH"`
	ProjectID          string `mapstructure:"FIREBASE_PROJECT_ID"`
}

// ResendConfig holds email notification settings.
type ResendConfig struct {
	APIKey    string `mapstructure:"RESEND_API_KEY"`
	FromEmail string `mapstructure:"RESEND_FROM_EMAIL"`
}

// SentryConfig holds error tracking settings.
type SentryConfig struct {
	DSN string `mapstructure:"SENTRY_DSN"`
}

// AppConfig holds general application settings.
type AppConfig struct {
	Environment string `mapstructure:"APP_ENV"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
}

// Load reads configuration from environment variables and .env file.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read .env file (ignore error if not found)
	_ = v.ReadInConfig()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	cfg := &Config{}

	cfg.Server.Port = v.GetString("PORT")
	cfg.Server.Mode = v.GetString("GIN_MODE")
	cfg.Server.ReadTimeout = v.GetInt("READ_TIMEOUT")
	cfg.Server.WriteTimeout = v.GetInt("WRITE_TIMEOUT")

	cfg.Database.DSN = v.GetString("DATABASE_URL")
	cfg.Database.MaxOpenConns = v.GetInt("DB_MAX_OPEN_CONNS")
	cfg.Database.MaxIdleConns = v.GetInt("DB_MAX_IDLE_CONNS")
	cfg.Database.ConnMaxLifetime = v.GetInt("DB_CONN_MAX_LIFETIME")

	cfg.Redis.URL = v.GetString("REDIS_URL")

	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.JWT.ExpiryHours = v.GetInt("JWT_EXPIRY_HOURS")
	cfg.JWT.RefreshDays = v.GetInt("JWT_REFRESH_DAYS")

	cfg.Encryption.Key = v.GetString("ENCRYPTION_KEY")

	cfg.OpenFDA.BaseURL = v.GetString("OPENFDA_BASE_URL")
	cfg.OpenFDA.APIKey = v.GetString("OPENFDA_API_KEY")

	cfg.Gemini.APIKey = v.GetString("GEMINI_API_KEY")
	cfg.Gemini.Model = v.GetString("GEMINI_MODEL")

	cfg.Resend.APIKey = v.GetString("RESEND_API_KEY")
	cfg.Resend.FromEmail = v.GetString("RESEND_FROM_EMAIL")

	cfg.Firebase.ServiceAccountPath = v.GetString("FIREBASE_SERVICE_ACCOUNT_PATH")
	cfg.Firebase.ProjectID = v.GetString("FIREBASE_PROJECT_ID")

	cfg.Sentry.DSN = v.GetString("SENTRY_DSN")

	cfg.Storage.Endpoint = v.GetString("MINIO_ENDPOINT")
	cfg.Storage.AccessKey = v.GetString("MINIO_ACCESS_KEY")
	cfg.Storage.SecretKey = v.GetString("MINIO_SECRET_KEY")
	cfg.Storage.Bucket = v.GetString("MINIO_BUCKET")
	cfg.Storage.UseSSL = v.GetBool("MINIO_USE_SSL")

	cfg.App.Environment = v.GetString("APP_ENV")
	cfg.App.LogLevel = v.GetString("LOG_LEVEL")

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("PORT", "8080")
	v.SetDefault("GIN_MODE", "debug")
	v.SetDefault("READ_TIMEOUT", 30)
	v.SetDefault("WRITE_TIMEOUT", 30)
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 5)
	v.SetDefault("DB_CONN_MAX_LIFETIME", 300)
	v.SetDefault("JWT_EXPIRY_HOURS", 2)
	v.SetDefault("JWT_REFRESH_DAYS", 7)
	v.SetDefault("OPENFDA_BASE_URL", "https://api.fda.gov")
	v.SetDefault("GEMINI_MODEL", "gemini-1.5-flash")
	v.SetDefault("FIREBASE_SERVICE_ACCOUNT_PATH", "./firebase-adminsdk.json")
	v.SetDefault("FIREBASE_PROJECT_ID", "")
	v.SetDefault("MINIO_ENDPOINT", "minio:9000")
	v.SetDefault("MINIO_ACCESS_KEY", "MediLink")
	v.SetDefault("MINIO_SECRET_KEY", "MediLink_minio_dev")
	v.SetDefault("MINIO_BUCKET", "medilink-lab-reports")
	v.SetDefault("MINIO_USE_SSL", false)
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("LOG_LEVEL", "debug")
}

func validate(cfg *Config) error {
	if cfg.Database.DSN == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.Encryption.Key == "" {
		return fmt.Errorf("config: ENCRYPTION_KEY is required")
	}
	if len(cfg.Encryption.Key) != 64 {
		return fmt.Errorf("config: ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes), got %d", len(cfg.Encryption.Key))
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}
	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("config: JWT_SECRET must be at least 32 characters for HMAC-SHA256 security, got %d", len(cfg.JWT.Secret))
	}
	return nil
}
