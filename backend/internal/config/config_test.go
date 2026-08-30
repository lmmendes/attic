package config

import (
	"os"
	"testing"
)

func Test_Load_SessionSecret_GeneratesEphemeralSecretWhenUnset(t *testing.T) {
	t.Setenv("ATTIC_SESSION_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.SessionSecret) < 32 {
		t.Fatalf("expected a strong generated session secret, got %d characters", len(cfg.SessionSecret))
	}
	if !cfg.SessionSecretEphemeral {
		t.Fatal("expected generated secret to be marked ephemeral")
	}
}

func Test_Load_SessionSecret_UsesConfiguredSecret(t *testing.T) {
	t.Setenv("ATTIC_SESSION_SECRET", "configured-session-secret-with-32-characters")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.SessionSecret != "configured-session-secret-with-32-characters" {
		t.Fatal("expected configured session secret")
	}
	if cfg.SessionSecretEphemeral {
		t.Fatal("expected configured secret to be persistent")
	}
}

func Test_Load_PUID_PGID_NotSet(t *testing.T) {
	// Clear any existing values
	os.Unsetenv("ATTIC_PUID")
	os.Unsetenv("ATTIC_PGID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PUID != nil {
		t.Errorf("expected PUID to be nil, got %v", cfg.PUID)
	}
	if cfg.PGID != nil {
		t.Errorf("expected PGID to be nil, got %v", cfg.PGID)
	}
}

func Test_Load_OIDCAutoRedirect_DefaultsToFalse(t *testing.T) {
	t.Setenv("ATTIC_OIDC_ENABLED", "true")
	t.Setenv("ATTIC_OIDC_ISSUER_URL", "")
	t.Setenv("ATTIC_OIDC_CLIENT_ID", "")
	t.Setenv("ATTIC_OIDC_AUTO_REDIRECT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.OIDCAutoRedirect {
		t.Error("expected OIDC auto redirect to default to false")
	}
}

func Test_Load_OIDCAutoRedirect_ParsesTrue(t *testing.T) {
	t.Setenv("ATTIC_OIDC_ENABLED", "true")
	t.Setenv("ATTIC_OIDC_AUTO_REDIRECT", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.OIDCAutoRedirect {
		t.Error("expected OIDC auto redirect to be true")
	}
}

func Test_Load_OIDCAutoRedirect_IgnoredWhenOIDCDisabled(t *testing.T) {
	t.Setenv("ATTIC_OIDC_ENABLED", "false")
	t.Setenv("ATTIC_OIDC_ISSUER_URL", "")
	t.Setenv("ATTIC_OIDC_CLIENT_ID", "")
	t.Setenv("ATTIC_OIDC_AUTO_REDIRECT", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.OIDCAutoRedirect {
		t.Error("expected OIDC auto redirect to be false when OIDC is disabled")
	}
}

func Test_Load_PUID_PGID_Set(t *testing.T) {
	os.Setenv("ATTIC_PUID", "1000")
	os.Setenv("ATTIC_PGID", "1000")
	defer os.Unsetenv("ATTIC_PUID")
	defer os.Unsetenv("ATTIC_PGID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PUID == nil || *cfg.PUID != 1000 {
		t.Errorf("expected PUID to be 1000, got %v", cfg.PUID)
	}
	if cfg.PGID == nil || *cfg.PGID != 1000 {
		t.Errorf("expected PGID to be 1000, got %v", cfg.PGID)
	}
}

func Test_Load_PUID_PGID_InvalidValues(t *testing.T) {
	os.Setenv("ATTIC_PUID", "invalid")
	os.Setenv("ATTIC_PGID", "not-a-number")
	defer os.Unsetenv("ATTIC_PUID")
	defer os.Unsetenv("ATTIC_PGID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Invalid values should result in nil (silently ignored)
	if cfg.PUID != nil {
		t.Errorf("expected PUID to be nil for invalid value, got %v", cfg.PUID)
	}
	if cfg.PGID != nil {
		t.Errorf("expected PGID to be nil for invalid value, got %v", cfg.PGID)
	}
}

func Test_Load_PUID_Only(t *testing.T) {
	os.Setenv("ATTIC_PUID", "1000")
	os.Unsetenv("ATTIC_PGID")
	defer os.Unsetenv("ATTIC_PUID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PUID == nil || *cfg.PUID != 1000 {
		t.Errorf("expected PUID to be 1000, got %v", cfg.PUID)
	}
	if cfg.PGID != nil {
		t.Errorf("expected PGID to be nil, got %v", cfg.PGID)
	}
}

func Test_Load_PGID_Only(t *testing.T) {
	os.Unsetenv("ATTIC_PUID")
	os.Setenv("ATTIC_PGID", "1000")
	defer os.Unsetenv("ATTIC_PGID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PUID != nil {
		t.Errorf("expected PUID to be nil, got %v", cfg.PUID)
	}
	if cfg.PGID == nil || *cfg.PGID != 1000 {
		t.Errorf("expected PGID to be 1000, got %v", cfg.PGID)
	}
}

func Test_Load_PUID_PGID_Zero(t *testing.T) {
	os.Setenv("ATTIC_PUID", "0")
	os.Setenv("ATTIC_PGID", "0")
	defer os.Unsetenv("ATTIC_PUID")
	defer os.Unsetenv("ATTIC_PGID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Zero is a valid UID/GID (root)
	if cfg.PUID == nil || *cfg.PUID != 0 {
		t.Errorf("expected PUID to be 0, got %v", cfg.PUID)
	}
	if cfg.PGID == nil || *cfg.PGID != 0 {
		t.Errorf("expected PGID to be 0, got %v", cfg.PGID)
	}
}

func Test_HasFileOwnership_BothSet(t *testing.T) {
	puid := 1000
	pgid := 1000
	cfg := &Config{
		PUID: &puid,
		PGID: &pgid,
	}

	if !cfg.HasFileOwnership() {
		t.Error("expected HasFileOwnership to return true when both PUID and PGID are set")
	}
}

func Test_HasFileOwnership_NeitherSet(t *testing.T) {
	cfg := &Config{
		PUID: nil,
		PGID: nil,
	}

	if cfg.HasFileOwnership() {
		t.Error("expected HasFileOwnership to return false when neither PUID nor PGID are set")
	}
}

func Test_HasFileOwnership_OnlyPUID(t *testing.T) {
	puid := 1000
	cfg := &Config{
		PUID: &puid,
		PGID: nil,
	}

	if cfg.HasFileOwnership() {
		t.Error("expected HasFileOwnership to return false when only PUID is set")
	}
}

func Test_HasFileOwnership_OnlyPGID(t *testing.T) {
	pgid := 1000
	cfg := &Config{
		PUID: nil,
		PGID: &pgid,
	}

	if cfg.HasFileOwnership() {
		t.Error("expected HasFileOwnership to return false when only PGID is set")
	}
}
