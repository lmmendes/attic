package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	S3Endpoint    string
	S3Bucket      string
	S3Region      string
	S3AccessKey   string
	S3SecretKey   string
	OIDCIssuer    string
	OIDCClientID  string
	AuthDisabled  bool
	CORSOrigins   string
	BaseURL       string
	SessionSecret string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("ATTIC_PORT", "8080"),
		DatabaseURL:   getEnv("ATTIC_DATABASE_URL", "postgres://attic:attic@localhost:5432/attic?sslmode=disable"),
		S3Endpoint:    getEnv("ATTIC_S3_ENDPOINT", "http://localhost:4566"),
		S3Bucket:      getEnv("ATTIC_S3_BUCKET", "attic-attachments"),
		S3Region:      getEnv("ATTIC_S3_REGION", "us-east-1"),
		S3AccessKey:   getEnv("ATTIC_S3_ACCESS_KEY", "test"),
		S3SecretKey:   getEnv("ATTIC_S3_SECRET_KEY", "test"),
		OIDCIssuer:    getEnv("ATTIC_OIDC_ISSUER", "http://localhost:8180/realms/attic"),
		OIDCClientID:  getEnv("ATTIC_OIDC_CLIENT_ID", "attic-web"),
		AuthDisabled:  getEnv("ATTIC_AUTH_DISABLED", "false") == "true",
		CORSOrigins:   getEnv("ATTIC_CORS_ORIGINS", "http://localhost:3000"),
		BaseURL:       getEnv("ATTIC_BASE_URL", "http://localhost:8080"),
		SessionSecret: getEnv("ATTIC_SESSION_SECRET", "change-me-in-production-32chars!"),
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
