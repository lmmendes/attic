package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

// mockPlugin implements ImportPlugin interface for testing
type mockPlugin struct {
	id            string
	name          string
	description   string
	categoryName  string
	categoryDesc  string
	searchFields  []domain.SearchField
	attributes    []domain.PluginAttribute
	searchResults []domain.SearchResult
	searchErr     error
	fetchData     *domain.ImportData
	fetchErr      error
}

func (m *mockPlugin) ID() string                           { return m.id }
func (m *mockPlugin) Name() string                         { return m.name }
func (m *mockPlugin) Description() string                  { return m.description }
func (m *mockPlugin) Enabled() bool                        { return true }
func (m *mockPlugin) DisabledReason() string               { return "" }
func (m *mockPlugin) CategoryName() string                 { return m.categoryName }
func (m *mockPlugin) CategoryDescription() string          { return m.categoryDesc }
func (m *mockPlugin) SearchFields() []domain.SearchField   { return m.searchFields }
func (m *mockPlugin) Attributes() []domain.PluginAttribute { return m.attributes }

func (m *mockPlugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResults, nil
}

func (m *mockPlugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.fetchData, nil
}

// mockPluginRegistry for testing
type mockPluginRegistry struct {
	plugins map[string]domain.ImportPlugin
}

func newMockPluginRegistry() *mockPluginRegistry {
	return &mockPluginRegistry{
		plugins: make(map[string]domain.ImportPlugin),
	}
}

func (r *mockPluginRegistry) Register(p domain.ImportPlugin) {
	r.plugins[p.ID()] = p
}

func (r *mockPluginRegistry) Get(id string) (domain.ImportPlugin, bool) {
	p, ok := r.plugins[id]
	return p, ok
}

func (r *mockPluginRegistry) List() []domain.ImportPlugin {
	result := make([]domain.ImportPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

// mockCategoryRepoForPlugin extends category repo for plugin tests
type mockCategoryRepoForPlugin struct {
	*mockCategoryRepo
	byPluginID         map[string]*domain.Category
	assignments        map[uuid.UUID][]domain.CategoryAttributeAssignment
	setAttributesCalls int
}

func newMockCategoryRepoForPlugin() *mockCategoryRepoForPlugin {
	return &mockCategoryRepoForPlugin{
		mockCategoryRepo: newMockCategoryRepo(),
		byPluginID:       make(map[string]*domain.Category),
		assignments:      make(map[uuid.UUID][]domain.CategoryAttributeAssignment),
	}
}

func (m *mockCategoryRepoForPlugin) GetByPluginID(ctx context.Context, orgID uuid.UUID, pluginID string) (*domain.Category, error) {
	return m.byPluginID[pluginID], nil
}

func (m *mockCategoryRepoForPlugin) Create(ctx context.Context, category *domain.Category) error {
	if err := m.mockCategoryRepo.Create(ctx, category); err != nil {
		return err
	}
	if category.PluginID != nil {
		m.byPluginID[*category.PluginID] = category
	}
	return nil
}

func (m *mockCategoryRepoForPlugin) SetAttributes(_ context.Context, categoryID uuid.UUID, assignments []domain.CategoryAttributeAssignment) error {
	m.setAttributesCalls++
	m.assignments[categoryID] = append([]domain.CategoryAttributeAssignment(nil), assignments...)
	category := m.categories[categoryID]
	category.Attributes = make([]domain.CategoryAttribute, 0, len(assignments))
	for _, assignment := range assignments {
		category.Attributes = append(category.Attributes, domain.CategoryAttribute{
			CategoryID:  categoryID,
			AttributeID: assignment.AttributeID,
			Required:    assignment.Required,
			SortOrder:   assignment.SortOrder,
		})
	}
	return nil
}

// mockAttributeRepoForPlugin for plugin attribute creation
type mockAttributeRepoForPlugin struct {
	attributes map[string]*domain.Attribute
}

func newMockAttributeRepoForPlugin() *mockAttributeRepoForPlugin {
	return &mockAttributeRepoForPlugin{
		attributes: make(map[string]*domain.Attribute),
	}
}

func (m *mockAttributeRepoForPlugin) GetByKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.Attribute, error) {
	return m.attributes[key], nil
}

func (m *mockAttributeRepoForPlugin) Create(ctx context.Context, attr *domain.Attribute) error {
	attr.ID = uuid.New()
	attr.CreatedAt = time.Now()
	m.attributes[attr.Key] = attr
	return nil
}

// testPluginHandler wraps plugin handler logic for testing
type testPluginHandler struct {
	registry     *mockPluginRegistry
	categoryRepo *mockCategoryRepoForPlugin
	assetRepo    *mockAssetRepo
	attrRepo     *mockAttributeRepoForPlugin
	orgID        uuid.UUID
}

func newTestPluginHandler() *testPluginHandler {
	return &testPluginHandler{
		registry:     newMockPluginRegistry(),
		categoryRepo: newMockCategoryRepoForPlugin(),
		assetRepo:    newMockAssetRepo(),
		attrRepo:     newMockAttributeRepoForPlugin(),
		orgID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testPluginHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := h.registry.List()

	response := PluginListResponse{
		Plugins: make([]PluginResponse, 0, len(plugins)),
	}

	for _, p := range plugins {
		pr := PluginResponse{
			ID:                  p.ID(),
			Name:                p.Name(),
			Description:         p.Description(),
			CategoryName:        p.CategoryName(),
			CategoryDescription: p.CategoryDescription(),
			SearchFields:        p.SearchFields(),
			Attributes:          p.Attributes(),
		}

		cat, _ := h.categoryRepo.GetByPluginID(r.Context(), h.orgID, p.ID())
		if cat != nil {
			pr.CategoryID = &cat.ID
		}

		response.Plugins = append(response.Plugins, pr)
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *testPluginHandler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, "plugin not found")
		return
	}

	pr := PluginResponse{
		ID:                  p.ID(),
		Name:                p.Name(),
		Description:         p.Description(),
		CategoryName:        p.CategoryName(),
		CategoryDescription: p.CategoryDescription(),
		SearchFields:        p.SearchFields(),
		Attributes:          p.Attributes(),
	}

	cat, _ := h.categoryRepo.GetByPluginID(r.Context(), h.orgID, p.ID())
	if cat != nil {
		pr.CategoryID = &cat.ID
	}

	writeJSON(w, http.StatusOK, pr)
}

func (h *testPluginHandler) Search(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, "plugin '"+pluginID+"' not found")
		return
	}

	q := r.URL.Query()
	field := q.Get("field")
	query := q.Get("q")

	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	if len(query) < 2 {
		writeError(w, http.StatusBadRequest, "search query must be at least 2 characters")
		return
	}

	searchFields := p.SearchFields()
	if field == "" {
		if len(searchFields) > 0 {
			field = searchFields[0].Key
		}
	} else {
		validField := false
		for _, f := range searchFields {
			if f.Key == field {
				validField = true
				break
			}
		}
		if !validField {
			writeError(w, http.StatusBadRequest, "invalid search field '"+field+"'")
			return
		}
	}

	results, err := p.Search(r.Context(), field, query, 10)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		writeError(w, http.StatusBadGateway, "search service temporarily unavailable")
		return
	}

	if results == nil {
		results = []domain.SearchResult{}
	}

	writeJSON(w, http.StatusOK, SearchResponse{Results: results})
}

func (h *testPluginHandler) Import(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, "plugin '"+pluginID+"' not found")
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "external_id is required")
		return
	}

	importData, err := p.Fetch(r.Context(), req.ExternalID)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		writeError(w, http.StatusBadGateway, "failed to fetch data from external source")
		return
	}

	if importData.Name == "" {
		writeError(w, http.StatusBadGateway, "external source returned invalid data (missing name)")
		return
	}

	// Create or get category for plugin
	cat := h.categoryRepo.byPluginID[pluginID]
	if cat == nil {
		cat = &domain.Category{
			ID:             uuid.New(),
			OrganizationID: h.orgID,
			PluginID:       &pluginID,
			Name:           p.CategoryName(),
		}
		h.categoryRepo.byPluginID[pluginID] = cat
		h.categoryRepo.addCategory(cat)
	}

	attrsJSON, _ := json.Marshal(importData.Attributes)

	asset := &domain.Asset{
		OrganizationID:   h.orgID,
		CategoryID:       &cat.ID,
		Name:             importData.Name,
		Description:      importData.Description,
		Quantity:         1,
		Attributes:       attrsJSON,
		ImportPluginID:   &pluginID,
		ImportExternalID: &importData.ExternalID,
	}

	if err := h.assetRepo.Create(r.Context(), asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save imported item")
		return
	}

	asset.Category = cat

	writeJSON(w, http.StatusCreated, ImportResponse{Asset: asset})
}

func withPluginChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestEnsurePluginCategoryCreatesCategoryAndAttributesOnce(t *testing.T) {
	h := newTestPluginHandler()
	p := &mockPlugin{
		id:           "google_books",
		name:         "Google Books",
		categoryName: "Books",
		categoryDesc: "Books imported from Google Books",
		attributes: []domain.PluginAttribute{
			{Key: "books.isbn", Name: "ISBN", DataType: domain.AttributeTypeString},
			{Key: "books.author", Name: "Author", DataType: domain.AttributeTypeString},
		},
	}

	category, created, err := ensurePluginCategory(t.Context(), h.categoryRepo, h.attrRepo, h.orgID, p)
	if err != nil {
		t.Fatalf("initializing plugin category: %v", err)
	}
	if !created {
		t.Fatal("expected category to be created")
	}
	if category.Name != "Books" || category.PluginID == nil || *category.PluginID != p.ID() {
		t.Fatalf("unexpected category: %#v", category)
	}
	if len(h.attrRepo.attributes) != 2 {
		t.Fatalf("expected 2 plugin attributes, got %d", len(h.attrRepo.attributes))
	}
	if len(h.categoryRepo.assignments[category.ID]) != 2 {
		t.Fatalf("expected 2 category assignments, got %d", len(h.categoryRepo.assignments[category.ID]))
	}
	if h.categoryRepo.setAttributesCalls != 1 {
		t.Fatalf("expected attributes to be assigned once, got %d calls", h.categoryRepo.setAttributesCalls)
	}

	existing, created, err := ensurePluginCategory(t.Context(), h.categoryRepo, h.attrRepo, h.orgID, p)
	if err != nil {
		t.Fatalf("reinitializing plugin category: %v", err)
	}
	if created {
		t.Fatal("expected existing category to be reused")
	}
	if existing.ID != category.ID {
		t.Fatalf("expected category %s to be reused, got %s", category.ID, existing.ID)
	}
	if h.categoryRepo.setAttributesCalls != 1 {
		t.Fatalf("expected complete category to remain unchanged, got %d assignment calls", h.categoryRepo.setAttributesCalls)
	}
}

// Tests for ListPlugins

func Test_ListPlugins_NoPlugins_ReturnsEmptyArray(t *testing.T) {
	h := newTestPluginHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec := httptest.NewRecorder()

	h.ListPlugins(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response PluginListResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if len(response.Plugins) != 0 {
		t.Errorf("expected empty plugins array, got %d", len(response.Plugins))
	}
}

func Test_ListPlugins_WithPlugins_ReturnsAll(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		description:  "A test plugin",
		categoryName: "Test Category",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec := httptest.NewRecorder()

	h.ListPlugins(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response PluginListResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if len(response.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(response.Plugins))
	}
	if response.Plugins[0].ID != "test-plugin" {
		t.Errorf("expected plugin ID 'test-plugin', got '%s'", response.Plugins[0].ID)
	}
}

func Test_ListPlugins_WithCategory_IncludesCategoryID(t *testing.T) {
	h := newTestPluginHandler()

	pluginID := "test-plugin"
	h.registry.Register(&mockPlugin{
		id:           pluginID,
		name:         "Test Plugin",
		categoryName: "Test Category",
	})

	catID := uuid.New()
	h.categoryRepo.byPluginID[pluginID] = &domain.Category{
		ID:       catID,
		PluginID: &pluginID,
		Name:     "Test Category",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec := httptest.NewRecorder()

	h.ListPlugins(rec, req)

	var response PluginListResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Plugins[0].CategoryID == nil {
		t.Error("expected category ID to be set")
	}
	if *response.Plugins[0].CategoryID != catID {
		t.Errorf("expected category ID %s, got %s", catID, *response.Plugins[0].CategoryID)
	}
}

// Tests for GetPlugin

func Test_GetPlugin_Exists_ReturnsPlugin(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		description:  "Test description",
		categoryName: "Test Category",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.GetPlugin(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response PluginResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Name != "Test Plugin" {
		t.Errorf("expected name 'Test Plugin', got '%s'", response.Name)
	}
}

func Test_GetPlugin_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestPluginHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/nonexistent", nil)
	req = withPluginChiURLParam(req, "pluginId", "nonexistent")
	rec := httptest.NewRecorder()

	h.GetPlugin(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

// Tests for Search

func Test_Search_ValidQuery_ReturnsResults(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
		searchResults: []domain.SearchResult{
			{ExternalID: "123", Title: "Test Book", Subtitle: "A test"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin/search?q=test", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response SearchResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if len(response.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(response.Results))
	}
	if response.Results[0].Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got '%s'", response.Results[0].Title)
	}
}

func Test_Search_PluginNotFound_ReturnsNotFound(t *testing.T) {
	h := newTestPluginHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/nonexistent/search?q=test", nil)
	req = withPluginChiURLParam(req, "pluginId", "nonexistent")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_Search_MissingQuery_ReturnsBadRequest(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin/search", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Search_QueryTooShort_ReturnsBadRequest(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin/search?q=a", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Search_InvalidField_ReturnsBadRequest(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin/search?q=test&field=invalid", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Search_PluginError_ReturnsBadGateway(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		searchFields: []domain.SearchField{{Key: "title", Label: "Title"}},
		searchErr:    errors.New("api error"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/test-plugin/search?q=test", nil)
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

// Tests for Import

func Test_Import_ValidRequest_ReturnsCreatedAsset(t *testing.T) {
	h := newTestPluginHandler()

	desc := "Test description"
	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		categoryName: "Books",
		fetchData: &domain.ImportData{
			ExternalID:  "123",
			Name:        "Test Book",
			Description: &desc,
			Attributes:  map[string]any{"author": "Test Author"},
		},
	})

	body := `{"external_id":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var response ImportResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Asset == nil {
		t.Fatal("expected asset in response")
	}
	if response.Asset.Name != "Test Book" {
		t.Errorf("expected asset name 'Test Book', got '%s'", response.Asset.Name)
	}
}

func Test_Import_PluginNotFound_ReturnsNotFound(t *testing.T) {
	h := newTestPluginHandler()

	body := `{"external_id":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/nonexistent/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "nonexistent")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_Import_InvalidBody_ReturnsBadRequest(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:   "test-plugin",
		name: "Test Plugin",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString("invalid"))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Import_MissingExternalID_ReturnsBadRequest(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:   "test-plugin",
		name: "Test Plugin",
	})

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Import_FetchError_ReturnsBadGateway(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:       "test-plugin",
		name:     "Test Plugin",
		fetchErr: errors.New("api error"),
	})

	body := `{"external_id":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

func Test_Import_EmptyName_ReturnsBadGateway(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:   "test-plugin",
		name: "Test Plugin",
		fetchData: &domain.ImportData{
			ExternalID: "123",
			Name:       "", // Empty name
		},
	})

	body := `{"external_id":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

func Test_Import_CreatesCategory_WhenNotExists(t *testing.T) {
	h := newTestPluginHandler()

	h.registry.Register(&mockPlugin{
		id:           "test-plugin",
		name:         "Test Plugin",
		categoryName: "Books",
		fetchData: &domain.ImportData{
			ExternalID: "123",
			Name:       "Test Book",
		},
	})

	body := `{"external_id":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/test-plugin/import", bytes.NewBufferString(body))
	req = withPluginChiURLParam(req, "pluginId", "test-plugin")
	rec := httptest.NewRecorder()

	h.Import(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	// Verify category was created
	if h.categoryRepo.byPluginID["test-plugin"] == nil {
		t.Error("expected category to be created for plugin")
	}
}
