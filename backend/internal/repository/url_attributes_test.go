package repository

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
	"os"
	"testing"
)

func TestURLValueValidation(t *testing.T) {
	for _, tc := range []struct {
		raw             string
		required, valid bool
	}{
		{`{"url":"https://paperless.local:8000/documents/123/?q=a#preview","label":"Invoice"}`, true, true},
		{`{"url":"http://localhost:8000"}`, false, true},
		{`{"url":" https://example.com ","label":"  "}`, false, true},
		{`{"url":""}`, false, true}, {`null`, false, true},
		{`{"url":""}`, true, false}, {`null`, true, false},
		{`{"url":"","label":"Invoice"}`, false, false},
		{`{"url":"javascript:alert(1)"}`, false, false},
		{`{"url":"data:text/html,test"}`, false, false},
		{`{"url":"/documents/1"}`, false, false},
		{`{"url":"https:///documents/1"}`, false, false},
		{`{"url":"https://user:password@example.com/documents/1"}`, false, false},
		{`{"url":"https://user@example.com/documents/1"}`, false, false},
		{`{"url":"https://example.com\\evil"}`, false, false},
		{`{"url":42}`, false, false}, {`{"url":"https://example.com","label":42}`, false, false},
		{`"https://example.com"`, false, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			var value any
			if err := json.Unmarshal([]byte(tc.raw), &value); err != nil {
				t.Fatal(err)
			}
			_, err := domain.NormalizeURLAttribute(value, tc.required)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestURLAssetLifecycle(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	f := testutil.NewFixtures(testDB.Pool)
	org, err := f.CreateOrganization(ctx, "URL tests")
	if err != nil {
		t.Fatal(err)
	}
	field := &domain.Attribute{OrganizationID: org.ID, Name: "Invoice", Key: "invoice", DataType: domain.AttributeTypeURL}
	if err = NewAttributeRepository(testDB.Pool).Create(ctx, field); err != nil {
		t.Fatal(err)
	}
	repo := NewAssetRepository(testDB.Pool)
	asset := &domain.Asset{OrganizationID: org.ID, Name: "Printer", Quantity: 1, Attributes: json.RawMessage(`{"invoice":{"url":"https://paperless.local/documents/1","label":"View invoice"}}`)}
	if err = repo.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err = json.Unmarshal(loaded.Attributes, &value); err != nil {
		t.Fatal(err)
	}
	if value["invoice"].(map[string]any)["label"] != "View invoice" {
		t.Fatalf("lost label: %s", loaded.Attributes)
	}
	loaded.Attributes = json.RawMessage(`{"invoice":{"url":"http://localhost:8000/documents/2"}}`)
	if err = repo.Update(ctx, loaded); err != nil {
		t.Fatal(err)
	}
	loaded.Attributes = json.RawMessage(`{"invoice":{"url":"javascript:alert(1)"}}`)
	if err = repo.Update(ctx, loaded); err == nil {
		t.Fatal("accepted unsafe URL on update")
	}
	asset.Attributes = loaded.Attributes
	if err = repo.Create(ctx, asset); err == nil {
		t.Fatal("accepted unsafe URL on create")
	}
}

func TestURLRequiredInheritanceAndRollback(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	f := testutil.NewFixtures(testDB.Pool)
	org, err := f.CreateOrganization(ctx, "URL required tests")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := f.CreateCategory(ctx, org.ID, "Parent", nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.CreateCategory(ctx, org.ID, "Child", &parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	field := &domain.Attribute{OrganizationID: org.ID, Name: "Manual", Key: "manual", DataType: domain.AttributeTypeURL}
	fields := NewAttributeRepository(testDB.Pool)
	if err = fields.Create(ctx, field); err != nil {
		t.Fatal(err)
	}
	if err = NewCategoryRepository(testDB.Pool).SetAttributes(ctx, parent.ID, []domain.CategoryAttributeAssignment{{AttributeID: field.ID, Required: true}}); err != nil {
		t.Fatal(err)
	}
	assets := NewAssetRepository(testDB.Pool)
	asset := &domain.Asset{OrganizationID: org.ID, CategoryID: &child.ID, Name: "Printer", Quantity: 1, Attributes: json.RawMessage(`{}`)}
	if err = assets.Create(ctx, asset); err == nil {
		t.Fatal("accepted missing inherited required URL")
	}
	asset.Attributes = json.RawMessage(`{"manual":{"url":"https://example.com/manual","label":"Read manual"},"notes":"preserved"}`)
	if err = assets.Create(ctx, asset); err != nil {
		t.Fatal(err)
	}
	stringType := domain.AttributeTypeString
	if _, err = fields.ChangeWithImpact(ctx, org.ID, uuid.New(), field.ID, AttributeAction{Action: "update_attribute", Changes: AttributeChange{DataType: &stringType}}, "", true); err == nil {
		t.Fatal("changed populated URL type")
	}
	tx, err := testDB.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	down, err := os.ReadFile("../../migrations/000027_url_attributes.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	if err = tx.QueryRow(ctx, "SELECT attributes FROM assets WHERE id=$1", asset.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err = json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	if values["manual"] != "https://example.com/manual" || values["notes"] != "preserved" {
		t.Fatalf("rollback lost data: %s", raw)
	}
	var typ string
	if err = tx.QueryRow(ctx, "SELECT data_type FROM attributes WHERE id=$1", field.ID).Scan(&typ); err != nil || typ != "string" {
		t.Fatalf("rollback definition: %s %v", typ, err)
	}
	up, err := os.ReadFile("../../migrations/000027_url_attributes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
}
