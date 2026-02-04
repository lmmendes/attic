package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_ConditionRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	cond := &domain.Condition{
		OrganizationID: org.ID,
		Code:           "NEW",
		Label:          "New",
		SortOrder:      1,
	}

	err := repo.Create(ctx, cond)
	if err != nil {
		t.Fatalf("failed to create condition: %v", err)
	}

	if cond.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if cond.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_ConditionRepository_Create_WithDescription(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	desc := "Brand new item"
	cond := &domain.Condition{
		OrganizationID: org.ID,
		Code:           "NEW",
		Label:          "New",
		Description:    &desc,
		SortOrder:      1,
	}

	err := repo.Create(ctx, cond)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, cond.ID)
	if fetched.Description == nil || *fetched.Description != desc {
		t.Error("expected description to be set")
	}
}

func Test_ConditionRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	cond := &domain.Condition{
		OrganizationID: org.ID,
		Code:           "GOOD",
		Label:          "Good Condition",
		SortOrder:      2,
	}
	repo.Create(ctx, cond)

	fetched, err := repo.GetByID(ctx, cond.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected condition to be found")
	}
	if fetched.Code != "GOOD" {
		t.Errorf("expected code 'GOOD', got '%s'", fetched.Code)
	}
}

func Test_ConditionRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewConditionRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent condition")
	}
}

func Test_ConditionRepository_List_ReturnsConditionsForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org1, _ := fixtures.CreateOrganization(ctx, "Org 1")
	org2, _ := fixtures.CreateOrganization(ctx, "Org 2")

	repo := NewConditionRepository(testDB.Pool)
	repo.Create(ctx, &domain.Condition{OrganizationID: org1.ID, Code: "NEW", Label: "New", SortOrder: 1})
	repo.Create(ctx, &domain.Condition{OrganizationID: org1.ID, Code: "GOOD", Label: "Good", SortOrder: 2})
	repo.Create(ctx, &domain.Condition{OrganizationID: org2.ID, Code: "FAIR", Label: "Fair", SortOrder: 1})

	conditions, err := repo.List(ctx, org1.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(conditions) != 2 {
		t.Errorf("expected 2 conditions for org1, got %d", len(conditions))
	}
}

func Test_ConditionRepository_List_OrderedBySortOrderThenLabel(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	repo.Create(ctx, &domain.Condition{OrganizationID: org.ID, Code: "FAIR", Label: "Fair", SortOrder: 3})
	repo.Create(ctx, &domain.Condition{OrganizationID: org.ID, Code: "NEW", Label: "New", SortOrder: 1})
	repo.Create(ctx, &domain.Condition{OrganizationID: org.ID, Code: "GOOD", Label: "Good", SortOrder: 2})

	conditions, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if conditions[0].Code != "NEW" || conditions[1].Code != "GOOD" || conditions[2].Code != "FAIR" {
		t.Error("expected conditions to be ordered by sort_order")
	}
}

func Test_ConditionRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	cond := &domain.Condition{
		OrganizationID: org.ID,
		Code:           "OLD",
		Label:          "Old Label",
		SortOrder:      1,
	}
	repo.Create(ctx, cond)

	// Update
	cond.Code = "UPDATED"
	cond.Label = "Updated Label"
	desc := "Now with description"
	cond.Description = &desc
	cond.SortOrder = 5

	err := repo.Update(ctx, cond)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, cond.ID)
	if fetched.Code != "UPDATED" {
		t.Errorf("expected code 'UPDATED', got '%s'", fetched.Code)
	}
	if fetched.Label != "Updated Label" {
		t.Errorf("expected label 'Updated Label', got '%s'", fetched.Label)
	}
	if fetched.SortOrder != 5 {
		t.Errorf("expected sort_order 5, got %d", fetched.SortOrder)
	}
}

func Test_ConditionRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	cond := &domain.Condition{
		OrganizationID: org.ID,
		Code:           "TO_DELETE",
		Label:          "To Delete",
		SortOrder:      1,
	}
	repo.Create(ctx, cond)

	// Delete
	err := repo.Delete(ctx, cond.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Should not be found
	fetched, _ := repo.GetByID(ctx, cond.ID)
	if fetched != nil {
		t.Error("expected deleted condition not to be found")
	}

	// But should not affect list for other orgs
	conditions, _ := repo.List(ctx, org.ID)
	if len(conditions) != 0 {
		t.Error("expected no conditions after delete")
	}
}

func Test_ConditionRepository_UniqueCodePerOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewConditionRepository(testDB.Pool)
	repo.Create(ctx, &domain.Condition{OrganizationID: org.ID, Code: "NEW", Label: "New", SortOrder: 1})

	// Try to create duplicate code
	err := repo.Create(ctx, &domain.Condition{OrganizationID: org.ID, Code: "NEW", Label: "Another New", SortOrder: 2})
	if err == nil {
		t.Error("expected error for duplicate code within same org")
	}
}
