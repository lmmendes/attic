package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_UserRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email to be lowercase, got '%s'", user.Email)
	}
}

func Test_UserRepository_Create_NormalizesEmail(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "Test@EXAMPLE.COM",
		Role:           domain.UserRoleUser,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email to be normalized to lowercase, got '%s'", user.Email)
	}
}

func Test_UserRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	fetched, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to get by ID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected user to be found")
	}
	if fetched.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", fetched.Email)
	}
}

func Test_UserRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewUserRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent user")
	}
}

func Test_UserRepository_GetByEmail_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	// Search with different case
	fetched, err := repo.GetByEmail(ctx, "TEST@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("failed to get by email: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected user to be found")
	}
	if fetched.ID != user.ID {
		t.Error("expected same user to be found")
	}
}

func Test_UserRepository_GetByOIDCSubject(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	oidcSubject := "oidc-subject-123"
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "test@example.com",
		OIDCSubject:    &oidcSubject,
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	fetched, err := repo.GetByOIDCSubject(ctx, oidcSubject)
	if err != nil {
		t.Fatalf("failed to get by OIDC subject: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected user to be found")
	}
	if fetched.ID != user.ID {
		t.Error("expected same user")
	}
}

func Test_UserRepository_List_ReturnsUsersForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org1, _ := fixtures.CreateOrganization(ctx, "Org 1")
	org2, _ := fixtures.CreateOrganization(ctx, "Org 2")

	repo := NewUserRepository(testDB.Pool)

	// Create users in org1
	repo.Create(ctx, &domain.User{OrganizationID: org1.ID, Email: "user1@org1.com", Role: domain.UserRoleUser})
	repo.Create(ctx, &domain.User{OrganizationID: org1.ID, Email: "user2@org1.com", Role: domain.UserRoleUser})

	// Create user in org2
	repo.Create(ctx, &domain.User{OrganizationID: org2.ID, Email: "user1@org2.com", Role: domain.UserRoleUser})

	users, err := repo.List(ctx, org1.ID)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users for org1, got %d", len(users))
	}
}

func Test_UserRepository_List_OrderedByEmail(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	repo.Create(ctx, &domain.User{OrganizationID: org.ID, Email: "zebra@example.com", Role: domain.UserRoleUser})
	repo.Create(ctx, &domain.User{OrganizationID: org.ID, Email: "alpha@example.com", Role: domain.UserRoleUser})
	repo.Create(ctx, &domain.User{OrganizationID: org.ID, Email: "beta@example.com", Role: domain.UserRoleUser})

	users, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if users[0].Email != "alpha@example.com" {
		t.Error("expected users to be ordered by email")
	}
}

func Test_UserRepository_Count(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	repo.Create(ctx, &domain.User{OrganizationID: org.ID, Email: "user1@example.com", Role: domain.UserRoleUser})
	repo.Create(ctx, &domain.User{OrganizationID: org.ID, Email: "user2@example.com", Role: domain.UserRoleUser})

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func Test_UserRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "original@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	// Update
	displayName := "John Doe"
	user.Email = "updated@example.com"
	user.DisplayName = &displayName
	user.Role = domain.UserRoleAdmin

	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, user.ID)
	if fetched.Email != "updated@example.com" {
		t.Errorf("expected email 'updated@example.com', got '%s'", fetched.Email)
	}
	if fetched.DisplayName == nil || *fetched.DisplayName != "John Doe" {
		t.Error("expected display name to be updated")
	}
	if fetched.Role != domain.UserRoleAdmin {
		t.Error("expected role to be admin")
	}
}

func Test_UserRepository_UpdatePassword(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "user@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	// Update password
	err := repo.UpdatePassword(ctx, user.ID, "hashed-password")
	if err != nil {
		t.Fatalf("failed to update password: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, user.ID)
	if fetched.PasswordHash == nil || *fetched.PasswordHash != "hashed-password" {
		t.Error("expected password hash to be updated")
	}
}

func Test_UserRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "user@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	// Delete
	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Should not be found
	fetched, _ := repo.GetByID(ctx, user.ID)
	if fetched != nil {
		t.Error("expected deleted user not to be found")
	}
}

func Test_UserRepository_LinkOIDC(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user := &domain.User{
		OrganizationID: org.ID,
		Email:          "user@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, user)

	// Link OIDC
	err := repo.LinkOIDC(ctx, user.ID, "oidc-subject-123")
	if err != nil {
		t.Fatalf("failed to link OIDC: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, user.ID)
	if fetched.OIDCSubject == nil || *fetched.OIDCSubject != "oidc-subject-123" {
		t.Error("expected OIDC subject to be linked")
	}
}

func Test_UserRepository_GetOrCreate_CreatesNewUser(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	user, created, err := repo.GetOrCreate(ctx, org.ID, "oidc-subject", "new@example.com", "New User")
	if err != nil {
		t.Fatalf("failed to get or create: %v", err)
	}
	if !created {
		t.Error("expected user to be created")
	}
	if user.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got '%s'", user.Email)
	}
}

func Test_UserRepository_GetOrCreate_FindsExistingBySubject(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	oidcSubject := "oidc-subject"
	existing := &domain.User{
		OrganizationID: org.ID,
		Email:          "existing@example.com",
		OIDCSubject:    &oidcSubject,
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, existing)

	user, created, err := repo.GetOrCreate(ctx, org.ID, "oidc-subject", "different@example.com", "")
	if err != nil {
		t.Fatalf("failed to get or create: %v", err)
	}
	if created {
		t.Error("expected existing user to be found")
	}
	if user.ID != existing.ID {
		t.Error("expected same user")
	}
}

func Test_UserRepository_GetOrCreate_LinksExistingByEmail(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewUserRepository(testDB.Pool)
	existing := &domain.User{
		OrganizationID: org.ID,
		Email:          "existing@example.com",
		Role:           domain.UserRoleUser,
	}
	repo.Create(ctx, existing)

	user, created, err := repo.GetOrCreate(ctx, org.ID, "new-oidc-subject", "existing@example.com", "")
	if err != nil {
		t.Fatalf("failed to get or create: %v", err)
	}
	if created {
		t.Error("expected existing user to be found")
	}
	if user.ID != existing.ID {
		t.Error("expected same user")
	}
	if user.OIDCSubject == nil || *user.OIDCSubject != "new-oidc-subject" {
		t.Error("expected OIDC subject to be linked")
	}
}
