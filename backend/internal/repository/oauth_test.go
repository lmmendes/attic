package repository

import (
	"context"
	"testing"
	"time"

	"github.com/lmmendes/attic/internal/testutil"
)

func Test_OAuthRepository_AuthorizationCode_IsSingleUse(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "OAuth Test")
	user, _ := fixtures.CreateUser(ctx, org.ID, "oauth@example.com")
	repo := NewOAuthRepository(testDB.Pool)

	err := repo.CreateAuthorizationCode(ctx, "raw-code", AuthorizationCode{
		UserID: user.ID, ClientID: "attic-mobile", RedirectURI: "app:/callback",
		CodeChallenge: "challenge", Scope: "attic",
	}, time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatalf("create authorization code: %v", err)
	}

	code, err := repo.ConsumeAuthorizationCode(ctx, "raw-code")
	if err != nil || code == nil {
		t.Fatalf("consume authorization code: code=%v err=%v", code, err)
	}
	code, err = repo.ConsumeAuthorizationCode(ctx, "raw-code")
	if err != nil {
		t.Fatalf("consume code a second time: %v", err)
	}
	if code != nil {
		t.Fatal("expected consumed authorization code to be rejected")
	}
}

func Test_OAuthRepository_RefreshTokenReuse_RevokesSession(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "OAuth Test")
	user, _ := fixtures.CreateUser(ctx, org.ID, "oauth@example.com")
	repo := NewOAuthRepository(testDB.Pool)
	now := time.Now().UTC()
	if err := repo.CreateSession(ctx, user.ID, "attic-mobile", "attic", "access-1", "refresh-1", now.Add(time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := repo.RotateRefreshToken(ctx, "refresh-1", "access-2", "refresh-2", now.Add(time.Minute)); err != nil {
		t.Fatalf("rotate refresh token: %v", err)
	}
	if _, err := repo.RotateRefreshToken(ctx, "refresh-1", "attacker-access", "attacker-refresh", now.Add(time.Minute)); err == nil {
		t.Fatal("expected refresh token replay to be detected")
	}
	userByToken, err := repo.GetUserByAccessToken(ctx, "access-2")
	if err != nil {
		t.Fatalf("look up access token: %v", err)
	}
	if userByToken != nil {
		t.Fatal("expected replay to revoke the token family")
	}
}
