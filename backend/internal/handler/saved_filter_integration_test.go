package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestSavedFilterHTTPIntegration(t *testing.T) {
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
	f := testutil.NewFixtures(db.Pool)
	org, err := f.CreateOrganization(ctx, "Filter HTTP")
	must(err)
	owner, err := f.CreateUser(ctx, org.ID, "owner@filters.test")
	must(err)
	other, err := f.CreateUser(ctx, org.ID, "other@filters.test")
	must(err)
	admin, err := f.CreateUser(ctx, org.ID, "admin@filters.test")
	must(err)
	admin.Role = domain.UserRoleAdmin
	users := repository.NewUserRepository(db.Pool)
	must(users.Update(ctx, admin))
	repos := &Repositories{Assets: repository.NewAssetRepository(db.Pool), Attributes: repository.NewAttributeRepository(db.Pool), Organizations: repository.NewOrganizationRepository(db.Pool), SavedFilters: repository.NewSavedFilterRepository(db.Pool)}
	h := New(nil, repos, nil, org.ID)
	sessions := auth.NewSessionManager("filter-test-secret", 24)
	authentication, err := auth.NewMiddleware(ctx, auth.Config{})
	must(err)
	authentication.SetSessionManager(sessions)
	router := chi.NewRouter()
	router.Use(authentication.Authenticate)
	router.Use(auth.NewUserProvisioner(users, org.ID).LoadLocalUser)
	router.Use(h.LoadFeatures)
	router.Get("/api/assets", h.ListAssets)
	router.Post("/api/assets/search", h.SearchAssets)
	router.Get("/api/saved-filters", h.ListSavedFilters)
	router.Post("/api/saved-filters", h.CreateSavedFilter)
	router.Get("/api/saved-filters/{id}", h.GetSavedFilter)
	router.Put("/api/saved-filters/{id}", h.UpdateSavedFilter)
	router.Delete("/api/saved-filters/{id}", h.DeleteSavedFilter)
	request := func(method, path, body string, user *domain.User, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if user != nil {
			login := httptest.NewRecorder()
			must(sessions.CreateSession(login, req, user))
			for _, cookie := range login.Result().Cookies() {
				req.AddCookie(cookie)
			}
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, rec.Code, want, rec.Body)
		}
		return rec
	}
	attr, err := f.CreateAttribute(ctx, org.ID, "Number", "number", domain.AttributeTypeNumber)
	must(err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO assets(organization_id,name,attributes) VALUES($1,'Zero','{"number":0}'),($1,'Ten','{"number":10}')`, org.ID)
	must(err)
	criteria := domain.FilterCriteria{Version: 1, Expression: &domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: attr.ID.String(), DataType: "number", Operator: "eq", Value: float64(0)}}
	encode := func(v any) string { b, err := json.Marshal(v); must(err); return string(b) }
	search := encode(map[string]any{"criteria": criteria, "limit": 1, "offset": 0})
	response := request("POST", "/api/assets/search", search, owner, 200)
	var page AssetListResponse
	must(json.Unmarshal(response.Body.Bytes(), &page))
	if page.Total != 1 || len(page.Assets) != 1 || page.Assets[0].Name != "Zero" {
		t.Fatalf("search: %s", response.Body)
	}
	response = request("GET", "/api/assets?attribute_q=0&limit=1&offset=1", "", owner, 200)
	must(json.Unmarshal(response.Body.Bytes(), &page))
	if page.Total != 2 || len(page.Assets) != 1 {
		t.Fatal("GET query/paging mismatch")
	}
	request("POST", "/api/saved-filters", `{"name":"Private","criteria":{"version":1}}`, nil, 401)
	response = request("POST", "/api/saved-filters", encode(map[string]any{"name": " Private ", "pinned": true, "criteria": criteria}), owner, 201)
	var saved domain.SavedFilter
	must(json.Unmarshal(response.Body.Bytes(), &saved))
	if saved.Name != "Private" || !saved.Pinned || saved.CreatedAt.IsZero() {
		t.Fatal("create response")
	}
	path := "/api/saved-filters/" + saved.ID.String()
	for _, u := range []*domain.User{other, admin} {
		response = request("GET", "/api/saved-filters", "", u, 200)
		if strings.TrimSpace(response.Body.String()) != "[]" {
			t.Fatal("private list exposed")
		}
		request("GET", path, "", u, 404)
		request("PUT", path, `{"name":"Stolen"}`, u, 404)
		request("DELETE", path, "", u, 404)
	}
	request("PUT", path, `{"name":"Renamed"}`, owner, 200)
	response = request("GET", path, "", owner, 200)
	must(json.Unmarshal(response.Body.Bytes(), &saved))
	if saved.Name != "Renamed" || !saved.Pinned || saved.Criteria.Expression.AttributeID != attr.ID.String() {
		t.Fatal("rename lost criteria")
	}
	for i := 0; i < 4; i++ {
		request("POST", "/api/saved-filters", encode(map[string]any{"name": "Pinned " + string(rune('A'+i)), "pinned": true, "criteria": domain.FilterCriteria{Version: 1}}), owner, 201)
	}
	request("POST", "/api/saved-filters", `{"name":"Too many","pinned":true,"criteria":{"version":1}}`, owner, 400)
	request("PUT", path, `{"name":"Renamed","pinned":false}`, owner, 200)
	response = request("GET", path, "", owner, 200)
	must(json.Unmarshal(response.Body.Bytes(), &saved))
	if saved.Pinned {
		t.Fatal("unpin was not persisted")
	}
	for _, body := range []string{`{"criteria":{"version":1},"unknown":true}`, `{"criteria":{"version":1}} {}`, `{"criteria":{"version":2}}`, `{"criteria":{"version":1,"expression":{"kind":"group","match":"all","children":[]}}}`, `{"criteria":{"version":1,"attribute_q":"` + strings.Repeat("x", 1001) + `"}}`, strings.Repeat(" ", 65537) + `{}`} {
		request("POST", "/api/assets/search", body, owner, 400)
	}
	for _, body := range []string{`{"name":""}`, `{"name":"Missing"}`, `{"name":"` + strings.Repeat("a", 101) + `","criteria":{"version":1}}`} {
		request("POST", "/api/saved-filters", body, owner, 400)
	}
	t.Run("precise numeric saved criteria", func(t *testing.T) {
		_, err := db.Pool.Exec(ctx, `INSERT INTO assets(organization_id,name,attributes) VALUES ($1,'Exact number','{"number":9007199254740993}'),($1,'Adjacent number','{"number":9007199254740992}')`, org.ID)
		must(err)
		precise := criteria
		precise.Expression = &domain.FilterNode{Kind: "rule", Field: "attribute", AttributeID: attr.ID.String(), DataType: "number", Operator: "eq", Value: json.Number("9007199254740993")}
		created := request("POST", "/api/saved-filters", encode(map[string]any{"name": "Precise", "criteria": precise}), owner, 201)
		var record domain.SavedFilter
		decoder := json.NewDecoder(strings.NewReader(created.Body.String()))
		decoder.UseNumber()
		must(decoder.Decode(&record))
		precisePath := "/api/saved-filters/" + record.ID.String()
		request("PUT", precisePath, `{"name":"Precision retained"}`, owner, 200)
		loaded := request("GET", precisePath, "", owner, 200)
		decoder = json.NewDecoder(strings.NewReader(loaded.Body.String()))
		decoder.UseNumber()
		must(decoder.Decode(&record))
		if record.Criteria.Expression.Value != json.Number("9007199254740993") {
			t.Fatalf("numeric criterion changed: %s", loaded.Body)
		}
		found := request("POST", "/api/assets/search", encode(map[string]any{"criteria": record.Criteria}), owner, 200)
		var result AssetListResponse
		must(json.Unmarshal(found.Body.Bytes(), &result))
		if result.Total != 1 || result.Assets[0].Name != "Exact number" {
			t.Fatalf("numeric comparison rounded: %s", found.Body)
		}
		request("DELETE", precisePath, "", owner, 204)
	})
	t.Run("disabled authentication uses configured owner", func(t *testing.T) {
		middleware, err := auth.NewMiddleware(ctx, auth.Config{Disabled: true, DisabledUser: owner})
		must(err)
		local := chi.NewRouter()
		local.Use(middleware.Authenticate)
		local.Use(h.LoadFeatures)
		local.Get("/api/saved-filters/{id}", h.GetSavedFilter)
		rec := httptest.NewRecorder()
		local.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 {
			t.Fatalf("configured owner cannot access saved filter: %d %s", rec.Code, rec.Body)
		}
	})
	_, err = db.Pool.Exec(ctx, `UPDATE attributes SET name='Renamed number' WHERE id=$1`, attr.ID)
	must(err)
	request("POST", "/api/assets/search", search, owner, 200)
	_, err = db.Pool.Exec(ctx, `UPDATE attributes SET deleted_at=NOW() WHERE id=$1`, attr.ID)
	must(err)
	response = request("GET", path, "", owner, 200)
	must(json.Unmarshal(response.Body.Bytes(), &saved))
	if len(saved.Issues) == 0 || saved.Issues[0].Path != "expression" {
		t.Fatalf("stale issue: %s", response.Body)
	}
	response = request("GET", "/api/saved-filters", "", owner, 200)
	if !strings.Contains(response.Body.String(), `"issues"`) {
		t.Fatal("list omitted stale issues")
	}
	request("POST", "/api/assets/search", search, owner, 400)
	request("PUT", path, `{"name":"Repair later"}`, owner, 200)
	request("PUT", path, `{"name":"Repaired","criteria":{"version":1}}`, owner, 200)
	request("DELETE", path, "", owner, 204)
	request("GET", path, "", owner, 404)
	foreignOrg, err := f.CreateOrganization(ctx, "Other workspace")
	must(err)
	foreign, err := f.CreateUser(ctx, foreignOrg.ID, "foreign@filters.test")
	must(err)
	request("GET", "/api/saved-filters", "", foreign, 401)
}
