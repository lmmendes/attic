package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	S3Endpoint    string
	S3Bucket      string
	S3Region      string
	S3AccessKey   string
	S3SecretKey   string
	OIDCEnabled   bool
	OIDCIssuer    string
	OIDCClientID  string
	AuthDisabled  bool
	CORSOrigins   string
	BaseURL       string
	SessionSecret string

	// Storage settings
	LocalStoragePath string // Path for local file storage (used when S3 is not configured)
	PUID             *int   // User ID for file ownership (nil = don't change ownership)
	PGID             *int   // Group ID for file ownership (nil = don't change ownership)

	// Auth settings
	AdminEmail           string
	AdminPassword        string
	SessionDurationHours int
	PasswordMinLength    int
}

// UseS3Storage returns true if S3 credentials are configured
func (c *Config) UseS3Storage() bool {
	return c.S3AccessKey != "" && c.S3SecretKey != ""
}

// HasFileOwnership returns true if PUID and PGID are configured
func (c *Config) HasFileOwnership() bool {
	return c.PUID != nil && c.PGID != nil
}

func Load() (*Config, error) {
	sessionHours, _ := strconv.Atoi(getEnv("ATTIC_SESSION_DURATION_HOURS", "24"))
	if sessionHours <= 0 {
		sessionHours = 24
	}

	passwordMinLength, _ := strconv.Atoi(getEnv("ATTIC_PASSWORD_MIN_LENGTH", "8"))
	if passwordMinLength <= 0 {
		passwordMinLength = 8
	}

	// Parse optional PUID/PGID for file ownership
	var puid, pgid *int
	if puidStr := os.Getenv("ATTIC_PUID"); puidStr != "" {
		if val, err := strconv.Atoi(puidStr); err == nil {
			puid = &val
		}
	}
	if pgidStr := os.Getenv("ATTIC_PGID"); pgidStr != "" {
		if val, err := strconv.Atoi(pgidStr); err == nil {
			pgid = &val
		}
	}

	cfg := &Config{
		Port:          getEnv("ATTIC_PORT", "8080"),
		DatabaseURL:   getEnv("ATTIC_DATABASE_URL", "postgres://attic:attic@localhost:5432/attic?sslmode=disable"),
		S3Endpoint:    getEnv("ATTIC_S3_ENDPOINT", "http://localhost:4566"),
		S3Bucket:      getEnv("ATTIC_S3_BUCKET", "attic-attachments"),
		S3Region:      getEnv("ATTIC_S3_REGION", "us-east-1"),
		S3AccessKey:   getEnv("ATTIC_S3_ACCESS_KEY", ""),  // Empty = use local storage
		S3SecretKey:   getEnv("ATTIC_S3_SECRET_KEY", ""),  // Empty = use local storage
		OIDCEnabled:   getEnv("ATTIC_OIDC_ENABLED", "false") == "true",
		OIDCIssuer:    getEnv("ATTIC_OIDC_ISSUER", "http://localhost:8180/realms/attic"),
		OIDCClientID:  getEnv("ATTIC_OIDC_CLIENT_ID", "attic-web"),
		AuthDisabled:  getEnv("ATTIC_AUTH_DISABLED", "false") == "true",
		CORSOrigins:   getEnv("ATTIC_CORS_ORIGINS", "http://localhost:3000"),
		BaseURL:       getEnv("ATTIC_BASE_URL", "http://localhost:8080"),
		SessionSecret: getEnv("ATTIC_SESSION_SECRET", "change-me-in-production-32chars!"),

		LocalStoragePath: getEnv("ATTIC_LOCAL_STORAGE_PATH", "./uploads"),
		PUID:             puid,
		PGID:             pgid,

		AdminEmail:           getEnv("ATTIC_ADMIN_EMAIL", "admin"),
		AdminPassword:        getEnv("ATTIC_ADMIN_PASSWORD", "admin"),
		SessionDurationHours: sessionHours,
		PasswordMinLength:    passwordMinLength,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("ATTIC_DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
