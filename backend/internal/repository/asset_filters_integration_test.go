package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestAssetFiltersSQL(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(testDB.TruncateAll(ctx))
	f := testutil.NewFixtures(testDB.Pool)
	org, err := f.CreateOrganization(ctx, "Filters")
	must(err)
	repo := NewAssetRepository(testDB.Pool)
	features := domain.OrganizationFeatures{Attributes: true, Categories: true, Collections: true, Tags: true, Locations: true, Conditions: true, Plugins: true}
	attrs := map[string]*domain.Attribute{}
	for key, typ := range map[string]domain.AttributeDataType{"s": "string", "t": "text", "n": "number", "b": "boolean", "d": "date"} {
		a, err := f.CreateAttribute(ctx, org.ID, key, key, typ)
		must(err)
		attrs[key] = a
	}
	exec := func(sql string, args ...any) { t.Helper(); _, err := testDB.Pool.Exec(ctx, sql, args...); must(err) }
	single, multi, o1, o2, o3 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO attributes(id,organization_id,name,key,data_type,selection_mode) VALUES ($1,$3,'Single','single','select','single'),($2,$3,'Multi','multi','select','multiple')`, single, multi, org.ID)
	exec(`INSERT INTO attribute_options(id,attribute_id,label,value) VALUES ($1,$4,'Ruby label','internal-red'),($2,$5,'Azure label','internal-blue'),($3,$5,'Gold label','internal-gold')`, o1, o2, o3, single, multi)
	for i, raw := range []string{
		`{"s":"Commodore 100%_\\ literal","t":"Long Memo","n":0,"b":false,"d":"1989-01-01","single":"internal-red","multi":["internal-blue","internal-gold"]}`,
		`{"s":"Other","t":"memo","n":10,"b":true,"d":"1990-01-01","multi":["internal-blue"]}`,
		`{"s":"","t":null,"n":"broken","b":"false","d":"2024-02-30","multi":[]}`,
		`{}`, `{"n":null,"d":123}`, `{"n":[],"d":"not-a-date"}`,
	} {
		exec(`INSERT INTO assets(organization_id,name,attributes) VALUES($1,$2,$3::jsonb)`, org.ID, fmt.Sprintf("Asset %d", i), raw)
	}
	check := func(t *testing.T, c domain.FilterCriteria, want int) {
		t.Helper()
		rows, total, err := repo.List(ctx, org.ID, domain.AssetFilter{Criteria: &c, Features: &features}, domain.Pagination{Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		if total != want || len(rows) != want {
			t.Fatalf("got rows=%d total=%d want=%d", len(rows), total, want)
		}
	}
	rule := func(key, op string, value any) domain.FilterNode {
		a := attrs[key]
		return domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: a.ID.String(), DataType: a.DataType, Operator: op, Value: value}
	}
	for query, want := range map[string]int{"commodore": 1, "MEMO": 2, "0": 3, "false": 2, "1989": 1, "Ruby": 1, "Azure": 2, "Gold": 1, "internal-red": 0, "100%_\\": 1, "%": 1, "_": 1, "\\": 1, "single": 0} {
		t.Run("quick/"+query, func(t *testing.T) { check(t, domain.FilterCriteria{Version: 1, AttributeQuery: query}, want) })
	}
	for _, tc := range []struct {
		name string
		n    domain.FilterNode
		want int
	}{
		{"number zero", rule("n", "eq", float64(0)), 1}, {"boolean false", rule("b", "eq", false), 1},
		{"text case", rule("s", "contains", "COMMODORE"), 1}, {"literal percent", rule("s", "contains", "%"), 1},
		{"number malformed", rule("n", "gte", float64(0)), 2}, {"date malformed", rule("d", "lte", "1990-01-01"), 2},
		{"empty missing null array", rule("n", "empty", nil), 3}, {"false populated", rule("b", "not_empty", nil), 3},
		{"string empty", rule("s", "empty", nil), 4},
		{"single option", domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: single.String(), DataType: "select", SelectionMode: "single", Operator: "any", Values: []string{o1.String()}}, 1},
		{"multi any", domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: multi.String(), DataType: "select", SelectionMode: "multiple", Operator: "any", Values: []string{o2.String(), o3.String()}}, 2},
		{"multi all", domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: multi.String(), DataType: "select", SelectionMode: "multiple", Operator: "all", Values: []string{o2.String(), o3.String()}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) { check(t, domain.FilterCriteria{Version: 1, Expression: &tc.n}, tc.want) })
	}
	for _, key := range []string{"n", "d"} {
		n := rule(key, "between", float64(0))
		n.Upper = float64(10)
		if key == "d" {
			n.Value = "1989-01-01"
			n.Upper = "1990-01-01"
		}
		t.Run("range/"+key, func(t *testing.T) { check(t, domain.FilterCriteria{Version: 1, Expression: &n}, 2) })
	}
	c1, c2 := uuid.New(), uuid.New()
	exec(`INSERT INTO collections(id,organization_id,name) VALUES($1,$3,'One'),($2,$3,'Two')`, c1, c2, org.ID)
	exec(`INSERT INTO asset_collections(asset_id,collection_id) SELECT id,$2 FROM assets WHERE organization_id=$1 AND name IN ('Asset 0','Asset 1')`, org.ID, c1)
	exec(`INSERT INTO asset_collections(asset_id,collection_id) SELECT id,$2 FROM assets WHERE organization_id=$1 AND name='Asset 0'`, org.ID, c2)
	for op, want := range map[string]int{"any": 2, "all": 1} {
		n := domain.FilterNode{Kind: "rule", Field: "collections", Operator: op, Values: []string{c1.String(), c2.String()}}
		t.Run("collections/"+op, func(t *testing.T) { check(t, domain.FilterCriteria{Version: 1, Expression: &n}, want) })
	}
	t1, t2 := uuid.New(), uuid.New()
	exec(`INSERT INTO tags(id,organization_id,name) VALUES($1,$3,'Vintage'),($2,$3,'Portable')`, t1, t2, org.ID)
	exec(`INSERT INTO asset_tags(asset_id,tag_id) SELECT id,$2 FROM assets WHERE organization_id=$1 AND name IN ('Asset 0','Asset 1')`, org.ID, t1)
	exec(`INSERT INTO asset_tags(asset_id,tag_id) SELECT id,$2 FROM assets WHERE organization_id=$1 AND name='Asset 0'`, org.ID, t2)
	for op, want := range map[string]int{"any": 2, "all": 1} {
		n := domain.FilterNode{Kind: "rule", Field: "tags", Operator: op, Values: []string{t1.String(), t2.String()}}
		t.Run("tags/"+op, func(t *testing.T) { check(t, domain.FilterCriteria{Version: 1, Expression: &n}, want) })
	}
	check(t, domain.FilterCriteria{Version: 1, TagIDs: []string{t1.String(), t2.String()}, TagMatch: "all"}, 1)
	check(t, domain.FilterCriteria{Version: 1, Query: "vint"}, 2)
	features.Tags = false
	tagRule := domain.FilterNode{Kind: "rule", Field: "tags", Operator: "any", Values: []string{t1.String()}}
	if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &tagRule}, features, 1); err == nil {
		t.Fatal("disabled tags rule was accepted")
	}
	rows, total, err := repo.List(ctx, org.ID, domain.AssetFilter{Query: "vint", Features: &features}, domain.Pagination{Limit: 100})
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("disabled tags affected search: rows=%d total=%d err=%v", len(rows), total, err)
	}
	features.Tags = true
	{
		rows, total, err := repo.List(ctx, org.ID, domain.AssetFilter{TagIDs: []uuid.UUID{t1, t2}, TagMatch: "all"}, domain.Pagination{Limit: 100})
		if err != nil || total != 1 || len(rows) != 1 {
			t.Fatalf("direct all-tag filter: rows=%d total=%d err=%v", len(rows), total, err)
		}
	}
	nested := domain.FilterNode{Kind: "group", Match: "all", Children: []domain.FilterNode{rule("n", "gte", float64(0)), {Kind: "group", Match: "any", Children: []domain.FilterNode{rule("b", "eq", false), rule("s", "eq", "absent")}}}}
	check(t, domain.FilterCriteria{Version: 1, Expression: &nested}, 1)
	c := domain.FilterCriteria{Version: 1, AttributeQuery: "memo"}
	rows, total, err = repo.List(ctx, org.ID, domain.AssetFilter{Criteria: &c, Features: &features}, domain.Pagination{Limit: 1, Offset: 1})
	must(err)
	if total != 2 || len(rows) != 1 {
		t.Fatalf("paging rows=%d total=%d", len(rows), total)
	}
	rows, total, err = repo.List(ctx, org.ID, domain.AssetFilter{Criteria: &c, Features: &features}, domain.Pagination{Limit: 1, Offset: 9})
	must(err)
	if total != 2 || len(rows) != 0 {
		t.Fatal("out of range paging lost total")
	}
	// Stable references resolve the current key after the data and definition are renamed.
	n := rule("s", "contains", "Commodore")
	exec(`UPDATE attributes SET key='renamed',name='Renamed' WHERE id=$1`, attrs["s"].ID)
	exec(`UPDATE assets SET attributes=(attributes-'s') || jsonb_build_object('renamed',attributes->'s') WHERE organization_id=$1 AND attributes ? 's'`, org.ID)
	check(t, domain.FilterCriteria{Version: 1, Expression: &n}, 1)
	exec(`UPDATE attributes SET plugin_id='test' WHERE id=$1`, attrs["s"].ID)
	features.Plugins = false
	check(t, domain.FilterCriteria{Version: 1, AttributeQuery: "Commodore"}, 0)
	if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &n}, features, 1); err == nil {
		t.Fatal("disabled plugin rule accepted")
	}
	features.Plugins = true
	exec(`UPDATE attributes SET deleted_at=NOW() WHERE id=$1`, attrs["s"].ID)
	if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &n}, features, 1); err == nil {
		t.Fatal("deleted attribute accepted")
	}
	check(t, domain.FilterCriteria{Version: 1, AttributeQuery: "Commodore"}, 0)
	t.Run("strict limits", func(t *testing.T) {
		leaf := rule("n", "eq", float64(0))
		deep := leaf
		for i := 0; i < 6; i++ {
			deep = domain.FilterNode{Kind: "group", Match: "all", Children: []domain.FilterNode{deep}}
		}
		many := domain.FilterNode{Kind: "group", Match: "all"}
		for i := 0; i < 51; i++ {
			many.Children = append(many.Children, leaf)
		}
		values := domain.FilterNode{Kind: "rule", Field: "collections", Operator: "any", Values: make([]string, 101)}
		invalid := []domain.FilterNode{deep, many, values, {Kind: "group", Match: "all"}, rule("n", "eq", "0"), rule("d", "eq", "2024-02-30"), rule("b", "eq", "false"), rule("n", "wat", float64(1)), {Kind: "group", Match: "all", Field: "attribute", Children: []domain.FilterNode{leaf}}}
		for i, n := range invalid {
			if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &n}, features, 1); err == nil {
				t.Errorf("invalid case %d accepted", i)
			}
		}
		if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 2}, features, 1); err == nil {
			t.Error("unknown version accepted")
		}
		deep = leaf
		for i := 0; i < 5; i++ {
			deep = domain.FilterNode{Kind: "group", Match: "all", Children: []domain.FilterNode{deep}}
		}
		_, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &deep}, features, 1)
		must(err)
	})
	t.Run("legacy null and scalar roots", func(t *testing.T) {
		exec(`INSERT INTO assets(organization_id,name,attributes) VALUES ($1,'Legacy null','null'),($1,'Legacy scalar','123'),($1,'Legacy array','[]')`, org.ID)
		check(t, domain.FilterCriteria{Version: 1, AttributeQuery: "memo"}, 2)
	})
	t.Run("precise numeric comparison", func(t *testing.T) {
		exec(`INSERT INTO assets(organization_id,name,attributes) VALUES ($1,'Large number','{"n":9007199254740993}'),($1,'Adjacent number','{"n":9007199254740992}')`, org.ID)
		n := rule("n", "eq", json.Number("9007199254740993"))
		check(t, domain.FilterCriteria{Version: 1, Expression: &n}, 1)
		n.Operator = "between"
		n.Upper = json.Number("9007199254740993.00000000001")
		check(t, domain.FilterCriteria{Version: 1, Expression: &n}, 1)
		n.Upper = json.Number("9007199254740992.99999999999")
		if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, Expression: &n}, features, 1); err == nil {
			t.Fatal("inverted precise range accepted")
		}
	})
	t.Run("category descendants and disabled plugin categories", func(t *testing.T) {
		parent, err := f.CreateCategory(ctx, org.ID, "Parent", nil)
		must(err)
		child, err := f.CreateCategory(ctx, org.ID, "Child", &parent.ID)
		must(err)
		plugin, err := f.CreateCategory(ctx, org.ID, "Plugin child", &parent.ID)
		must(err)
		exec(`UPDATE categories SET plugin_id='test' WHERE id=$1`, plugin.ID)
		exec(`UPDATE assets SET category_id=$2 WHERE organization_id=$1 AND name='Asset 0'`, org.ID, child.ID)
		exec(`UPDATE assets SET category_id=$2 WHERE organization_id=$1 AND name='Asset 1'`, org.ID, plugin.ID)
		check(t, domain.FilterCriteria{Version: 1, CategoryID: parent.ID.String()}, 2)
		features.Plugins = false
		check(t, domain.FilterCriteria{Version: 1, CategoryID: parent.ID.String()}, 1)
		if _, _, err := repo.CompileCriteria(ctx, org.ID, domain.FilterCriteria{Version: 1, CategoryID: plugin.ID.String()}, features, 1); err == nil {
			t.Fatal("hidden plugin category accepted")
		}
		features.Plugins = true
	})
	// Log reproducible plans without asserting machine-dependent timing thresholds.
	exec(`INSERT INTO assets(organization_id,name,attributes) SELECT $1,'Benchmark '||i,jsonb_build_object('t','fixture text '||i,'n',i) FROM generate_series(1,2000) i`, org.ID)
	exec(`ANALYZE assets`)
	for name, criteria := range map[string]domain.FilterCriteria{"quick": {Version: 1, AttributeQuery: "fixture text 1999"}, "equality": {Version: 1, Expression: func() *domain.FilterNode { n := rule("n", "eq", float64(1999)); return &n }()}} {
		predicate, args, err := repo.CompileCriteria(ctx, org.ID, criteria, features, 2)
		must(err)
		plans, err := testDB.Pool.Query(ctx, `EXPLAIN (ANALYZE, BUFFERS) SELECT a.id FROM assets a WHERE a.organization_id=$1 AND a.deleted_at IS NULL AND `+predicate, append([]any{org.ID}, args...)...)
		must(err)
		var lines []string
		for plans.Next() {
			var line string
			must(plans.Scan(&line))
			lines = append(lines, line)
		}
		must(plans.Err())
		plans.Close()
		t.Logf("%s (2000 fixtures):\n%s", name, strings.Join(lines, "\n"))
	}
}
