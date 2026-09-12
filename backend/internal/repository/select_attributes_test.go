package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestSelectAttributeLifecycle(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Select tests")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAttributeRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	user := uuid.New()
	field := &domain.Attribute{OrganizationID: org.ID, Name: "Platforms", Key: "platforms", DataType: domain.AttributeTypeSelect, SelectionMode: "multiple", Options: []domain.AttributeOption{{Label: "Linux", Value: "linux"}, {Label: "macOS", Value: "macos"}}}
	if err = repo.Create(ctx, field); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetByID(ctx, field.ID)
	if err != nil || len(loaded.Options) != 2 || loaded.SelectionMode != "multiple" {
		t.Fatalf("definition: %+v %v", loaded, err)
	}
	a := &domain.Asset{OrganizationID: org.ID, Name: "Workstation", Quantity: 37, Attributes: json.RawMessage(`{"platforms":["linux","macos"],"notes":"keep","large":9007199254740993}`)}
	if err = assets.Create(ctx, a); err != nil {
		t.Fatal(err)
	}
	optionID := field.Options[0].ID
	name, value := "GNU/Linux", "gnu_linux"
	rename := AttributeAction{Action: "update_option", OptionID: &optionID, Changes: AttributeChange{Label: &name, Value: &value}}
	preview, err := repo.ChangeWithImpact(ctx, org.ID, user, field.ID, rename, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if preview.AffectedAssets != 1 {
		t.Fatalf("count assets, not quantity: %+v", preview)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.New(), field.ID, rename, preview.ConfirmationToken, false); err == nil {
		t.Fatal("another user reused token")
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, rename, preview.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	current, err := assets.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var vals map[string]any
	json.Unmarshal(current.Attributes, &vals)
	if !selectContains(vals["platforms"], "gnu_linux") || vals["notes"] != "keep" {
		t.Fatalf("rename: %s", current.Attributes)
	}
	// A label-only change preserves stored JSON.
	newLabel := "GNU Linux"
	labelAction := AttributeAction{Action: "update_option", OptionID: &optionID, Changes: AttributeChange{Label: &newLabel}}
	p, err := repo.ChangeWithImpact(ctx, org.ID, user, field.ID, labelAction, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, labelAction, p.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	after, _ := assets.GetByID(ctx, a.ID)
	if string(current.Attributes) != string(after.Attributes) {
		t.Fatal("label rename rewrote values")
	}
	// Preview invalidates when the affected set changes, even with the same count.
	del := AttributeAction{Action: "delete_option", OptionID: &optionID}
	p, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, del, "", true)
	if err != nil {
		t.Fatal(err)
	}
	after.Attributes = json.RawMessage(`{"platforms":["macos"],"notes":"keep","large":9007199254740993}`)
	if err = assets.Update(ctx, after); err != nil {
		t.Fatal(err)
	}
	b := &domain.Asset{OrganizationID: org.ID, Name: "Second", Quantity: 1, Attributes: json.RawMessage(`{"platforms":["gnu_linux"]}`)}
	if err = assets.Create(ctx, b); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, del, p.ConfirmationToken, false); err == nil {
		t.Fatal("same-count different-set preview accepted")
	}
	if err = assets.Delete(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	p, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, del, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if p.AffectedAssets != 1 || p.DeletedAssets != 1 {
		t.Fatalf("deleted asset count: %+v", p)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, del, p.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	var raw json.RawMessage
	if err = testDB.Pool.QueryRow(ctx, "SELECT attributes FROM assets WHERE id=$1", b.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}" {
		t.Fatalf("empty array not removed: %s", raw)
	}
	key := "operating_systems"
	change := AttributeAction{Action: "update_attribute", Changes: AttributeChange{Key: &key}}
	p, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, change, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, change, p.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	current, _ = assets.GetByID(ctx, a.ID)
	vals = nil
	json.Unmarshal(current.Attributes, &vals)
	if vals["platforms"] != nil || !selectContains(vals[key], "macos") {
		t.Fatalf("key migration: %s", current.Attributes)
	}
	stale := &domain.Asset{OrganizationID: org.ID, Name: "Stale client", Quantity: 1, Attributes: json.RawMessage(`{"platforms":["macos"]}`)}
	if err = assets.Create(ctx, stale); err == nil {
		t.Fatal("retired key accepted")
	}
	remove := AttributeAction{Action: "delete_attribute"}
	p, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, remove, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, user, field.ID, remove, p.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	current, _ = assets.GetByID(ctx, a.ID)
	vals = nil
	json.Unmarshal(current.Attributes, &vals)
	if len(vals) != 2 || vals["notes"] != "keep" || !strings.Contains(string(current.Attributes), "9007199254740993") {
		t.Fatalf("delete: %s", current.Attributes)
	}
}

func TestSelectAttributeOptionsAreSavedAsOneBatch(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Batch option changes")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAttributeRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	field := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Platform",
		Key:            "platform",
		DataType:       domain.AttributeTypeSelect,
		SelectionMode:  "multiple",
		Options: []domain.AttributeOption{
			{Label: "Linux", Value: "linux"},
			{Label: "macOS", Value: "macos"},
			{Label: "Windows", Value: "windows"},
		},
	}
	if err = repo.Create(ctx, field); err != nil {
		t.Fatal(err)
	}
	asset := &domain.Asset{OrganizationID: org.ID, Name: "Computer", Quantity: 1, Attributes: json.RawMessage(`{"platform":["linux","macos","windows"]}`)}
	if err = assets.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	newOptionID := uuid.New()
	options := []domain.AttributeOption{
		{ID: field.Options[1].ID, Label: "Apple macOS", Value: "apple_macos"},
		{ID: field.Options[0].ID, Label: "GNU/Linux", Value: "linux"},
		{ID: newOptionID, Label: "FreeBSD", Value: "freebsd"},
	}
	newKey := "operating_system"
	action := AttributeAction{Action: "update_attribute", Changes: AttributeChange{Key: &newKey, Options: &options}}
	preview, err := repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if preview.AffectedAssets != 1 {
		t.Fatalf("batch impact: %+v", preview)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, preview.ConfirmationToken, false); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetByID(ctx, field.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Options) != 3 || loaded.Options[0].ID != field.Options[1].ID || loaded.Options[0].Label != "Apple macOS" || loaded.Options[2].ID != newOptionID {
		t.Fatalf("options not replaced atomically: %+v", loaded.Options)
	}
	updated, err := assets.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err = json.Unmarshal(updated.Attributes, &values); err != nil {
		t.Fatal(err)
	}
	if values["platform"] != nil || !selectContains(values[newKey], "linux") || !selectContains(values[newKey], "apple_macos") || selectContains(values[newKey], "windows") {
		t.Fatalf("batch value migration: %s", updated.Attributes)
	}
}

func TestSelectValidationAndImpactConflicts(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	f := testutil.NewFixtures(testDB.Pool)
	org, err := f.CreateOrganization(ctx, "Validation")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAttributeRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	for _, mode := range []string{"single", "multiple"} {
		field := &domain.Attribute{OrganizationID: org.ID, Name: mode, Key: mode, DataType: domain.AttributeTypeSelect, SelectionMode: mode, Options: []domain.AttributeOption{{Label: "One", Value: "one"}, {Label: "Two", Value: "two"}}}
		if err = repo.Create(ctx, field); err != nil {
			t.Fatal(err)
		}
		invalid := []string{`null`, `true`, `42`, `["one","one"]`, `["unknown"]`, `"unknown"`}
		if mode == "single" {
			invalid = append(invalid, `["one"]`)
		} else {
			invalid = append(invalid, `"one"`)
		}
		for _, v := range invalid {
			a := &domain.Asset{OrganizationID: org.ID, Name: "Bad", Quantity: 1, Attributes: json.RawMessage(`{"` + mode + `":` + v + `}`)}
			if err = assets.Create(ctx, a); err == nil {
				t.Fatalf("%s accepted %s", mode, v)
			}
		}
		var good string
		if mode == "single" {
			good = `"one"`
		} else {
			good = `["one","two"]`
		}
		a := &domain.Asset{OrganizationID: org.ID, Name: "Good", Quantity: 1, Attributes: json.RawMessage(`{"` + mode + `":` + good + `}`)}
		if err = assets.Create(ctx, a); err != nil {
			t.Fatal(err)
		}
		changeMode := "multiple"
		if mode == "multiple" {
			changeMode = "single"
		}
		action := AttributeAction{Action: "update_attribute", Changes: AttributeChange{SelectionMode: &changeMode}}
		if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.New(), field.ID, action, "", true); err == nil {
			t.Fatal("populated mode converted")
		}
		duplicate := "two"
		oid := field.Options[0].ID
		action = AttributeAction{Action: "update_option", OptionID: &oid, Changes: AttributeChange{Value: &duplicate}}
		if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.New(), field.ID, action, "", true); err == nil {
			t.Fatal("duplicate option accepted")
		}
		action = AttributeAction{Action: "delete_option", OptionID: &oid}
		if _, err = repo.ChangeWithImpact(ctx, uuid.New(), uuid.New(), field.ID, action, "", true); err == nil {
			t.Fatal("foreign org accepted")
		}
		p, err := repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, "", true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, "1.expired", false); err == nil {
			t.Fatal("expired token accepted")
		}
		if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, p.ConfirmationToken, false); err != nil {
			t.Fatal(err)
		}
		current, _ := assets.GetByID(ctx, a.ID)
		var vals map[string]any
		json.Unmarshal(current.Attributes, &vals)
		if mode == "single" && vals[mode] != nil {
			t.Fatalf("single not cleared: %s", current.Attributes)
		}
		if mode == "multiple" && !selectContains(vals[mode], "two") {
			t.Fatalf("other option lost: %s", current.Attributes)
		}
	}
	// A held organization lock serializes asset writes with definition changes.
	tx, err := testDB.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeWrites(ctx, tx, org.ID); err != nil {
		t.Fatal(err)
	}
	timeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err = assets.Create(timeout, &domain.Asset{OrganizationID: org.ID, Name: "Blocked", Quantity: 1})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("write did not wait for lock: %v", err)
	}
}

func TestSelectMutationRollbackAndOrdering(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	f := testutil.NewFixtures(testDB.Pool)
	org, err := f.CreateOrganization(ctx, "Rollback")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAttributeRepository(testDB.Pool)
	assets := NewAssetRepository(testDB.Pool)
	field := &domain.Attribute{OrganizationID: org.ID, Name: "Choice", Key: "choice", DataType: domain.AttributeTypeSelect, Options: []domain.AttributeOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}}
	if err = repo.Create(ctx, field); err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{field.Options[1].ID, field.Options[0].ID}
	if err = repo.OrderOptions(ctx, org.ID, field.ID, ids); err != nil {
		t.Fatal(err)
	}
	if err = repo.OrderOptions(ctx, org.ID, field.ID, []uuid.UUID{ids[0], ids[0]}); err == nil {
		t.Fatal("duplicate order accepted")
	}
	loaded, err := repo.GetByID(ctx, field.ID)
	if err != nil || loaded.Options[0].ID != ids[0] {
		t.Fatalf("order: %+v %v", loaded, err)
	}
	asset := &domain.Asset{OrganizationID: org.ID, Name: "Rollback", Quantity: 1, Attributes: json.RawMessage(`{"choice":"a"}`)}
	if err = assets.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	if _, err = testDB.Pool.Exec(ctx, "ALTER TABLE assets ADD CONSTRAINT test_keep_choice CHECK (attributes ? 'choice')"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := testDB.Pool.Exec(ctx, "ALTER TABLE assets DROP CONSTRAINT test_keep_choice"); err != nil {
			t.Errorf("drop test constraint: %v", err)
		}
	}()
	key := "renamed"
	action := AttributeAction{Action: "update_attribute", Changes: AttributeChange{Key: &key}}
	p, err := repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.ChangeWithImpact(ctx, org.ID, uuid.Nil, field.ID, action, p.ConfirmationToken, false); err == nil {
		t.Fatal("expected injected failure")
	}
	loaded, err = repo.GetByID(ctx, field.ID)
	if err != nil || loaded.Key != "choice" {
		t.Fatalf("definition was not rolled back: %+v %v", loaded, err)
	}
	current, err := assets.GetByID(ctx, asset.ID)
	if err != nil || !strings.Contains(string(current.Attributes), "choice") {
		t.Fatalf("asset was not rolled back: %+v %v", current, err)
	}
}
