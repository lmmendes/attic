package repository

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestIGDBURLMigration(t *testing.T) {
	ctx := t.Context()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	fixture := testutil.NewFixtures(testDB.Pool)
	pluginID, key := "igdb_games", "plugin.igdb_games.games.url"
	var notices []string
	config := testDB.Pool.Config().ConnConfig.Copy()
	config.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { notices = append(notices, n.Message) }
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	up, err := os.ReadFile("../../migrations/000028_igdb_url_attribute.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000028_igdb_url_attribute.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	type expectation struct {
		id      uuid.UUID
		value   any
		deleted bool
	}
	var expected []expectation
	var definitionIDs []uuid.UUID
	var assignmentIDs []uuid.UUID
	values := []any{
		" https://www.igdb.com/games/example?q=a#details ", "http://localhost:8000/game",
		"https://example.com/a%20b?q=hello%20world#fragment", "http://[::1]:8000/game",
		"https://example.com/ends-in-v", "\u00a0https://example.com/unicode-trim\u3000", "\u00a0\u3000",
		"http://example.com/#a?%zz", "http://example.com/?good=%zz#a?%zz", "http://example.com/#a?%20",
		"http://[fe80::1%25en%C3%A9]", "http://[fe80::1%25ené]", "http://[fe80::1%25]", "http://[fe80::1%25en%0a]", "http://[fe80::1%25en%2f]", "http://[fe80::1%25en%00]",
		"http://[fe80::1%25en%20]", "http://example.com/path\u2000space", "http://example.com/path\u00a0space",
		"http://%C3%A9.example.com", "http://<example>.com", "http://exa\"mple.com",
		"http://example.com/?bad=%zz", "http://example.com/#bad%zz", "http://example.com/%",
		"http://example.com:", "http://example.com:99999", "http://example.com:-1", "http://foo:bar:baz",
		"http://::1", "http://[abc]", "http://[::1]invalid", "http://[::1]:abc",
		"http://example.com/<bad>", "http://example.com/a|b", "http://example.com/\x01", "http://exam_ple.com",
		"http://%65xample.com", "http://[fe80::1%25en0]", "http://[fe80::1%25en%30]",
		"https://例え.テスト/game", "HTTP://example.com", "https://example.com:443/a",
		"", " \t ", nil,
		"javascript:alert(1)", "/relative", "https:///missing", "https://user:secret@example.com",
		"https://example.com/bad%zz", "http://example.com:abc", "http://:8000",
		"https://example.com\\evil", "https://example.com/a b", "http://example.com/%00",
		map[string]any{"url": " https://example.com/labeled ", "label": " Invoice "},
		map[string]any{"url": "https://example.com/unchanged", "label": "Details"},
		map[string]any{"url": "https://example.com", "extra": "bad"},
		map[string]any{"url": "", "label": "orphan"}, map[string]any{"url": 42}, map[string]any{"url": ""}, map[string]any{"url": "  ", "label": " "}, true, 42,
	}
	invalid := 0
	for orgIndex := 0; orgIndex < 2; orgIndex++ {
		org, err := fixture.CreateOrganization(ctx, "URL migration")
		if err != nil {
			t.Fatal(err)
		}
		attr := &domain.Attribute{OrganizationID: org.ID, PluginID: &pluginID, Name: "IGDB URL", Key: key, DataType: domain.AttributeTypeString}
		if err := NewAttributeRepository(testDB.Pool).Create(ctx, attr); err != nil {
			t.Fatal(err)
		}
		definitionIDs = append(definitionIDs, attr.ID)
		cat, err := fixture.CreateCategory(ctx, org.ID, "Games", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := NewCategoryRepository(testDB.Pool).SetAttributes(ctx, cat.ID, []domain.CategoryAttributeAssignment{{AttributeID: attr.ID, SortOrder: 7}}); err != nil {
			t.Fatal(err)
		}
		assignmentIDs = append(assignmentIDs, cat.ID)
		for i, value := range values {
			input := value
			if text, ok := value.(string); ok {
				input = map[string]any{"url": text}
			}
			normalized, validationErr := domain.NormalizeURLAttribute(input, false)
			if validationErr != nil {
				invalid++
			}
			id := uuid.New()
			raw, _ := json.Marshal(map[string]any{key: value, "notes": "keep", "manual": map[string]string{"url": "https://example.com/manual"}})
			deleted := i%2 == 0
			_, err := conn.Exec(ctx, `INSERT INTO assets (id,organization_id,name,quantity,attributes,deleted_at) VALUES ($1,$2,'Game',1,$3,CASE WHEN $4 THEN now() END)`, id, org.ID, raw, deleted)
			if err != nil {
				t.Fatal(err)
			}
			var want any
			if normalized != nil {
				want = normalized
			}
			expected = append(expected, expectation{id: id, value: want, deleted: deleted})
		}
	}
	// A same-key field without IGDB ownership must remain untouched.
	org, err := fixture.CreateOrganization(ctx, "Custom field")
	if err != nil {
		t.Fatal(err)
	}
	custom := &domain.Attribute{OrganizationID: org.ID, Name: "Custom URL", Key: "custom.url", DataType: domain.AttributeTypeString}
	if err := NewAttributeRepository(testDB.Pool).Create(ctx, custom); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `UPDATE attributes SET key=$1 WHERE id=$2`, key, custom.ID); err != nil {
		t.Fatal(err)
	}
	customID := uuid.New()
	raw, _ := json.Marshal(map[string]any{key: "unchanged"})
	if _, err := conn.Exec(ctx, `INSERT INTO assets (id,organization_id,name,quantity,attributes) VALUES ($1,$2,'Custom',1,$3)`, customID, org.ID, raw); err != nil {
		t.Fatal(err)
	}
	// A failure after conversion must roll back values and definitions together.
	broken := strings.Replace(string(up), "\nCOMMIT;", "\nDO $$ BEGIN RAISE EXCEPTION 'test migration failure'; END $$;\nCOMMIT;", 1)
	if _, err := conn.Exec(ctx, broken); err == nil {
		t.Fatal("expected injected migration failure")
	}
	if _, err := conn.Exec(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	var beforeType string
	if err := conn.QueryRow(ctx, `SELECT data_type FROM attributes WHERE id=$1`, definitionIDs[0]).Scan(&beforeType); err != nil {
		t.Fatal(err)
	}
	if beforeType != "string" {
		t.Fatal("failed migration changed definition")
	}
	var beforeValue []byte
	if err := conn.QueryRow(ctx, `SELECT attributes->$1 FROM assets WHERE id=$2`, key, expected[0].id).Scan(&beforeValue); err != nil {
		t.Fatal(err)
	}
	if len(beforeValue) == 0 || beforeValue[0] != '"' {
		t.Fatalf("failed migration changed value: %s", beforeValue)
	}
	notices = nil
	if _, err := conn.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if len(notices) != invalid {
		t.Fatalf("logged %d removals, want %d: %v", len(notices), invalid, notices)
	}
	for _, notice := range notices {
		if !strings.Contains(notice, "organization_id=") || !strings.Contains(notice, "asset_id=") || !strings.Contains(notice, "attribute_key="+key) || strings.Contains(notice, "secret") {
			t.Fatalf("unexpected notice: %s", notice)
		}
	}
	for _, want := range expected {
		var raw []byte
		var deleted bool
		if err := conn.QueryRow(ctx, `SELECT attributes,deleted_at IS NOT NULL FROM assets WHERE id=$1`, want.id).Scan(&raw, &deleted); err != nil {
			t.Fatal(err)
		}
		var actual map[string]any
		if err := json.Unmarshal(raw, &actual); err != nil {
			t.Fatal(err)
		}
		wantRaw, _ := json.Marshal(want.value)
		var wantValue any
		json.Unmarshal(wantRaw, &wantValue)
		if !reflect.DeepEqual(actual[key], wantValue) || actual["notes"] != "keep" || deleted != want.deleted || actual["manual"].(map[string]any)["url"] != "https://example.com/manual" {
			t.Fatalf("asset %s: got %s, want URL %s", want.id, raw, wantRaw)
		}
		if want.value == nil {
			if _, exists := actual[key]; exists {
				t.Fatal("invalid value was not deleted")
			}
		}
	}
	for i, id := range definitionIDs {
		var dataType string
		var order int
		if err := conn.QueryRow(ctx, `SELECT a.data_type,ca.sort_order FROM attributes a JOIN category_attributes ca ON ca.attribute_id=a.id WHERE a.id=$1 AND ca.category_id=$2`, id, assignmentIDs[i]).Scan(&dataType, &order); err != nil {
			t.Fatal(err)
		}
		if dataType != "url" || order != 7 {
			t.Fatal("definition or category assignment changed incorrectly")
		}
	}
	// Migrated values remain editable, and the new import shape is accepted.
	assets := NewAssetRepository(testDB.Pool)
	migrated, err := assets.GetByID(ctx, expected[1].id)
	if err != nil {
		t.Fatal(err)
	}
	migrated.Name = "Edited migrated game"
	if err := assets.Update(ctx, migrated); err != nil {
		t.Fatal(err)
	}
	imported := &domain.Asset{OrganizationID: migrated.OrganizationID, Name: "New import", Quantity: 1, Attributes: json.RawMessage(`{"plugin.igdb_games.games.url":{"url":"https://www.igdb.com/games/new"}}`), ImportPluginID: &pluginID}
	if err := assets.Create(ctx, imported); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if len(notices) != invalid {
		t.Fatal("repeat upgrade logged additional removals")
	}
	if _, err := conn.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	for _, want := range expected {
		var raw []byte
		if err := conn.QueryRow(ctx, `SELECT attributes FROM assets WHERE id=$1`, want.id).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var actual map[string]any
		json.Unmarshal(raw, &actual)
		if want.value != nil && actual[key] != want.value.(map[string]string)["url"] {
			t.Fatalf("rollback failed: %s", raw)
		}
		if want.value == nil {
			if _, exists := actual[key]; exists {
				t.Fatal("rollback restored deleted value")
			}
		}
	}
	var kind, text string
	if err := conn.QueryRow(ctx, `SELECT d.data_type,a.attributes->>$1 FROM attributes d JOIN assets a ON a.organization_id=d.organization_id WHERE a.id=$2 AND d.id=$3`, key, customID, custom.ID).Scan(&kind, &text); err != nil {
		t.Fatal(err)
	}
	if kind != "string" || text != "unchanged" {
		t.Fatal("migration changed custom field")
	}
	for _, id := range definitionIDs {
		if err := conn.QueryRow(ctx, `SELECT data_type FROM attributes WHERE id=$1`, id).Scan(&kind); err != nil {
			t.Fatal(err)
		}
		if kind != "string" {
			t.Fatal("rollback did not restore definition")
		}
	}
}
