package repository

import (
	"context"

	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/migrations"
	"strings"
	"testing"
	"time"
)

func Test_OIDCSessionRepository_Lifecycle(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewOIDCSessionRepository(testDB.Pool)
	session := &domain.OIDCSession{AccessToken: strings.Repeat("a", 5000), RefreshToken: strings.Repeat("r", 5000), IDToken: strings.Repeat("i", 5000), Subject: "subject", Email: "user@example.com", Name: "User", ExpiresAt: time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)}
	if err := repo.Create(ctx, "active", session); err != nil {
		t.Fatal(err)
	}
	second := NewOIDCSessionRepository(testDB.Pool)
	got, err := second.Get(ctx, "active")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.ExpiresAt.Equal(session.ExpiresAt) {
		t.Fatal("session expiry did not round trip")
	}
	got.ExpiresAt = session.ExpiresAt
	if *got != *session {
		t.Fatal("session credentials or claims did not round trip")
	}
	if got, err := repo.Get(ctx, "unknown"); err != nil || got != nil {
		t.Fatalf("unknown session: %+v, %v", got, err)
	}
	expired := *session
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	if err := repo.Create(ctx, "expired", &expired); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.Get(ctx, "expired"); err != nil || got != nil {
		t.Fatalf("expired session accepted: %+v, %v", got, err)
	}
	if err := repo.DeleteExpired(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := testDB.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM oidc_sessions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one active session, got %d", count)
	}
	if err := repo.Delete(ctx, "active"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "active"); err != nil {
		t.Fatal(err)
	}
	if got, err := second.Get(ctx, "active"); err != nil || got != nil {
		t.Fatalf("deleted session accepted: %+v, %v", got, err)
	}
}

func Test_OIDCSessionMigration_Rollback(t *testing.T) {
	ctx := context.Background()
	down, err := migrations.FS.ReadFile("000029_oidc_sessions.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := migrations.FS.ReadFile("000029_oidc_sessions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `SELECT expires_at FROM oidc_sessions`); err != nil {
		t.Fatal(err)
	}
}
