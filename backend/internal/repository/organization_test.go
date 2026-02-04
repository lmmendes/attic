package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

var testDB *testutil.TestDB

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	testDB, err = testutil.NewTestDB(ctx)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	defer testDB.Close(ctx)
	m.Run()
}

func Test_OrganizationRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	org := &domain.Organization{
		Name: "Test Organization",
	}

	err := repo.Create(ctx, org)
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	if org.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if org.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if org.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func Test_OrganizationRepository_Create_WithDescription(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	desc := "A test organization description"
	org := &domain.Organization{
		Name:        "Test Organization",
		Description: &desc,
	}

	err := repo.Create(ctx, org)
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	// Fetch and verify
	fetched, err := repo.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to get organization: %v", err)
	}
	if fetched.Description == nil || *fetched.Description != desc {
		t.Error("expected description to be set")
	}
}

func Test_OrganizationRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	org := &domain.Organization{Name: "Test Org"}
	if err := repo.Create(ctx, org); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, err := repo.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to get by ID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected organization to be found")
	}
	if fetched.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got '%s'", fetched.Name)
	}
}

func Test_OrganizationRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent organization")
	}
}

func Test_OrganizationRepository_GetDefault_ReturnsFirst(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)

	// Create first org
	org1 := &domain.Organization{Name: "First Org"}
	if err := repo.Create(ctx, org1); err != nil {
		t.Fatalf("failed to create first: %v", err)
	}

	// Create second org
	org2 := &domain.Organization{Name: "Second Org"}
	if err := repo.Create(ctx, org2); err != nil {
		t.Fatalf("failed to create second: %v", err)
	}

	defaultOrg, err := repo.GetDefault(ctx)
	if err != nil {
		t.Fatalf("failed to get default: %v", err)
	}
	if defaultOrg == nil {
		t.Fatal("expected default organization")
	}
	if defaultOrg.ID != org1.ID {
		t.Error("expected first created organization to be default")
	}
}

func Test_OrganizationRepository_GetDefault_NoOrganizations(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	defaultOrg, err := repo.GetDefault(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultOrg != nil {
		t.Error("expected nil when no organizations exist")
	}
}

func Test_OrganizationRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewOrganizationRepository(testDB.Pool)
	org := &domain.Organization{Name: "Original Name"}
	if err := repo.Create(ctx, org); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	originalUpdatedAt := org.UpdatedAt

	// Update
	org.Name = "Updated Name"
	desc := "New description"
	org.Description = &desc
	if err := repo.Update(ctx, org); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify
	fetched, err := repo.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	if fetched.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", fetched.Name)
	}
	if fetched.Description == nil || *fetched.Description != "New description" {
		t.Error("expected description to be updated")
	}
	if !fetched.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}
