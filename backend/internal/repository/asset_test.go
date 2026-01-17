package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
	"github.com/mendelui/attic/internal/testutil"
)

func Test_AssetRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)

	repo := NewAssetRepository(testDB.Pool)
	asset := &domain.Asset{
		OrganizationID: org.ID,
		CategoryID:     cat.ID,
		Name:           "iPhone 15",
		Quantity:       1,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	if asset.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if asset.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_AssetRepository_Create_WithAllFields(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	loc, _ := fixtures.CreateLocation(ctx, org.ID, "Office", nil)
	cond, _ := fixtures.CreateCondition(ctx, org.ID, "NEW", "New", 1)

	repo := NewAssetRepository(testDB.Pool)
	desc := "A new iPhone"
	price := 999.99
	purchaseAt := time.Now().UTC().Truncate(time.Second)
	purchaseNote := "Bought from Apple Store"
	attrs := json.RawMessage(`{"color": "black"}`)

	asset := &domain.Asset{
		OrganizationID: org.ID,
		CategoryID:     cat.ID,
		LocationID:     &loc.ID,
		ConditionID:    &cond.ID,
		Name:           "iPhone 15",
		Description:    &desc,
		Quantity:       2,
		Attributes:     attrs,
		PurchaseAt:     &purchaseAt,
		PurchasePrice:  &price,
		PurchaseNote:   &purchaseNote,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, asset.ID)
	if fetched.LocationID == nil || *fetched.LocationID != loc.ID {
		t.Error("expected location ID to be set")
	}
	if fetched.ConditionID == nil || *fetched.ConditionID != cond.ID {
		t.Error("expected condition ID to be set")
	}
	if fetched.PurchasePrice == nil || *fetched.PurchasePrice != 999.99 {
		t.Error("expected purchase price to be set")
	}
}

func Test_AssetRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAssetRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected asset to be found")
	}
	if fetched.Name != "Test Asset" {
		t.Errorf("expected name 'Test Asset', got '%s'", fetched.Name)
	}
}

func Test_AssetRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewAssetRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent asset")
	}
}

func Test_AssetRepository_GetByIDFull_WithRelations(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	loc, _ := fixtures.CreateLocation(ctx, org.ID, "Office", nil)
	cond, _ := fixtures.CreateCondition(ctx, org.ID, "NEW", "New", 1)

	repo := NewAssetRepository(testDB.Pool)
	asset := &domain.Asset{
		OrganizationID: org.ID,
		CategoryID:     cat.ID,
		LocationID:     &loc.ID,
		ConditionID:    &cond.ID,
		Name:           "iPhone",
		Quantity:       1,
	}
	repo.Create(ctx, asset)

	// Add warranty
	warRepo := NewWarrantyRepository(testDB.Pool)
	endDate := time.Now().AddDate(1, 0, 0)
	warranty := &domain.Warranty{
		AssetID: asset.ID,
		EndDate: &endDate,
	}
	warRepo.Create(ctx, warranty)

	// Add tag
	tagID, _ := fixtures.CreateTag(ctx, org.ID, "premium")
	fixtures.AddTagToAsset(ctx, asset.ID, tagID)

	fetched, err := repo.GetByIDFull(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to get full: %v", err)
	}

	if fetched.Category == nil {
		t.Error("expected category to be populated")
	}
	if fetched.Location == nil {
		t.Error("expected location to be populated")
	}
	if fetched.Condition == nil {
		t.Error("expected condition to be populated")
	}
	if fetched.Warranty == nil {
		t.Error("expected warranty to be populated")
	}
	if len(fetched.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(fetched.Tags))
	}
}

func Test_AssetRepository_List_WithPagination(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)

	for i := 0; i < 15; i++ {
		fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset")
	}

	repo := NewAssetRepository(testDB.Pool)
	assets, total, err := repo.List(ctx, org.ID, domain.AssetFilter{}, domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if total != 15 {
		t.Errorf("expected total 15, got %d", total)
	}
	if len(assets) != 10 {
		t.Errorf("expected 10 assets in first page, got %d", len(assets))
	}

	// Second page
	assets, _, err = repo.List(ctx, org.ID, domain.AssetFilter{}, domain.Pagination{Limit: 10, Offset: 10})
	if err != nil {
		t.Fatalf("failed to list second page: %v", err)
	}
	if len(assets) != 5 {
		t.Errorf("expected 5 assets in second page, got %d", len(assets))
	}
}

func Test_AssetRepository_List_FilterByCategory(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat1, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	cat2, _ := fixtures.CreateCategory(ctx, org.ID, "Books", nil)

	fixtures.CreateAsset(ctx, org.ID, cat1.ID, "Phone")
	fixtures.CreateAsset(ctx, org.ID, cat1.ID, "Laptop")
	fixtures.CreateAsset(ctx, org.ID, cat2.ID, "Book")

	repo := NewAssetRepository(testDB.Pool)
	assets, total, err := repo.List(ctx, org.ID, domain.AssetFilter{CategoryID: &cat1.ID}, domain.Pagination{Limit: 100})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 assets in Electronics, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}
}

func Test_AssetRepository_List_FilterByLocation(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	loc1, _ := fixtures.CreateLocation(ctx, org.ID, "Office", nil)
	loc2, _ := fixtures.CreateLocation(ctx, org.ID, "Warehouse", nil)

	repo := NewAssetRepository(testDB.Pool)
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, LocationID: &loc1.ID, Name: "Asset 1", Quantity: 1})
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, LocationID: &loc1.ID, Name: "Asset 2", Quantity: 1})
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, LocationID: &loc2.ID, Name: "Asset 3", Quantity: 1})

	assets, total, err := repo.List(ctx, org.ID, domain.AssetFilter{LocationID: &loc1.ID}, domain.Pagination{Limit: 100})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if total != 2 {
		t.Errorf("expected 2 assets in Office, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}
}

func Test_AssetRepository_List_FilterByCondition(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	condNew, _ := fixtures.CreateCondition(ctx, org.ID, "NEW", "New", 1)
	condUsed, _ := fixtures.CreateCondition(ctx, org.ID, "USED", "Used", 2)

	repo := NewAssetRepository(testDB.Pool)
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, ConditionID: &condNew.ID, Name: "New Asset", Quantity: 1})
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, ConditionID: &condUsed.ID, Name: "Used Asset", Quantity: 1})

	assets, total, err := repo.List(ctx, org.ID, domain.AssetFilter{ConditionID: &condNew.ID}, domain.Pagination{Limit: 100})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 new asset, got %d", total)
	}
	if len(assets) != 1 || assets[0].Name != "New Asset" {
		t.Error("expected to find the new asset")
	}
}

func Test_AssetRepository_Search_FullText(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)

	repo := NewAssetRepository(testDB.Pool)
	desc1 := "Apple smartphone with great camera"
	desc2 := "Samsung tablet for reading"
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, Name: "iPhone 15 Pro", Description: &desc1, Quantity: 1})
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, Name: "Galaxy Tab", Description: &desc2, Quantity: 1})

	assets, total, err := repo.Search(ctx, org.ID, "iPhone", domain.Pagination{Limit: 100})
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if total != 1 {
		t.Errorf("expected 1 result for 'iPhone', got %d", total)
	}
	if len(assets) != 1 {
		t.Error("expected to find iPhone")
	}
}

func Test_AssetRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	loc, _ := fixtures.CreateLocation(ctx, org.ID, "Office", nil)

	repo := NewAssetRepository(testDB.Pool)
	asset := &domain.Asset{
		OrganizationID: org.ID,
		CategoryID:     cat.ID,
		Name:           "Old Name",
		Quantity:       1,
	}
	repo.Create(ctx, asset)

	asset.Name = "New Name"
	asset.LocationID = &loc.ID
	asset.Quantity = 5

	err := repo.Update(ctx, asset)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, asset.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", fetched.Name)
	}
	if fetched.LocationID == nil || *fetched.LocationID != loc.ID {
		t.Error("expected location to be set")
	}
	if fetched.Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", fetched.Quantity)
	}
}

func Test_AssetRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "To Delete")

	repo := NewAssetRepository(testDB.Pool)
	err := repo.Delete(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, asset.ID)
	if fetched != nil {
		t.Error("expected deleted asset not to be found")
	}
}

func Test_AssetRepository_SetTags(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	tag1, _ := fixtures.CreateTag(ctx, org.ID, "premium")
	tag2, _ := fixtures.CreateTag(ctx, org.ID, "sale")

	repo := NewAssetRepository(testDB.Pool)

	// Set initial tags
	err := repo.SetTags(ctx, asset.ID, []uuid.UUID{tag1, tag2})
	if err != nil {
		t.Fatalf("failed to set tags: %v", err)
	}

	fetched, _ := repo.GetByIDFull(ctx, asset.ID)
	if len(fetched.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(fetched.Tags))
	}

	// Replace with single tag
	err = repo.SetTags(ctx, asset.ID, []uuid.UUID{tag1})
	if err != nil {
		t.Fatalf("failed to replace tags: %v", err)
	}

	fetched, _ = repo.GetByIDFull(ctx, asset.ID)
	if len(fetched.Tags) != 1 {
		t.Errorf("expected 1 tag after replace, got %d", len(fetched.Tags))
	}
}

func Test_AssetRepository_GetTotalValue(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)

	repo := NewAssetRepository(testDB.Pool)
	price1, price2 := 100.0, 200.0
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, Name: "Asset 1", Quantity: 2, PurchasePrice: &price1})
	repo.Create(ctx, &domain.Asset{OrganizationID: org.ID, CategoryID: cat.ID, Name: "Asset 2", Quantity: 1, PurchasePrice: &price2})

	total, err := repo.GetTotalValue(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to get total value: %v", err)
	}

	// 2 * 100 + 1 * 200 = 400
	if total != 400.0 {
		t.Errorf("expected total value 400, got %f", total)
	}
}
