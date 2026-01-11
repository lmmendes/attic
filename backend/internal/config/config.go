package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port         string
	DatabaseURL  string
	S3Endpoint   string
	S3Bucket     string
	S3Region     string
	S3AccessKey  string
	S3SecretKey  string
	OIDCIssuer   string
	OIDCClientID string
	AuthDisabled bool
	CORSOrigins  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://attic:attic@localhost:5432/attic?sslmode=disable"),
		S3Endpoint:   getEnv("S3_ENDPOINT", "http://localhost:4566"),
		S3Bucket:     getEnv("S3_BUCKET", "attic-attachments"),
		S3Region:     getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:  getEnv("S3_ACCESS_KEY", "test"),
		S3SecretKey:  getEnv("S3_SECRET_KEY", "test"),
		OIDCIssuer:   getEnv("OIDC_ISSUER", "http://localhost:8180/realms/attic"),
		OIDCClientID: getEnv("OIDC_CLIENT_ID", "attic-web"),
		AuthDisabled: getEnv("AUTH_DISABLED", "false") == "true",
		CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:3000"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
