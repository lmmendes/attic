package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_CategoryRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	cat := &domain.Category{
		OrganizationID: org.ID,
		Name:           "Electronics",
	}

	err := repo.Create(ctx, cat)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	if cat.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if cat.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_CategoryRepository_Create_WithParent(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)

	parent := &domain.Category{OrganizationID: org.ID, Name: "Electronics"}
	repo.Create(ctx, parent)

	child := &domain.Category{
		OrganizationID: org.ID,
		ParentID:       &parent.ID,
		Name:           "Phones",
	}
	err := repo.Create(ctx, child)
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Error("expected parent ID to be set")
	}
}

func Test_CategoryRepository_Create_WithPluginID(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	pluginID := "google_books"
	cat := &domain.Category{
		OrganizationID: org.ID,
		Name:           "Books",
		PluginID:       &pluginID,
	}

	err := repo.Create(ctx, cat)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, cat.ID)
	if fetched.PluginID == nil || *fetched.PluginID != pluginID {
		t.Error("expected plugin ID to be set")
	}
}

func Test_CategoryRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	cat := &domain.Category{OrganizationID: org.ID, Name: "Electronics"}
	repo.Create(ctx, cat)

	fetched, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected category to be found")
	}
	if fetched.Name != "Electronics" {
		t.Errorf("expected name 'Electronics', got '%s'", fetched.Name)
	}
}

func Test_CategoryRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewCategoryRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent category")
	}
}

func Test_CategoryRepository_GetByPluginID(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	pluginID := "google_books"
	cat := &domain.Category{
		OrganizationID: org.ID,
		Name:           "Books",
		PluginID:       &pluginID,
	}
	repo.Create(ctx, cat)

	fetched, err := repo.GetByPluginID(ctx, org.ID, pluginID)
	if err != nil {
		t.Fatalf("failed to get by plugin ID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected category to be found")
	}
	if fetched.ID != cat.ID {
		t.Error("expected same category")
	}
}

func Test_CategoryRepository_GetByIDWithAttributes(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	catRepo := NewCategoryRepository(testDB.Pool)
	attrRepo := NewAttributeRepository(testDB.Pool)

	cat := &domain.Category{OrganizationID: org.ID, Name: "Electronics"}
	catRepo.Create(ctx, cat)

	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Brand",
		Key:            "brand",
		DataType:       domain.AttributeTypeString,
	}
	attrRepo.Create(ctx, attr)

	// Assign attribute to category
	catRepo.SetAttributes(ctx, cat.ID, []domain.CategoryAttributeAssignment{
		{AttributeID: attr.ID, Required: true, SortOrder: 1},
	})

	fetched, err := catRepo.GetByIDWithAttributes(ctx, cat.ID)
	if err != nil {
		t.Fatalf("failed to get with attributes: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected category to be found")
	}
	if len(fetched.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(fetched.Attributes))
	}
	if fetched.Attributes[0].Attribute == nil {
		t.Fatal("expected attribute to be populated")
	}
	if fetched.Attributes[0].Attribute.Name != "Brand" {
		t.Errorf("expected attribute name 'Brand', got '%s'", fetched.Attributes[0].Attribute.Name)
	}
}

func Test_CategoryRepository_List_ReturnsCategoriesForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org1, _ := fixtures.CreateOrganization(ctx, "Org 1")
	org2, _ := fixtures.CreateOrganization(ctx, "Org 2")

	repo := NewCategoryRepository(testDB.Pool)
	repo.Create(ctx, &domain.Category{OrganizationID: org1.ID, Name: "Electronics"})
	repo.Create(ctx, &domain.Category{OrganizationID: org1.ID, Name: "Books"})
	repo.Create(ctx, &domain.Category{OrganizationID: org2.ID, Name: "Other"})

	categories, err := repo.List(ctx, org1.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(categories) != 2 {
		t.Errorf("expected 2 categories for org1, got %d", len(categories))
	}
}

func Test_CategoryRepository_ListTree_ReturnsHierarchy(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)

	electronics := &domain.Category{OrganizationID: org.ID, Name: "Electronics"}
	repo.Create(ctx, electronics)

	phones := &domain.Category{OrganizationID: org.ID, ParentID: &electronics.ID, Name: "Phones"}
	repo.Create(ctx, phones)

	laptops := &domain.Category{OrganizationID: org.ID, ParentID: &electronics.ID, Name: "Laptops"}
	repo.Create(ctx, laptops)

	books := &domain.Category{OrganizationID: org.ID, Name: "Books"}
	repo.Create(ctx, books)

	tree, err := repo.ListTree(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list tree: %v", err)
	}

	if len(tree) != 2 {
		t.Errorf("expected 2 root categories, got %d", len(tree))
	}
}

func Test_CategoryRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	cat := &domain.Category{OrganizationID: org.ID, Name: "Old Name"}
	repo.Create(ctx, cat)

	cat.Name = "New Name"
	desc := "Updated description"
	cat.Description = &desc
	icon := "📱"
	cat.Icon = &icon

	err := repo.Update(ctx, cat)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, cat.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", fetched.Name)
	}
	if fetched.Icon == nil || *fetched.Icon != "📱" {
		t.Error("expected icon to be updated")
	}
}

func Test_CategoryRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewCategoryRepository(testDB.Pool)
	cat := &domain.Category{OrganizationID: org.ID, Name: "To Delete"}
	repo.Create(ctx, cat)

	err := repo.Delete(ctx, cat.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, cat.ID)
	if fetched != nil {
		t.Error("expected deleted category not to be found")
	}
}

func Test_CategoryRepository_SetAttributes(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	catRepo := NewCategoryRepository(testDB.Pool)
	attrRepo := NewAttributeRepository(testDB.Pool)

	cat := &domain.Category{OrganizationID: org.ID, Name: "Electronics"}
	catRepo.Create(ctx, cat)

	attr1 := &domain.Attribute{OrganizationID: org.ID, Name: "Brand", Key: "brand", DataType: domain.AttributeTypeString}
	attr2 := &domain.Attribute{OrganizationID: org.ID, Name: "Model", Key: "model", DataType: domain.AttributeTypeString}
	attrRepo.Create(ctx, attr1)
	attrRepo.Create(ctx, attr2)

	// Set initial attributes
	err := catRepo.SetAttributes(ctx, cat.ID, []domain.CategoryAttributeAssignment{
		{AttributeID: attr1.ID, Required: true, SortOrder: 1},
		{AttributeID: attr2.ID, Required: false, SortOrder: 2},
	})
	if err != nil {
		t.Fatalf("failed to set attributes: %v", err)
	}

	fetched, _ := catRepo.GetByIDWithAttributes(ctx, cat.ID)
	if len(fetched.Attributes) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(fetched.Attributes))
	}

	// Replace with different attributes
	err = catRepo.SetAttributes(ctx, cat.ID, []domain.CategoryAttributeAssignment{
		{AttributeID: attr1.ID, Required: false, SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("failed to replace attributes: %v", err)
	}

	fetched, _ = catRepo.GetByIDWithAttributes(ctx, cat.ID)
	if len(fetched.Attributes) != 1 {
		t.Errorf("expected 1 attribute after replace, got %d", len(fetched.Attributes))
	}
}

func Test_CategoryRepository_GetAssetCounts(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat1, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	cat2, _ := fixtures.CreateCategory(ctx, org.ID, "Books", nil)

	// Create assets in categories
	fixtures.CreateAsset(ctx, org.ID, cat1.ID, "Phone 1")
	fixtures.CreateAsset(ctx, org.ID, cat1.ID, "Phone 2")
	fixtures.CreateAsset(ctx, org.ID, cat2.ID, "Book 1")

	repo := NewCategoryRepository(testDB.Pool)
	counts, err := repo.GetAssetCounts(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to get asset counts: %v", err)
	}

	if counts[cat1.ID.String()] != 2 {
		t.Errorf("expected 2 assets in cat1, got %d", counts[cat1.ID.String()])
	}
	if counts[cat2.ID.String()] != 1 {
		t.Errorf("expected 1 asset in cat2, got %d", counts[cat2.ID.String()])
	}
}
