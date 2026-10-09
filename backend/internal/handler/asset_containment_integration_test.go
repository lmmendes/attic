package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
	"github.com/lmmendes/attic/migrations"
)

func TestAssetContainmentHTTPIntegration(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	db, err := testutil.NewTestDB(ctx)
	must(err)
	defer db.Close(ctx)
	fixtures := testutil.NewFixtures(db.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Containment")
	must(err)
	user, err := fixtures.CreateUser(ctx, org.ID, "member@containment.test")
	must(err)
	otherOrg, err := fixtures.CreateOrganization(ctx, "Other workspace")
	must(err)
	garage, err := fixtures.CreateLocation(ctx, org.ID, "Garage", nil)
	must(err)
	attic, err := fixtures.CreateLocation(ctx, org.ID, "Attic", nil)
	must(err)
	assets := repository.NewAssetRepository(db.Pool)
	organizations := repository.NewOrganizationRepository(db.Pool)
	users := repository.NewUserRepository(db.Pool)
	h := New(nil, &Repositories{Assets: assets, Organizations: organizations}, nil, org.ID)
	sessions := auth.NewSessionManager("containment-test-secret", 24)
	authentication, err := auth.NewMiddleware(ctx, auth.Config{})
	must(err)
	authentication.SetSessionManager(sessions)
	router := chi.NewRouter()
	router.Use(authentication.Authenticate)
	router.Use(auth.NewUserProvisioner(users, org.ID).LoadLocalUser)
	router.Use(h.LoadFeatures)
	router.Get("/api/assets", h.ListAssets)
	router.Post("/api/assets", h.CreateAsset)
	router.Get("/api/assets/stats", h.GetAssetStats)
	router.Get("/api/assets/{id}", h.GetAsset)
	router.Put("/api/assets/{id}", h.UpdateAsset)
	router.Patch("/api/assets/{id}/parent", h.SetAssetParent)
	router.Delete("/api/assets/{id}", h.DeleteAsset)
	loginReq := httptest.NewRequest(http.MethodGet, "/", nil)
	login := httptest.NewRecorder()
	must(sessions.CreateSession(login, loginReq, user))
	request := func(method, path string, body any, signedIn bool) *httptest.ResponseRecorder {
		data, err := json.Marshal(body)
		must(err)
		req := httptest.NewRequest(method, path, strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		if signedIn {
			for _, cookie := range login.Result().Cookies() {
				req.AddCookie(cookie)
			}
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	call := func(method, path string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		response := request(method, path, body, true)
		if response.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.Code, want, response.Body)
		}
		return response
	}
	decode := func(response *httptest.ResponseRecorder) domain.Asset {
		t.Helper()
		var asset domain.Asset
		must(json.Unmarshal(response.Body.Bytes(), &asset))
		return asset
	}
	create := func(body map[string]any) domain.Asset {
		t.Helper()
		return decode(call("POST", "/api/assets", body, 201))
	}
	get := func(id uuid.UUID) domain.Asset {
		t.Helper()
		return decode(call("GET", "/api/assets/"+id.String(), nil, 200))
	}
	parent := func(id uuid.UUID, target any, status int) {
		t.Helper()
		call("PATCH", "/api/assets/"+id.String()+"/parent", map[string]any{"parent_id": target}, status)
	}
	update := func(id uuid.UUID, body map[string]any, status int) {
		t.Helper()
		call("PUT", "/api/assets/"+id.String(), body, status)
	}
	checkLocation := func(id uuid.UUID, location *uuid.UUID) {
		t.Helper()
		a, err := assets.GetByID(ctx, id)
		must(err)
		if a == nil || (location == nil && a.LocationID != nil) || (location != nil && (a.LocationID == nil || *a.LocationID != *location)) {
			t.Fatalf("wrong location for %s: %+v", id, a)
		}
	}

	t.Run("nested movement reparent detach and cost", func(t *testing.T) {
		crate := create(map[string]any{"name": "Crate", "location_id": garage.ID, "purchase_price": 10, "quantity": 2})
		box := create(map[string]any{"name": "CPU box", "parent_id": crate.ID, "purchase_price": 5})
		cpu := create(map[string]any{"name": "CPU", "parent_id": box.ID, "purchase_price": 20, "quantity": 3})
		zero := create(map[string]any{"name": "Free adapter", "parent_id": box.ID, "purchase_price": 0})
		unknown := create(map[string]any{"name": "Unknown cable", "parent_id": crate.ID})
		detail := get(crate.ID)
		if detail.ContainmentSummary.TotalValue != 85 || detail.ContainmentSummary.UnpricedAssetCount != 1 {
			t.Fatalf("wrong summary: %+v", detail.ContainmentSummary)
		}
		if got := get(cpu.ID); got.Parent == nil || got.Parent.ID != box.ID {
			t.Fatalf("missing parent: %+v", got)
		}
		var stats AssetStatsResponse
		must(json.Unmarshal(call("GET", "/api/assets/stats", nil, 200).Body.Bytes(), &stats))
		if stats.TotalValue != 85 {
			t.Fatalf("rollup was double counted: %+v", stats)
		}
		update(crate.ID, map[string]any{"name": "Moved crate", "location_id": attic.ID, "purchase_price": 10, "quantity": 2}, 200)
		for _, id := range []uuid.UUID{crate.ID, box.ID, cpu.ID, zero.ID, unknown.ID} {
			checkLocation(id, &attic.ID)
		}
		update(cpu.ID, map[string]any{"name": "CPU renamed", "purchase_price": 20, "quantity": 3}, 200)
		if get(cpu.ID).ParentID == nil {
			t.Fatal("omitted parent detached child")
		}
		checkLocation(cpu.ID, &attic.ID)
		update(cpu.ID, map[string]any{"name": "Bad edit", "location_id": garage.ID}, 400)
		if get(cpu.ID).Name != "CPU renamed" {
			t.Fatal("failed save changed fields")
		}
		parent(box.ID, nil, 200)
		checkLocation(cpu.ID, &attic.ID)
		parent(box.ID, crate.ID, 200)
		update(crate.ID, map[string]any{"name": "No location"}, 200)
		for _, id := range []uuid.UUID{crate.ID, box.ID, cpu.ID} {
			checkLocation(id, nil)
		}
		parent(box.ID, nil, 200)
		if got := get(cpu.ID); got.ParentID == nil || *got.ParentID != box.ID {
			t.Fatal("detaching container detached its contents")
		}
		update(box.ID, map[string]any{"name": "Moved box", "location_id": garage.ID}, 200)
		parent(box.ID, crate.ID, 200)
		checkLocation(cpu.ID, nil)
		update(box.ID, map[string]any{"name": "Detached box", "parent_id": nil, "location_id": garage.ID}, 200)
		checkLocation(cpu.ID, &garage.ID)
	})

	t.Run("atomic contents saves and final graph validation", func(t *testing.T) {
		pc := create(map[string]any{"name": "PC", "location_id": garage.ID})
		old := create(map[string]any{"name": "Old GPU", "parent_id": pc.ID})
		replacement := create(map[string]any{"name": "Replacement GPU", "location_id": attic.ID})
		update(pc.ID, map[string]any{"name": "Upgraded PC", "location_id": garage.ID, "add_child_ids": []uuid.UUID{replacement.ID}, "remove_child_ids": []uuid.UUID{old.ID}}, 200)
		if get(old.ID).ParentID != nil || get(replacement.ID).ParentID == nil {
			t.Fatal("upgrade did not atomically swap parts")
		}
		checkLocation(old.ID, &garage.ID)
		checkLocation(replacement.ID, &garage.ID)
		update(pc.ID, map[string]any{"name": "Must roll back", "location_id": attic.ID, "add_child_ids": []uuid.UUID{old.ID}, "remove_child_ids": []uuid.UUID{uuid.New()}}, 409)
		if get(pc.ID).Name != "Upgraded PC" || get(old.ID).ParentID != nil {
			t.Fatal("conflict partially saved asset")
		}
		checkLocation(replacement.ID, &garage.ID)
		update(pc.ID, map[string]any{"name": "Must roll back", "add_child_ids": []uuid.UUID{old.ID}, "remove_child_ids": []uuid.UUID{old.ID}}, 400)
		parent(pc.ID, replacement.ID, 400)
		parent(pc.ID, pc.ID, 400)
		update(pc.ID, map[string]any{"name": "Must roll back", "add_child_ids": []uuid.UUID{pc.ID}}, 400)
		// Valid final graph: detach the child before making the edited asset its child.
		update(pc.ID, map[string]any{"name": "Reversed relationship", "parent_id": replacement.ID, "remove_child_ids": []uuid.UUID{replacement.ID}}, 200)
		if got := get(pc.ID); got.ParentID == nil || *got.ParentID != replacement.ID {
			t.Fatal("valid final graph rejected")
		}
	})

	t.Run("permissions validation filters and pagination", func(t *testing.T) {
		foreign := &domain.Asset{OrganizationID: otherOrg.ID, Name: "Foreign", Quantity: 1}
		must(assets.Create(ctx, foreign))
		root := create(map[string]any{"name": "Filter root"})
		child := create(map[string]any{"name": "Filter child", "parent_id": root.ID})
		leaf := create(map[string]any{"name": "Filter leaf", "parent_id": child.ID})
		sibling := create(map[string]any{"name": "Filter sibling", "parent_id": root.ID})
		deleted := create(map[string]any{"name": "Deleted"})
		call("DELETE", "/api/assets/"+deleted.ID.String(), nil, 204)
		for _, target := range []any{foreign.ID, deleted.ID, uuid.New(), "broken", uuid.Nil, true} {
			parent(child.ID, target, 400)
		}
		call("PATCH", "/api/assets/"+child.ID.String()+"/parent", map[string]any{}, 400)
		call("PATCH", "/api/assets/"+foreign.ID.String()+"/parent", map[string]any{"parent_id": nil}, 404)
		call("PATCH", "/api/assets/"+deleted.ID.String()+"/parent", map[string]any{"parent_id": nil}, 404)
		if response := request("PATCH", "/api/assets/"+child.ID.String()+"/parent", map[string]any{"parent_id": nil}, false); response.Code != 401 {
			t.Fatal("unauthenticated mutation accepted")
		}
		update(root.ID, map[string]any{"name": "No foreign children", "add_child_ids": []uuid.UUID{foreign.ID}}, 400)
		var page AssetListResponse
		must(json.Unmarshal(call("GET", "/api/assets?parent_id="+root.ID.String()+"&limit=1&offset=1", nil, 200).Body.Bytes(), &page))
		if page.Total != 2 || len(page.Assets) != 1 || page.Assets[0].Parent == nil {
			t.Fatalf("bad contents page: %+v", page)
		}

		must(json.Unmarshal(call("GET", "/api/assets?parent_id="+root.ID.String()+"&limit=100", nil, 200).Body.Bytes(), &page))
		for _, node := range page.Assets {
			want := 0
			if node.ID == child.ID {
				want = 1
			}
			if node.ChildCount != want {
				t.Fatalf("wrong child count for %s: got %d want %d", node.Name, node.ChildCount, want)
			}
		}
		call("GET", "/api/assets?parent_id="+foreign.ID.String(), nil, 404)
		call("GET", "/api/assets?exclude_subtree_of=bad", nil, 400)
		for param, excluded := range map[string][]uuid.UUID{"exclude_subtree_of": {root.ID, child.ID, leaf.ID, sibling.ID}, "exclude_ancestors_of": {root.ID, child.ID, leaf.ID}} {
			ref := root.ID
			if param == "exclude_ancestors_of" {
				ref = leaf.ID
			}
			must(json.Unmarshal(call("GET", "/api/assets?"+param+"="+ref.String()+"&limit=100", nil, 200).Body.Bytes(), &page))
			for _, a := range page.Assets {
				for _, id := range excluded {
					if a.ID == id {
						t.Fatalf("%s returned excluded %s", param, id)
					}
				}
			}
		}
		must(json.Unmarshal(call("GET", "/api/assets?q=Filter&limit=100", nil, 200).Body.Bytes(), &page))
		if page.Total != 4 {
			t.Fatalf("ordinary search hid children: %+v", page)
		}
	})

	t.Run("delete preserves nested children and independent data", func(t *testing.T) {
		root := create(map[string]any{"name": "Delete root", "location_id": garage.ID})
		box := create(map[string]any{"name": "Surviving box", "parent_id": root.ID, "purchase_price": 12})
		child := create(map[string]any{"name": "Surviving part", "parent_id": box.ID})
		_, err := fixtures.CreateWarranty(ctx, box.ID, "Warranty provider", box.CreatedAt.AddDate(1, 0, 0))
		must(err)
		_, err = fixtures.CreateAttachment(ctx, box.ID, "Receipt.pdf", "receipts/test.pdf")
		must(err)
		call("DELETE", "/api/assets/"+root.ID.String(), nil, 204)
		survivor := get(box.ID)
		if survivor.ParentID != nil || survivor.Warranty == nil || *survivor.PurchasePrice != 12 || get(child.ID).ParentID == nil {
			t.Fatalf("delete lost contents data: %+v", survivor)
		}
		checkLocation(box.ID, &garage.ID)
		checkLocation(child.ID, &garage.ID)
		var count int
		must(db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM attachments WHERE asset_id=$1`, box.ID).Scan(&count))
		if count != 1 {
			t.Fatal("child attachments lost")
		}
	})

	t.Run("hidden locations still synchronize", func(t *testing.T) {
		root := create(map[string]any{"name": "Hidden root", "location_id": garage.ID})
		features, err := organizations.GetFeatures(ctx, org.ID)
		must(err)
		features.Locations = false
		must(organizations.UpdateFeatures(ctx, org.ID, features))
		defer func() { features.Locations = true; must(organizations.UpdateFeatures(ctx, org.ID, features)) }()
		child := create(map[string]any{"name": "Hidden child", "parent_id": root.ID})
		checkLocation(child.ID, &garage.ID)
		if get(child.ID).LocationID != nil {
			t.Fatal("hidden location exposed")
		}
		update(root.ID, map[string]any{"name": "Hidden root edited"}, 200)
		checkLocation(root.ID, &garage.ID)
		checkLocation(child.ID, &garage.ID)
	})

	t.Run("concurrent opposing attachments cannot create a cycle", func(t *testing.T) {
		a := create(map[string]any{"name": "Concurrent A"})
		b := create(map[string]any{"name": "Concurrent B"})
		start := make(chan struct{})
		responses := make(chan int, 2)
		var wg sync.WaitGroup
		for _, pair := range [][2]uuid.UUID{{a.ID, b.ID}, {b.ID, a.ID}} {
			wg.Add(1)
			go func(pair [2]uuid.UUID) {
				defer wg.Done()
				<-start
				responses <- request("PATCH", "/api/assets/"+pair[0].String()+"/parent", map[string]any{"parent_id": pair[1]}, true).Code
			}(pair)
		}
		close(start)
		wg.Wait()
		close(responses)
		counts := map[int]int{}
		for code := range responses {
			counts[code]++
		}
		if counts[200] != 1 || counts[400] != 1 {
			t.Fatalf("unexpected concurrent results: %+v", counts)
		}
	})

	t.Run("migration down and up", func(t *testing.T) {
		tx, err := db.Pool.Begin(ctx)
		must(err)
		defer tx.Rollback(ctx)
		down, err := migrations.FS.ReadFile("000025_asset_containment.down.sql")
		must(err)
		up, err := migrations.FS.ReadFile("000025_asset_containment.up.sql")
		must(err)
		_, err = tx.Exec(ctx, string(down))
		must(err)
		var exists bool
		must(tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='assets' AND column_name='parent_id')`).Scan(&exists))
		if exists {
			t.Fatal("down migration retained parent_id")
		}
		_, err = tx.Exec(ctx, string(up))
		must(err)
		must(tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='assets' AND column_name='parent_id')`).Scan(&exists))
		if !exists {
			t.Fatal("up migration did not add parent_id")
		}
	})
}
