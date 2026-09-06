package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestCollectionsMembershipLifecycle(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(testDB.TruncateAll(ctx))
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Home")
	must(err)
	other, err := fixtures.CreateOrganization(ctx, "Other home")
	must(err)
	category, err := fixtures.CreateCategory(ctx, org.ID, "Games", nil)
	must(err)
	collections := NewCollectionRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	first := &domain.Collection{OrganizationID: org.ID, Name: "PS5 games", Icon: "i-lucide-gamepad-2"}
	second := &domain.Collection{OrganizationID: org.ID, Name: "Favorites", Icon: "i-lucide-library"}
	foreign := &domain.Collection{OrganizationID: other.ID, Name: "Private", Icon: "i-lucide-library"}
	for _, c := range []*domain.Collection{first, second, foreign} {
		must(collections.Create(ctx, c))
	}
	duplicate := &domain.Collection{OrganizationID: org.ID, Name: "ps5 GAMES", Icon: "i-lucide-library"}
	if err := collections.Create(ctx, duplicate); err == nil {
		t.Fatal("expected case-insensitive duplicate rejection")
	}
	list, err := collections.List(ctx, org.ID)
	must(err)
	if len(list) != 2 {
		t.Fatalf("workspace list: %v", list)
	}
	hidden, err := collections.GetByID(ctx, org.ID, foreign.ID)
	must(err)
	if hidden != nil {
		t.Fatal("foreign collection visible")
	}
	if !errors.Is(collections.Delete(ctx, org.ID, foreign.ID), pgx.ErrNoRows) {
		t.Fatal("foreign collection deleted")
	}
	a := &domain.Asset{OrganizationID: org.ID, CategoryID: &category.ID, Name: "Game", Quantity: 1, CollectionIDs: []uuid.UUID{first.ID, second.ID, first.ID}}
	must(assets.Create(ctx, a))
	loaded, err := assets.GetByID(ctx, a.ID)
	must(err)
	if len(loaded.Collections) != 2 {
		t.Fatalf("memberships: %v", loaded.Collections)
	}
	page, total, err := assets.List(ctx, org.ID, domain.AssetFilter{CollectionID: &first.ID}, domain.Pagination{Limit: 1})
	must(err)
	if total != 1 || len(page) != 1 || len(page[0].Collections) != 2 {
		t.Fatalf("filter/count/hydration: %d %v", total, page)
	}
	c, err := collections.GetByID(ctx, org.ID, first.ID)
	must(err)
	if c.AssetCount != 1 {
		t.Fatalf("count: %d", c.AssetCount)
	}
	c.Name = "PlayStation 5"
	must(collections.Update(ctx, c))
	loaded.Name = "Should roll back"
	loaded.CollectionIDs = []uuid.UUID{foreign.ID}
	if !errors.Is(assets.Update(ctx, loaded), ErrInvalidCollections) {
		t.Fatal("expected foreign assignment rejection")
	}
	loaded, err = assets.GetByID(ctx, a.ID)
	must(err)
	if loaded.Name != "Game" || len(loaded.Collections) != 2 {
		t.Fatal("invalid update was not atomic")
	}
	invalid := &domain.Asset{OrganizationID: org.ID, CategoryID: &category.ID, Name: "Invalid", Quantity: 1, CollectionIDs: []uuid.UUID{uuid.New()}}
	if !errors.Is(assets.Create(ctx, invalid), ErrInvalidCollections) {
		t.Fatal("expected missing collection rejection")
	}
	missing, err := assets.GetByID(ctx, invalid.ID)
	must(err)
	if missing != nil {
		t.Fatal("invalid create persisted an asset")
	}
	loaded.CollectionIDs = nil
	loaded.Name = "Renamed"
	must(assets.Update(ctx, loaded))
	if len(loaded.Collections) != 2 {
		t.Fatal("omission cleared memberships")
	}
	must(collections.Delete(ctx, org.ID, first.ID))
	loaded, err = assets.GetByID(ctx, a.ID)
	must(err)
	if loaded == nil || len(loaded.Collections) != 1 || loaded.Collections[0].ID != second.ID {
		t.Fatal("delete must preserve asset and other memberships")
	}
	loaded.CollectionIDs = []uuid.UUID{}
	must(assets.Update(ctx, loaded))
	if len(loaded.Collections) != 0 || loaded.CollectionIDs == nil {
		t.Fatal("explicit empty array did not clear memberships")
	}
	loaded.CollectionIDs = []uuid.UUID{second.ID}
	must(assets.Update(ctx, loaded))
	must(assets.Delete(ctx, loaded.ID))
	c, err = collections.GetByID(ctx, org.ID, second.ID)
	must(err)
	if c.AssetCount != 0 {
		t.Fatal("deleted asset counted")
	}
}
