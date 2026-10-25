package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

// TestAssetQuickSearchCoverage verifies consistent search coverage and isolation across all query forms.
func TestAssetQuickSearchCoverage(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(testDB.TruncateAll(ctx))
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Search")
	must(err)
	other, err := fixtures.CreateOrganization(ctx, "Other")
	must(err)
	exec := func(sql string, args ...any) { t.Helper(); _, err := testDB.Pool.Exec(ctx, sql, args...); must(err) }
	features := domain.OrganizationFeatures{Attributes: true, Tags: true, Plugins: true}
	repo := NewAssetRepository(testDB.Pool)
	attr, err := fixtures.CreateAttribute(ctx, org.ID, "Model", "model", domain.AttributeTypeString)
	must(err)
	exec(`INSERT INTO assets(organization_id,name,description,notes,attributes) VALUES
 ($1,'Camera','Optical instrument','Stored in cabinet','{"model":"ZX-100%_B"}'),
 ($1,'Spare','Replacement','Stored in cabinet','{}'),
 ($2,'Camera','Optical instrument','Stored in cabinet','{"model":"ZX-100%_B"}')`, org.ID, other.ID)
	check := func(query string, want int) {
		t.Helper()
		qRule := domain.FilterNode{Kind: "rule", Field: "q", Operator: "search", Value: query}
		for _, filter := range []domain.AssetFilter{
			{Query: query, Features: &features},
			{Criteria: &domain.FilterCriteria{Version: 1, Query: query}, Features: &features},
			{Criteria: &domain.FilterCriteria{Version: 1, Expression: &qRule}, Features: &features},
		} {
			rows, total, err := repo.List(ctx, org.ID, filter, domain.Pagination{Limit: 1})
			must(err)
			wantRows := min(want, 1)
			if total != want || len(rows) != wantRows {
				t.Fatalf("%q: total=%d rows=%d want=%d", query, total, len(rows), want)
			}
		}
	}
	for query, want := range map[string]int{"came": 1, "optic": 1, "cabin": 2, "ZX-100%_": 1, "100%_": 1, "%": 1, "_": 1} {
		check(query, want)
	}
	// Notes and attribute-only edits must be reflected immediately.
	exec(`UPDATE assets SET notes='Moved to attic', attributes='{"model":"AB-200"}' WHERE organization_id=$1 AND name='Camera'`, org.ID)
	check("cabin", 1)
	check("attic", 1)
	check("ZX-100", 0)
	check("AB-2", 1)
	features.Attributes = false
	check("AB-2", 0)
	check("attic", 1)
	features.Attributes = true
	exec(`UPDATE attributes SET plugin_id='example' WHERE id=$1`, attr.ID)
	features.Plugins = false
	check("AB-2", 0)
	features.Plugins = true
	check("AB-2", 1)
	exec(`UPDATE attributes SET deleted_at=NOW() WHERE id=$1`, attr.ID)
	check("AB-2", 0)
	// Missing definitions in this organization must not resolve another organization's key.
	_, err = fixtures.CreateAttribute(ctx, other.ID, "Other model", "model", domain.AttributeTypeString)
	must(err)
	check("AB-2", 0)
	criteria := domain.FilterCriteria{Version: 1, Query: "attic", AttributeQuery: "absent"}
	rows, total, err := repo.List(ctx, org.ID, domain.AssetFilter{Criteria: &criteria, Features: &features}, domain.Pagination{Limit: 100})
	must(err)
	if total != 0 || len(rows) != 0 {
		t.Fatal("combined criteria did not intersect")
	}
	exec(`UPDATE assets SET deleted_at=NOW() WHERE organization_id=$1 AND name='Camera'`, org.ID)
	check("attic", 0)
}

// TestSearchNotesMigration verifies notes backfill and rollback while preserving asset timestamps.
func TestSearchNotesMigration(t *testing.T) {
	ctx := context.Background()
	// Transactional DDL keeps the shared test schema intact even on failure.
	tx, err := testDB.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	apply := func(direction string) {
		t.Helper()
		sql, err := os.ReadFile("../../migrations/000029_search_notes." + direction + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	apply("down")
	orgID, assetID := uuid.New(), uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'Migration');`, orgID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO assets(id,organization_id,name,notes,updated_at) VALUES($1,$2,'Camera','Cabinet','2001-01-01')`, assetID, orgID); err != nil {
		t.Fatal(err)
	}
	var before, timeAfter time.Time
	if err = tx.QueryRow(ctx, `SELECT updated_at FROM assets WHERE id=$1`, assetID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	check := func(want bool) {
		t.Helper()
		var matches bool
		if err = tx.QueryRow(ctx, `SELECT search_vector @@ attic_prefix_tsquery('cabin') FROM assets WHERE id=$1`, assetID).Scan(&matches); err != nil {
			t.Fatal(err)
		}
		if matches != want {
			t.Fatalf("notes match=%v want=%v", matches, want)
		}
	}
	check(false)
	apply("up")
	check(true)
	if err = tx.QueryRow(ctx, `SELECT updated_at FROM assets WHERE id=$1`, assetID).Scan(&timeAfter); err != nil {
		t.Fatal(err)
	}
	if !before.Equal(timeAfter) {
		t.Fatal("backfill changed updated_at")
	}
	apply("down")
	check(false)
	apply("up")
	check(true)
}
