package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestTagLifecycleAndAssetAssignments(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	f := testutil.NewFixtures(testDB.Pool)
	org, _ := f.CreateOrganization(ctx, "Tags")
	other, _ := f.CreateOrganization(ctx, "Other tags")
	category, _ := f.CreateCategory(ctx, org.ID, "Things", nil)
	tags := NewTagRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	description := "Old computers"
	retro := &domain.Tag{OrganizationID: org.ID, Name: "Retro", Description: &description}
	if err := tags.Create(ctx, retro); err != nil {
		t.Fatal(err)
	}
	if duplicate := tags.Create(ctx, &domain.Tag{OrganizationID: org.ID, Name: "retro"}); duplicate == nil {
		t.Fatal("case-insensitive duplicate accepted")
	}
	foreign := &domain.Tag{OrganizationID: other.ID, Name: "Foreign"}
	if err := tags.Create(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	asset := &domain.Asset{OrganizationID: org.ID, CategoryID: &category.ID, Name: "Computer", Quantity: 1, TagIDs: []uuid.UUID{retro.ID}, NewTagNames: []string{"RETRO", "Portable"}}
	if err := assets.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	if len(asset.Tags) != 2 {
		t.Fatalf("expected two unique tags: %+v", asset.Tags)
	}
	listed, err := tags.List(ctx, org.ID)
	if err != nil || len(listed) != 2 || listed[0].AssetCount+listed[1].AssetCount != 2 {
		t.Fatalf("tag list/counts: %+v %v", listed, err)
	}
	loaded, err := assets.GetByIDFull(ctx, asset.ID)
	if err != nil || loaded == nil || len(loaded.Tags) != 2 {
		t.Fatalf("asset tags not loaded: %+v %v", loaded, err)
	}
	loaded.Name = "Should roll back"
	loaded.TagIDs = []uuid.UUID{foreign.ID}
	if err := assets.Update(ctx, loaded); !errors.Is(err, ErrInvalidTags) {
		t.Fatalf("foreign tag accepted: %v", err)
	}
	reloaded, _ := assets.GetByID(ctx, asset.ID)
	if reloaded.Name != "Computer" || len(reloaded.Tags) != 2 {
		t.Fatal("invalid assignment partially updated the asset")
	}
	portable, _ := tags.GetByName(ctx, org.ID, "portable")
	portable.Name = "Travel"
	if err := tags.Update(ctx, portable); err != nil {
		t.Fatal(err)
	}
	loaded, _ = assets.GetByID(ctx, asset.ID)
	if !hasTag(loaded.Tags, "Travel") {
		t.Fatal("tag rename did not update presentation")
	}
	if err := tags.Delete(ctx, org.ID, portable.ID); err != nil {
		t.Fatal(err)
	}
	loaded, _ = assets.GetByID(ctx, asset.ID)
	if len(loaded.Tags) != 1 || loaded.Name != "Computer" {
		t.Fatal("tag deletion did not detach safely")
	}
}

func hasTag(tags []domain.Tag, name string) bool {
	for _, tag := range tags {
		if tag.Name == name {
			return true
		}
	}
	return false
}
