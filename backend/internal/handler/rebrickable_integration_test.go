package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/plugin"
	"github.com/lmmendes/attic/internal/plugin/rebrickable"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

type rebrickableTransport func(*http.Request) (*http.Response, error)

func (f rebrickableTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRebrickableImportWorkflow(t *testing.T) {
	ctx := t.Context()
	db, err := testutil.NewTestDB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	org, err := testutil.NewFixtures(db.Pool).CreateOrganization(ctx, "Rebrickable imports")
	if err != nil {
		t.Fatal(err)
	}
	repos := &Repositories{Organizations: repository.NewOrganizationRepository(db.Pool), Assets: repository.NewAssetRepository(db.Pool), Categories: repository.NewCategoryRepository(db.Pool), Attributes: repository.NewAttributeRepository(db.Pool), Attachments: repository.NewAttachmentRepository(db.Pool)}
	registry := plugin.NewRegistry()
	client := rebrickable.NewClient()
	sources := []domain.ImportPlugin{rebrickable.NewSets(client), rebrickable.NewMinifigs(client)}
	for _, p := range sources {
		if err := registry.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	storage := newMockStorage()
	h := NewPluginHandler(registry, repos, storage, org.ID)
	features := allFeaturesEnabled()
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), organizationFeaturesContextKey{}, features)))
		})
	})
	router.Use(New(nil, repos, nil, org.ID).RequireFeature("plugins"))
	router.Get("/api/plugins", h.ListPlugins)
	router.Get("/api/plugins/{pluginId}/search", h.Search)
	router.Post("/api/plugins/{pluginId}/import", h.Import)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	t.Setenv("ATTIC_REBRICKABLE_API_KEY", "")
	if err := h.InitializeEnabledPluginCategories(ctx); err != nil {
		t.Fatal(err)
	}
	for _, p := range sources {
		cat, err := repos.Categories.GetByPluginID(ctx, org.ID, p.ID())
		if err != nil || cat != nil {
			t.Fatalf("disabled category created: %#v %v", cat, err)
		}
		if rec := request("GET", "/api/plugins/"+p.ID()+"/search?q=test", ""); rec.Code != 503 {
			t.Fatalf("disabled search: %d", rec.Code)
		}
		if rec := request("POST", "/api/plugins/"+p.ID()+"/import", `{"external_id":"123-2"}`); rec.Code != 503 {
			t.Fatalf("disabled import: %d", rec.Code)
		}
	}
	t.Setenv("ATTIC_REBRICKABLE_API_KEY", "test-key")
	for range 2 {
		if err := h.InitializeEnabledPluginCategories(ctx); err != nil {
			t.Fatal(err)
		}
	}
	rec := request("GET", "/api/plugins", "")
	var listing PluginListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listing); err != nil || len(listing.Plugins) != 2 {
		t.Fatalf("registration: %s %v", rec.Body, err)
	}
	for _, p := range listing.Plugins {
		if !p.Enabled || p.CategoryID == nil {
			t.Fatalf("missing initialized category: %#v", p)
		}
	}
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	imageStatus := 200
	http.DefaultTransport = rebrickableTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "images.example.test" {
			return &http.Response{StatusCode: imageStatus, Header: http.Header{"Content-Type": {"image/png"}}, Body: io.NopCloser(strings.NewReader("test-image"))}, nil
		}
		if r.URL.Host != "rebrickable.com" {
			return original.RoundTrip(r)
		}
		if r.Header.Get("Authorization") != "key test-key" {
			t.Error("missing catalog credentials")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"set_num":"123-2","name":"LEGO item","num_parts":42,"year":2024,"set_url":"https://rebrickable.com/item/","set_img_url":"https://images.example.test/cover.png"}`))}, nil
	})
	for _, p := range sources {
		cat, err := repos.Categories.GetByPluginID(ctx, org.ID, p.ID())
		if err != nil || cat == nil {
			t.Fatalf("missing category: %v", err)
		}
		full, err := repos.Categories.GetByIDWithAttributes(ctx, cat.ID)
		if err != nil {
			t.Fatal(err)
		}
		if cat.Name != p.CategoryName() || len(full.Attributes) != len(p.Attributes()) {
			t.Fatalf("wrong category schema: %#v", full)
		}
		for _, a := range full.Attributes {
			if !strings.HasPrefix(a.Attribute.Key, "plugin."+p.ID()+".") {
				t.Fatalf("missing namespace: %s", a.Attribute.Key)
			}
		}
		rec := request("POST", "/api/plugins/"+p.ID()+"/import", `{"external_id":"123-2"}`)
		if rec.Code != 201 {
			t.Fatalf("import: %d %s", rec.Code, rec.Body)
		}
		var imported ImportResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &imported); err != nil {
			t.Fatal(err)
		}
		asset, err := repos.Assets.GetByID(ctx, imported.Asset.ID)
		if err != nil {
			t.Fatal(err)
		}
		if asset.Quantity != 1 || asset.CategoryID == nil || *asset.CategoryID != cat.ID || asset.ImportPluginID == nil || *asset.ImportPluginID != p.ID() || asset.ImportExternalID == nil || *asset.ImportExternalID != "123-2" {
			t.Fatalf("wrong asset: %#v", asset)
		}
		var attrs map[string]any
		if err := json.Unmarshal(asset.Attributes, &attrs); err != nil {
			t.Fatal(err)
		}
		if attrs["plugin."+p.ID()+".catalog_number"] != "123-2" || len(attrs) != len(p.Attributes()) {
			t.Fatalf("wrong attributes: %#v", attrs)
		}
		attachments, err := repos.Attachments.ListByAsset(ctx, asset.ID)
		if err != nil || len(attachments) != 1 {
			t.Fatalf("missing image attachment: %#v %v", attachments, err)
		}
		if string(storage.files[attachments[0].FileKey]) != "test-image" {
			t.Fatal("image not stored")
		}
	}
	features.Categories = false
	features.Attributes = false
	imageStatus = 404
	rec = request("POST", "/api/plugins/rebrickable_sets/import", `{"external_id":"123-2"}`)
	if rec.Code != 201 {
		t.Fatalf("optional image failure blocked import: %d %s", rec.Code, rec.Body)
	}
	var imported ImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Asset.CategoryID != nil || string(imported.Asset.Attributes) != "{}" {
		t.Fatalf("disabled features imported: %#v", imported.Asset)
	}
	features.Plugins = false
	for _, p := range sources {
		if rec := request("GET", "/api/plugins/"+p.ID()+"/search?q=test", ""); rec.Code != 404 {
			t.Fatalf("plugin search feature gate: %d", rec.Code)
		}
		if rec := request("POST", "/api/plugins/"+p.ID()+"/import", `{"external_id":"123-2"}`); rec.Code != 404 {
			t.Fatalf("plugin import feature gate: %d", rec.Code)
		}
	}
}
