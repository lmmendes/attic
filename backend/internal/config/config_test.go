package config

import (
	"os"
	"testing"
)

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
