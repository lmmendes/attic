package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
	"github.com/mendelui/attic/internal/testutil"
)

func Test_LocationRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)
	loc := &domain.Location{
		OrganizationID: org.ID,
		Name:           "Office",
	}

	err := repo.Create(ctx, loc)
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	if loc.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if loc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_LocationRepository_Create_WithParent(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)

	// Create parent
	parent := &domain.Location{OrganizationID: org.ID, Name: "Building A"}
	repo.Create(ctx, parent)

	// Create child
	child := &domain.Location{
		OrganizationID: org.ID,
		ParentID:       &parent.ID,
		Name:           "Room 101",
	}
	err := repo.Create(ctx, child)
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Error("expected parent ID to be set")
	}
}

func Test_LocationRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)
	loc := &domain.Location{OrganizationID: org.ID, Name: "Warehouse"}
	repo.Create(ctx, loc)

	fetched, err := repo.GetByID(ctx, loc.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected location to be found")
	}
	if fetched.Name != "Warehouse" {
		t.Errorf("expected name 'Warehouse', got '%s'", fetched.Name)
	}
}

func Test_LocationRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewLocationRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent location")
	}
}

func Test_LocationRepository_List_ReturnsLocationsForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org1, _ := fixtures.CreateOrganization(ctx, "Org 1")
	org2, _ := fixtures.CreateOrganization(ctx, "Org 2")

	repo := NewLocationRepository(testDB.Pool)
	repo.Create(ctx, &domain.Location{OrganizationID: org1.ID, Name: "Office"})
	repo.Create(ctx, &domain.Location{OrganizationID: org1.ID, Name: "Warehouse"})
	repo.Create(ctx, &domain.Location{OrganizationID: org2.ID, Name: "Other"})

	locations, err := repo.List(ctx, org1.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(locations) != 2 {
		t.Errorf("expected 2 locations for org1, got %d", len(locations))
	}
}

func Test_LocationRepository_List_OrderedByName(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)
	repo.Create(ctx, &domain.Location{OrganizationID: org.ID, Name: "Zebra Room"})
	repo.Create(ctx, &domain.Location{OrganizationID: org.ID, Name: "Alpha Room"})
	repo.Create(ctx, &domain.Location{OrganizationID: org.ID, Name: "Beta Room"})

	locations, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if locations[0].Name != "Alpha Room" {
		t.Error("expected locations to be ordered by name")
	}
}

func Test_LocationRepository_ListTree_ReturnsHierarchy(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)

	// Create hierarchy: Building A -> Room 101, Room 102
	buildingA := &domain.Location{OrganizationID: org.ID, Name: "Building A"}
	repo.Create(ctx, buildingA)

	room101 := &domain.Location{OrganizationID: org.ID, ParentID: &buildingA.ID, Name: "Room 101"}
	repo.Create(ctx, room101)

	room102 := &domain.Location{OrganizationID: org.ID, ParentID: &buildingA.ID, Name: "Room 102"}
	repo.Create(ctx, room102)

	// Create another root
	buildingB := &domain.Location{OrganizationID: org.ID, Name: "Building B"}
	repo.Create(ctx, buildingB)

	tree, err := repo.ListTree(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list tree: %v", err)
	}

	// Should have 2 root locations (only roots are returned, children excluded)
	if len(tree) != 2 {
		t.Errorf("expected 2 root locations, got %d", len(tree))
	}

	// Verify both roots exist
	names := make(map[string]bool)
	for _, loc := range tree {
		names[loc.Name] = true
	}
	if !names["Building A"] || !names["Building B"] {
		t.Error("expected both Building A and Building B in roots")
	}
}

func Test_LocationRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)
	loc := &domain.Location{OrganizationID: org.ID, Name: "Old Name"}
	repo.Create(ctx, loc)

	// Update
	loc.Name = "New Name"
	desc := "Updated description"
	loc.Description = &desc

	err := repo.Update(ctx, loc)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, loc.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", fetched.Name)
	}
	if fetched.Description == nil || *fetched.Description != "Updated description" {
		t.Error("expected description to be updated")
	}
}

func Test_LocationRepository_Update_ChangeParent(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)

	parentA := &domain.Location{OrganizationID: org.ID, Name: "Parent A"}
	repo.Create(ctx, parentA)

	parentB := &domain.Location{OrganizationID: org.ID, Name: "Parent B"}
	repo.Create(ctx, parentB)

	child := &domain.Location{OrganizationID: org.ID, ParentID: &parentA.ID, Name: "Child"}
	repo.Create(ctx, child)

	// Move child to parentB
	child.ParentID = &parentB.ID
	err := repo.Update(ctx, child)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify
	fetched, _ := repo.GetByID(ctx, child.ID)
	if fetched.ParentID == nil || *fetched.ParentID != parentB.ID {
		t.Error("expected parent to be changed to parentB")
	}
}

func Test_LocationRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewLocationRepository(testDB.Pool)
	loc := &domain.Location{OrganizationID: org.ID, Name: "To Delete"}
	repo.Create(ctx, loc)

	// Delete
	err := repo.Delete(ctx, loc.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Should not be found
	fetched, _ := repo.GetByID(ctx, loc.ID)
	if fetched != nil {
		t.Error("expected deleted location not to be found")
	}
}
