package repository

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

func TestPluginAttributeNamespaceMigrationPreservesUserDataAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	organization := &domain.Organization{Name: "Migration test"}
	if err := NewOrganizationRepository(testDB.Pool).Create(ctx, organization); err != nil {
		t.Fatal(err)
	}

	var categoryID uuid.UUID
	if err := testDB.Pool.QueryRow(ctx, `
		INSERT INTO categories (organization_id, plugin_id, name)
		VALUES ($1, 'google_books', 'Books') RETURNING id
	`, organization.ID).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `
		INSERT INTO attributes (organization_id, plugin_id, name, key, data_type)
		VALUES ($1, 'google_books', 'ISBN', 'books.isbn', 'string')
	`, organization.ID); err != nil {
		t.Fatal(err)
	}

	var assetID uuid.UUID
	if err := testDB.Pool.QueryRow(ctx, `
		INSERT INTO assets (organization_id, category_id, name, import_plugin_id, import_external_id, attributes)
		VALUES ($1, $2, 'Imported book', 'google_books', 'book-1',
			'{"books.isbn":"old","plugin.google_books.books.isbn":"already-migrated","shelf":"top"}'::jsonb)
		RETURNING id
	`, organization.ID, categoryID).Scan(&assetID); err != nil {
		t.Fatal(err)
	}

	migration, err := os.ReadFile("../../migrations/000017_namespace_plugin_attributes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := testDB.Pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("migration run %d failed: %v", i+1, err)
		}
	}

	var attributeKey string
	if err := testDB.Pool.QueryRow(ctx, `
		SELECT key FROM attributes WHERE organization_id = $1 AND plugin_id = 'google_books'
	`, organization.ID).Scan(&attributeKey); err != nil {
		t.Fatal(err)
	}
	if attributeKey != "plugin.google_books.books.isbn" {
		t.Fatalf("unexpected migrated definition key: %q", attributeKey)
	}

	var rawAttributes []byte
	if err := testDB.Pool.QueryRow(ctx, "SELECT attributes FROM assets WHERE id = $1", assetID).Scan(&rawAttributes); err != nil {
		t.Fatal(err)
	}
	var attributes map[string]any
	if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
		t.Fatal(err)
	}
	if attributes["plugin.google_books.books.isbn"] != "already-migrated" {
		t.Fatalf("namespaced value was not preserved: %#v", attributes)
	}
	if attributes["shelf"] != "top" {
		t.Fatalf("user-defined value was not preserved: %#v", attributes)
	}
	if _, exists := attributes["books.isbn"]; exists {
		t.Fatalf("legacy plugin key remains after migration: %#v", attributes)
	}
}

func TestPluginAttributeNamespaceMigrationReconcilesDuplicateDefinitions(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	organization := &domain.Organization{Name: "Duplicate migration test"}
	if err := NewOrganizationRepository(testDB.Pool).Create(ctx, organization); err != nil {
		t.Fatal(err)
	}

	var categoryID, legacyID, retainedID uuid.UUID
	if err := testDB.Pool.QueryRow(ctx, `
		INSERT INTO categories (organization_id, plugin_id, name)
		VALUES ($1, 'google_books', 'Books') RETURNING id
	`, organization.ID).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if err := testDB.Pool.QueryRow(ctx, `
		INSERT INTO attributes (organization_id, plugin_id, name, key, data_type)
		VALUES ($1, 'google_books', 'Legacy ISBN', 'books.isbn', 'string') RETURNING id
	`, organization.ID).Scan(&legacyID); err != nil {
		t.Fatal(err)
	}
	if err := testDB.Pool.QueryRow(ctx, `
		INSERT INTO attributes (organization_id, plugin_id, name, key, data_type)
		VALUES ($1, 'google_books', 'ISBN', 'plugin.google_books.books.isbn', 'string') RETURNING id
	`, organization.ID).Scan(&retainedID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool.Exec(ctx, `
		INSERT INTO category_attributes (category_id, attribute_id, required, sort_order)
		VALUES ($1, $2, TRUE, 2), ($1, $3, FALSE, 8)
	`, categoryID, legacyID, retainedID); err != nil {
		t.Fatal(err)
	}

	migration, err := os.ReadFile("../../migrations/000017_namespace_plugin_attributes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := testDB.Pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("migration run %d failed: %v", i+1, err)
		}
	}

	var definitionCount int
	if err := testDB.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM attributes
		WHERE organization_id = $1
		  AND plugin_id = 'google_books'
		  AND key = 'plugin.google_books.books.isbn'
	`, organization.ID).Scan(&definitionCount); err != nil {
		t.Fatal(err)
	}
	if definitionCount != 1 {
		t.Fatalf("expected one retained definition, got %d", definitionCount)
	}

	var attributeID uuid.UUID
	var required bool
	var sortOrder int
	if err := testDB.Pool.QueryRow(ctx, `
		SELECT attribute_id, required, sort_order
		FROM category_attributes
		WHERE category_id = $1
	`, categoryID).Scan(&attributeID, &required, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if attributeID != retainedID || !required || sortOrder != 2 {
		t.Fatalf("unexpected reconciled assignment: id=%s required=%t sort_order=%d", attributeID, required, sortOrder)
	}

	var legacyCount int
	if err := testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM attributes WHERE id = $1", legacyID).Scan(&legacyCount); err != nil {
		t.Fatal(err)
	}
	if legacyCount != 0 {
		t.Fatal("legacy duplicate definition was not removed")
	}
}
