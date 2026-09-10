package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_AssetEventRepository_CRUDAndOrdering(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	category, _ := fixtures.CreateCategory(ctx, org.ID, "Tools", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, category.ID, "Drill")
	repo := NewAssetEventRepository(testDB.Pool)

	older := &domain.AssetEvent{
		AssetID: asset.ID, Title: "Bought", Category: domain.AssetEventCategoryNote,
		Description: "Recorded the original purchase", Icon: "i-lucide-shopping-bag", OccurredAt: timestamp(2025, 1, 1, 9, 0),
	}
	newer := &domain.AssetEvent{
		AssetID: asset.ID, Title: "Serviced", Category: domain.AssetEventCategoryMaintenance,
		Description: "Completed routine service", Icon: "i-lucide-wrench", OccurredAt: timestamp(2026, 9, 8, 14, 30),
	}
	if err := repo.Create(ctx, org.ID, newer); err != nil {
		t.Fatalf("create newer event: %v", err)
	}
	if err := repo.Create(ctx, org.ID, older); err != nil {
		t.Fatalf("create older event: %v", err)
	}

	events, err := repo.ListByAsset(ctx, org.ID, asset.ID)
	if err != nil || len(events) != 2 {
		t.Fatalf("list events: len=%d err=%v", len(events), err)
	}
	if events[0].ID != newer.ID || events[1].ID != older.ID {
		t.Fatal("expected reverse chronological ordering")
	}

	newer.Title, newer.Category, newer.Description = "Repaired", domain.AssetEventCategoryRepair, "Replaced worn brushes"
	if err := repo.Update(ctx, org.ID, newer); err != nil {
		t.Fatalf("update event: %v", err)
	}
	fetched, err := repo.GetByID(ctx, org.ID, asset.ID, newer.ID)
	if err != nil || fetched == nil || fetched.Title != "Repaired" || fetched.Category != domain.AssetEventCategoryRepair || fetched.Description != "Replaced worn brushes" {
		t.Fatalf("unexpected updated event: %#v err=%v", fetched, err)
	}

	if err := repo.Delete(ctx, org.ID, asset.ID, newer.ID); err != nil {
		t.Fatalf("delete event: %v", err)
	}
	if fetched, _ := repo.GetByID(ctx, org.ID, asset.ID, newer.ID); fetched != nil {
		t.Fatal("expected event to be deleted")
	}
}

func Test_AssetEventRepository_ScopesEventsAndCascades(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	otherOrg, _ := fixtures.CreateOrganization(ctx, "Other Org")
	category, _ := fixtures.CreateCategory(ctx, org.ID, "Tools", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, category.ID, "Drill")
	repo := NewAssetEventRepository(testDB.Pool)
	event := &domain.AssetEvent{
		AssetID: asset.ID, Title: "Serviced", Category: domain.AssetEventCategoryMaintenance,
		Description: "Completed routine service", Icon: "i-lucide-wrench", OccurredAt: timestamp(2026, 9, 8, 14, 30),
	}
	if err := repo.Create(ctx, org.ID, event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	if events, err := repo.ListByAsset(ctx, otherOrg.ID, asset.ID); err != nil || len(events) != 0 {
		t.Fatalf("cross-workspace list returned events: %#v err=%v", events, err)
	}
	if fetched, err := repo.GetByID(ctx, otherOrg.ID, asset.ID, event.ID); err != nil || fetched != nil {
		t.Fatalf("cross-workspace get returned event: %#v err=%v", fetched, err)
	}
	if err := repo.Delete(ctx, otherOrg.ID, asset.ID, event.ID); err != pgx.ErrNoRows {
		t.Fatalf("expected scoped delete to return no rows, got %v", err)
	}

	if _, err := testDB.Pool.Exec(ctx, "DELETE FROM assets WHERE id = $1", asset.ID); err != nil {
		t.Fatalf("hard delete asset: %v", err)
	}
	var count int
	if err := testDB.Pool.QueryRow(ctx, "SELECT count(*) FROM asset_events WHERE id = $1", event.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("expected cascade deletion, count=%d err=%v", count, err)
	}
}

func Test_AssetEventRepository_CreateRejectsUnavailableAsset(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	repo := NewAssetEventRepository(testDB.Pool)
	event := &domain.AssetEvent{
		AssetID: uuid.New(), Title: "Missing", Category: domain.AssetEventCategoryNote,
		Description: "Missing asset", Icon: "i-lucide-calendar", OccurredAt: timestamp(2026, 9, 8, 14, 30),
	}
	if err := repo.Create(ctx, uuid.New(), event); err != pgx.ErrNoRows {
		t.Fatalf("expected no rows, got %v", err)
	}
}

func Test_AssetEventRepository_EnforcesRequiredCategoryAndDescription(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	category, _ := fixtures.CreateCategory(ctx, org.ID, "Tools", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, category.ID, "Drill")
	repo := NewAssetEventRepository(testDB.Pool)

	tests := []struct {
		name        string
		category    domain.AssetEventCategory
		description string
	}{
		{name: "invalid category", category: "other", description: "Unknown event"},
		{name: "blank description", category: domain.AssetEventCategoryNote, description: "  "},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := &domain.AssetEvent{
				AssetID: asset.ID, Title: "Event", Category: test.category,
				Description: test.description, Icon: "i-lucide-calendar", OccurredAt: timestamp(2026, 9, 8, 14, 30),
			}
			if err := repo.Create(ctx, org.ID, event); err == nil {
				t.Fatal("expected database constraint error")
			}
		})
	}
}

func timestamp(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}
