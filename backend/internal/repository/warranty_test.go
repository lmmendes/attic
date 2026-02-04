package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_WarrantyRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewWarrantyRepository(testDB.Pool)
	provider := "Apple"
	startDate := time.Now().UTC().Truncate(24 * time.Hour)
	endDate := startDate.AddDate(1, 0, 0)
	notes := "One year warranty"

	warranty := &domain.Warranty{
		AssetID:   asset.ID,
		Provider:  &provider,
		StartDate: &startDate,
		EndDate:   &endDate,
		Notes:     &notes,
	}

	err := repo.Create(ctx, warranty)
	if err != nil {
		t.Fatalf("failed to create warranty: %v", err)
	}

	if warranty.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if warranty.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_WarrantyRepository_GetByAssetID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewWarrantyRepository(testDB.Pool)
	provider := "Samsung"
	warranty := &domain.Warranty{AssetID: asset.ID, Provider: &provider}
	repo.Create(ctx, warranty)

	fetched, err := repo.GetByAssetID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected warranty to be found")
	}
	if fetched.Provider == nil || *fetched.Provider != "Samsung" {
		t.Error("expected provider to be Samsung")
	}
}

func Test_WarrantyRepository_GetByAssetID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewWarrantyRepository(testDB.Pool)
	fetched, err := repo.GetByAssetID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent warranty")
	}
}

func Test_WarrantyRepository_List_ReturnsWarrantiesForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset1, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 1")
	asset2, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 2")

	repo := NewWarrantyRepository(testDB.Pool)
	endDate1 := time.Now().AddDate(0, 6, 0)
	endDate2 := time.Now().AddDate(1, 0, 0)
	repo.Create(ctx, &domain.Warranty{AssetID: asset1.ID, EndDate: &endDate1})
	repo.Create(ctx, &domain.Warranty{AssetID: asset2.ID, EndDate: &endDate2})

	warranties, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(warranties) != 2 {
		t.Errorf("expected 2 warranties, got %d", len(warranties))
	}

	// Should be ordered by end_date ASC
	if warranties[0].AssetID != asset1.ID {
		t.Error("expected earlier warranty first")
	}
}

func Test_WarrantyRepository_List_IncludesAssetName(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "iPhone 15")

	repo := NewWarrantyRepository(testDB.Pool)
	repo.Create(ctx, &domain.Warranty{AssetID: asset.ID})

	warranties, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(warranties) != 1 {
		t.Fatalf("expected 1 warranty, got %d", len(warranties))
	}
	if warranties[0].AssetName != "iPhone 15" {
		t.Errorf("expected asset name 'iPhone 15', got '%s'", warranties[0].AssetName)
	}
}

func Test_WarrantyRepository_ListExpiring_ReturnsExpiringWarranties(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset1, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 1")
	asset2, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 2")
	asset3, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 3")

	repo := NewWarrantyRepository(testDB.Pool)

	// Expiring in 10 days
	endDate1 := time.Now().AddDate(0, 0, 10)
	repo.Create(ctx, &domain.Warranty{AssetID: asset1.ID, EndDate: &endDate1})

	// Expiring in 60 days
	endDate2 := time.Now().AddDate(0, 0, 60)
	repo.Create(ctx, &domain.Warranty{AssetID: asset2.ID, EndDate: &endDate2})

	// Expiring in 1 year
	endDate3 := time.Now().AddDate(1, 0, 0)
	repo.Create(ctx, &domain.Warranty{AssetID: asset3.ID, EndDate: &endDate3})

	// Get warranties expiring in next 30 days
	warranties, err := repo.ListExpiring(ctx, org.ID, 30)
	if err != nil {
		t.Fatalf("failed to list expiring: %v", err)
	}

	if len(warranties) != 1 {
		t.Errorf("expected 1 warranty expiring in 30 days, got %d", len(warranties))
	}
	if len(warranties) > 0 && warranties[0].AssetID != asset1.ID {
		t.Error("expected asset1's warranty")
	}
}

func Test_WarrantyRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewWarrantyRepository(testDB.Pool)
	oldProvider := "Old Provider"
	warranty := &domain.Warranty{AssetID: asset.ID, Provider: &oldProvider}
	repo.Create(ctx, warranty)

	// Update
	newProvider := "New Provider"
	newEndDate := time.Now().AddDate(2, 0, 0)
	warranty.Provider = &newProvider
	warranty.EndDate = &newEndDate

	err := repo.Update(ctx, warranty)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	fetched, _ := repo.GetByAssetID(ctx, asset.ID)
	if fetched.Provider == nil || *fetched.Provider != "New Provider" {
		t.Error("expected provider to be updated")
	}
}

func Test_WarrantyRepository_Delete_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewWarrantyRepository(testDB.Pool)
	warranty := &domain.Warranty{AssetID: asset.ID}
	repo.Create(ctx, warranty)

	err := repo.Delete(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	fetched, _ := repo.GetByAssetID(ctx, asset.ID)
	if fetched != nil {
		t.Error("expected warranty to be deleted")
	}
}
