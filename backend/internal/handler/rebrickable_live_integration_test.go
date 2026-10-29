package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/plugin"
	"github.com/lmmendes/attic/internal/plugin/rebrickable"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

// Run explicitly with ATTIC_TEST_REBRICKABLE_LIVE=true and ATTIC_REBRICKABLE_API_KEY.
// The catalog and image requests are real; imported assets live in a disposable database.
func TestRebrickableLiveImportWorkflow(t *testing.T) {
	if os.Getenv("ATTIC_TEST_REBRICKABLE_LIVE") != "true" {
		t.Skip("set ATTIC_TEST_REBRICKABLE_LIVE=true to test the live Rebrickable API")
	}
	client := rebrickable.NewClient()
	p := rebrickable.NewSets(client)
	if !p.Enabled() {
		t.Fatal("ATTIC_REBRICKABLE_API_KEY is required")
	}
	ctx := context.WithValue(t.Context(), organizationFeaturesContextKey{}, allFeaturesEnabled())
	db, err := testutil.NewTestDB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())
	org, err := testutil.NewFixtures(db.Pool).CreateOrganization(ctx, "Live Rebrickable import test")
	if err != nil {
		t.Fatal(err)
	}
	repos := &Repositories{
		Organizations: repository.NewOrganizationRepository(db.Pool),
		Assets:        repository.NewAssetRepository(db.Pool),
		Categories:    repository.NewCategoryRepository(db.Pool),
		Attributes:    repository.NewAttributeRepository(db.Pool),
		Attachments:   repository.NewAttachmentRepository(db.Pool),
	}
	registry := plugin.NewRegistry()
	if err := registry.Register(p); err != nil {
		t.Fatal(err)
	}
	storage := newMockStorage()
	h := NewPluginHandler(registry, repos, storage, org.ID)
	if err := h.InitializeEnabledPluginCategories(ctx); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Get("/api/plugins/{pluginId}/search", h.Search)
	router.Post("/api/plugins/{pluginId}/import", h.Import)
	const number = "75192-1"
	search := httptest.NewRecorder()
	router.ServeHTTP(search, httptest.NewRequest(http.MethodGet, "/api/plugins/rebrickable_sets/search?field=catalog_number&q="+number, nil).WithContext(ctx))
	if search.Code != http.StatusOK {
		t.Fatalf("live search: %d %s", search.Code, search.Body)
	}
	var results struct {
		Results []domain.SearchResult `json:"results"`
	}
	if err := json.Unmarshal(search.Body.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results.Results) != 1 || results.Results[0].ExternalID != number {
		t.Fatalf("unexpected search result: %s", search.Body)
	}
	imported := httptest.NewRecorder()
	router.ServeHTTP(imported, httptest.NewRequest(http.MethodPost, "/api/plugins/rebrickable_sets/import", strings.NewReader(`{"external_id":"`+number+`"}`)).WithContext(ctx))
	if imported.Code != http.StatusCreated {
		t.Fatalf("live import: %d %s", imported.Code, imported.Body)
	}
	var result ImportResponse
	if err := json.Unmarshal(imported.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	asset, err := repos.Assets.GetByID(ctx, result.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	var attributes map[string]any
	if err := json.Unmarshal(asset.Attributes, &attributes); err != nil {
		t.Fatal(err)
	}
	theme, ok := attributes["plugin.rebrickable_sets.theme"].(string)
	if !ok || strings.TrimSpace(theme) == "" {
		t.Fatalf("live import did not persist a theme: %#v", attributes)
	}
	if asset.Name != results.Results[0].Title || asset.CategoryID == nil || asset.ImportExternalID == nil || *asset.ImportExternalID != number {
		t.Fatalf("wrong imported asset: %#v", asset)
	}
	category, err := repos.Categories.GetByIDWithAttributes(ctx, *asset.CategoryID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, attribute := range category.Attributes {
		if attribute.Attribute.Key == "plugin.rebrickable_sets.theme" && attribute.Attribute.DataType == domain.AttributeTypeString {
			found = true
		}
	}
	if !found {
		t.Fatal("LEGO category is missing the Theme field")
	}
	attachments, err := repos.Attachments.ListByAsset(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 || len(storage.files[attachments[0].FileKey]) == 0 {
		t.Fatal("live import did not store the set image")
	}
	if asset.MainAttachmentID == nil || *asset.MainAttachmentID != attachments[0].ID || result.Asset.MainAttachmentID == nil || *result.Asset.MainAttachmentID != attachments[0].ID {
		t.Fatal("live import did not select the downloaded image as the default")
	}
	t.Logf("Imported %s (%s): theme=%q, pieces=%v, year=%v, image=%d bytes", asset.Name, number, theme, attributes["plugin.rebrickable_sets.piece_count"], attributes["plugin.rebrickable_sets.release_year"], len(storage.files[attachments[0].FileKey]))
}
