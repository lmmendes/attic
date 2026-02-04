package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

// mockCategoryRepo implements a minimal category repository for testing
type mockCategoryRepo struct {
	categories  map[uuid.UUID]*domain.Category
	assetCounts map[uuid.UUID]int
	ListError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{
		categories:  make(map[uuid.UUID]*domain.Category),
		assetCounts: make(map[uuid.UUID]int),
	}
}

func (r *mockCategoryRepo) addCategory(c *domain.Category) {
	r.categories[c.ID] = c
}

func (r *mockCategoryRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Category, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.categories[id], nil
}

func (r *mockCategoryRepo) GetByIDWithAttributes(_ context.Context, id uuid.UUID) (*domain.Category, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.categories[id], nil
}

func (r *mockCategoryRepo) List(_ context.Context, _ uuid.UUID) ([]domain.Category, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	cats := make([]domain.Category, 0, len(r.categories))
	for _, c := range r.categories {
		cats = append(cats, *c)
	}
	return cats, nil
}

func (r *mockCategoryRepo) ListTree(_ context.Context, _ uuid.UUID) ([]domain.Category, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	// Return only root categories (no parent) with children populated
	roots := make([]domain.Category, 0)
	for _, c := range r.categories {
		if c.ParentID == nil {
			// Find children
			cat := *c
			for _, child := range r.categories {
				if child.ParentID != nil && *child.ParentID == c.ID {
					cat.Children = append(cat.Children, *child)
				}
			}
			roots = append(roots, cat)
		}
	}
	return roots, nil
}

func (r *mockCategoryRepo) Create(_ context.Context, c *domain.Category) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = time.Now().UTC()
	r.categories[c.ID] = c
	return nil
}

func (r *mockCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	c.UpdatedAt = time.Now().UTC()
	r.categories[c.ID] = c
	return nil
}

func (r *mockCategoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.categories, id)
	return nil
}

func (r *mockCategoryRepo) SetAttributes(_ context.Context, _ uuid.UUID, _ []domain.CategoryAttributeAssignment) error {
	return nil
}

func (r *mockCategoryRepo) GetAssetCounts(_ context.Context, _ uuid.UUID) (map[uuid.UUID]int, error) {
	return r.assetCounts, nil
}

// testCategoryHandler wraps category handler logic for testing
type testCategoryHandler struct {
	categoryRepo *mockCategoryRepo
	orgID        uuid.UUID
}

func newTestCategoryHandler() *testCategoryHandler {
	return &testCategoryHandler{
		categoryRepo: newMockCategoryRepo(),
		orgID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testCategoryHandler) listCategories(w http.ResponseWriter, r *http.Request) {
	tree := r.URL.Query().Get("tree") == "true"

	var categories []domain.Category
	var err error

	if tree {
		categories, err = h.categoryRepo.ListTree(r.Context(), h.orgID)
	} else {
		categories, err = h.categoryRepo.List(r.Context(), h.orgID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}

	if categories == nil {
		categories = []domain.Category{}
	}

	writeJSON(w, http.StatusOK, categories)
}

func (h *testCategoryHandler) getCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	cat, err := h.categoryRepo.GetByIDWithAttributes(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

func (h *testCategoryHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	cat := &domain.Category{
		OrganizationID: h.orgID,
		Name:           req.Name,
		Description:    req.Description,
		Icon:           req.Icon,
	}

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		cat.ParentID = &parentID
	}

	if err := h.categoryRepo.Create(r.Context(), cat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

func (h *testCategoryHandler) updateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.categoryRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}

	cat.Name = req.Name
	cat.Description = req.Description
	cat.Icon = req.Icon

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		cat.ParentID = &parentID
	} else {
		cat.ParentID = nil
	}

	if err := h.categoryRepo.Update(r.Context(), cat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update category")
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

func (h *testCategoryHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	cat, err := h.categoryRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	if cat == nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	if cat.PluginID != nil {
		writeError(w, http.StatusForbidden, "cannot delete plugin-managed category")
		return
	}

	if err := h.categoryRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *testCategoryHandler) getCategoryAssetCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.categoryRepo.GetAssetCounts(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset counts")
		return
	}
	writeJSON(w, http.StatusOK, counts)
}

func createTestCategory(name string, parentID *uuid.UUID) *domain.Category {
	return &domain.Category{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ParentID:       parentID,
		Name:           name,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_ListCategories_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestCategoryHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()
	h.listCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Category
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func Test_ListCategories_WithCategories_ReturnsAll(t *testing.T) {
	h := newTestCategoryHandler()
	h.categoryRepo.addCategory(createTestCategory("Electronics", nil))
	h.categoryRepo.addCategory(createTestCategory("Books", nil))
	h.categoryRepo.addCategory(createTestCategory("Furniture", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()
	h.listCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Errorf("expected 3 categories, got %d", len(resp))
	}
}

func Test_ListCategories_TreeMode_ReturnsHierarchy(t *testing.T) {
	h := newTestCategoryHandler()
	electronics := createTestCategory("Electronics", nil)
	h.categoryRepo.addCategory(electronics)

	phones := createTestCategory("Phones", &electronics.ID)
	h.categoryRepo.addCategory(phones)

	laptops := createTestCategory("Laptops", &electronics.ID)
	h.categoryRepo.addCategory(laptops)

	books := createTestCategory("Books", nil)
	h.categoryRepo.addCategory(books)

	req := httptest.NewRequest(http.MethodGet, "/api/categories?tree=true", nil)
	rec := httptest.NewRecorder()
	h.listCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	// Should return only root categories
	if len(resp) != 2 {
		t.Errorf("expected 2 root categories, got %d", len(resp))
	}
}

func Test_GetCategory_ExistingCategory_ReturnsCategory(t *testing.T) {
	h := newTestCategoryHandler()
	cat := createTestCategory("Electronics", nil)
	h.categoryRepo.addCategory(cat)

	req := httptest.NewRequest(http.MethodGet, "/api/categories/"+cat.ID.String(), nil)
	req = withChiURLParam(req, "id", cat.ID.String())
	rec := httptest.NewRecorder()
	h.getCategory(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Electronics" {
		t.Errorf("expected name 'Electronics', got '%s'", resp.Name)
	}
}

func Test_GetCategory_NonExistentCategory_ReturnsNotFound(t *testing.T) {
	h := newTestCategoryHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/categories/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getCategory(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetCategory_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/categories/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCategory_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestCategoryHandler()

	body := strings.NewReader(`{
		"name": "New Category",
		"description": "A test category"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "New Category" {
		t.Errorf("expected name 'New Category', got '%s'", resp.Name)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected ID to be generated")
	}
}

func Test_CreateCategory_WithParent_SetsParentID(t *testing.T) {
	h := newTestCategoryHandler()
	parent := createTestCategory("Parent", nil)
	h.categoryRepo.addCategory(parent)

	body := strings.NewReader(`{
		"name": "Child Category",
		"parent_id": "` + parent.ID.String() + `"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.ParentID == nil || *resp.ParentID != parent.ID {
		t.Error("expected parent_id to be set")
	}
}

func Test_CreateCategory_WithIcon_SetsIcon(t *testing.T) {
	h := newTestCategoryHandler()

	body := strings.NewReader(`{
		"name": "Category with Icon",
		"icon": "📚"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Icon == nil || *resp.Icon != "📚" {
		t.Error("expected icon to be set")
	}
}

func Test_CreateCategory_MissingName_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	body := strings.NewReader(`{
		"description": "Category without name"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCategory_InvalidParentID_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	body := strings.NewReader(`{
		"name": "Category",
		"parent_id": "not-a-uuid"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCategory_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/categories", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateCategory_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestCategoryHandler()
	cat := createTestCategory("Original Name", nil)
	h.categoryRepo.addCategory(cat)

	body := strings.NewReader(`{
		"name": "Updated Name",
		"description": "Updated description"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/categories/"+cat.ID.String(), body)
	req = withChiURLParam(req, "id", cat.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateCategory(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", resp.Name)
	}
}

func Test_UpdateCategory_NonExistentCategory_ReturnsNotFound(t *testing.T) {
	h := newTestCategoryHandler()
	nonExistentID := uuid.New()

	body := strings.NewReader(`{
		"name": "Updated"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/categories/"+nonExistentID.String(), body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateCategory(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateCategory_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/categories/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateCategory_ClearParent_SetsParentToNil(t *testing.T) {
	h := newTestCategoryHandler()
	parent := createTestCategory("Parent", nil)
	h.categoryRepo.addCategory(parent)

	child := createTestCategory("Child", &parent.ID)
	h.categoryRepo.addCategory(child)

	// Update without parent_id should clear it
	body := strings.NewReader(`{
		"name": "Former Child"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/categories/"+child.ID.String(), body)
	req = withChiURLParam(req, "id", child.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateCategory(rec, req)

	var resp domain.Category
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.ParentID != nil {
		t.Error("expected parent_id to be nil after update")
	}
}

func Test_DeleteCategory_ExistingCategory_ReturnsNoContent(t *testing.T) {
	h := newTestCategoryHandler()
	cat := createTestCategory("To Delete", nil)
	h.categoryRepo.addCategory(cat)

	req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+cat.ID.String(), nil)
	req = withChiURLParam(req, "id", cat.ID.String())
	rec := httptest.NewRecorder()
	h.deleteCategory(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	// Verify category is deleted
	if _, exists := h.categoryRepo.categories[cat.ID]; exists {
		t.Error("expected category to be deleted from repository")
	}
}

func Test_DeleteCategory_NonExistentCategory_ReturnsNotFound(t *testing.T) {
	h := newTestCategoryHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.deleteCategory(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_DeleteCategory_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestCategoryHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/categories/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteCategory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteCategory_PluginManagedCategory_ReturnsForbidden(t *testing.T) {
	h := newTestCategoryHandler()
	pluginID := "google_books"
	cat := &domain.Category{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:           "Books",
		PluginID:       &pluginID,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	h.categoryRepo.addCategory(cat)

	req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+cat.ID.String(), nil)
	req = withChiURLParam(req, "id", cat.ID.String())
	rec := httptest.NewRecorder()
	h.deleteCategory(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for plugin-managed category, got %d", rec.Code)
	}
}

func Test_GetCategoryAssetCounts_ReturnsCorrectCounts(t *testing.T) {
	h := newTestCategoryHandler()
	cat1 := createTestCategory("Category 1", nil)
	cat2 := createTestCategory("Category 2", nil)
	h.categoryRepo.addCategory(cat1)
	h.categoryRepo.addCategory(cat2)

	// Set asset counts
	h.categoryRepo.assetCounts[cat1.ID] = 5
	h.categoryRepo.assetCounts[cat2.ID] = 10

	req := httptest.NewRequest(http.MethodGet, "/api/categories/counts", nil)
	rec := httptest.NewRecorder()
	h.getCategoryAssetCounts(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[uuid.UUID]int
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp[cat1.ID] != 5 {
		t.Errorf("expected count 5 for cat1, got %d", resp[cat1.ID])
	}
	if resp[cat2.ID] != 10 {
		t.Errorf("expected count 10 for cat2, got %d", resp[cat2.ID])
	}
}
