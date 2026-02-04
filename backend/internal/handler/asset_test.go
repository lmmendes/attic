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

// mockAssetRepo implements a minimal asset repository for testing
type mockAssetRepo struct {
	assets       map[uuid.UUID]*domain.Asset
	ListError    error
	GetError     error
	CreateError  error
	UpdateError  error
	DeleteError  error
}

func newMockAssetRepo() *mockAssetRepo {
	return &mockAssetRepo{
		assets: make(map[uuid.UUID]*domain.Asset),
	}
}

func (r *mockAssetRepo) addAsset(a *domain.Asset) {
	r.assets[a.ID] = a
}

func (r *mockAssetRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Asset, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.assets[id], nil
}

func (r *mockAssetRepo) GetByIDFull(_ context.Context, id uuid.UUID) (*domain.Asset, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.assets[id], nil
}

func (r *mockAssetRepo) List(_ context.Context, _ uuid.UUID, _ domain.AssetFilter, page domain.Pagination) ([]domain.Asset, int, error) {
	if r.ListError != nil {
		return nil, 0, r.ListError
	}
	assets := make([]domain.Asset, 0, len(r.assets))
	for _, a := range r.assets {
		assets = append(assets, *a)
	}
	total := len(assets)
	// Apply pagination
	start := page.Offset
	if start > len(assets) {
		start = len(assets)
	}
	end := start + page.Limit
	if end > len(assets) {
		end = len(assets)
	}
	return assets[start:end], total, nil
}

func (r *mockAssetRepo) Create(_ context.Context, a *domain.Asset) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = time.Now().UTC()
	r.assets[a.ID] = a
	return nil
}

func (r *mockAssetRepo) Update(_ context.Context, a *domain.Asset) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	a.UpdatedAt = time.Now().UTC()
	r.assets[a.ID] = a
	return nil
}

func (r *mockAssetRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.assets, id)
	return nil
}

func (r *mockAssetRepo) Search(_ context.Context, _ uuid.UUID, _ string, _ domain.Pagination) ([]domain.Asset, int, error) {
	return nil, 0, nil
}

func (r *mockAssetRepo) SetTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *mockAssetRepo) GetTotalValue(_ context.Context, _ uuid.UUID) (float64, error) {
	var total float64
	for _, a := range r.assets {
		if a.PurchasePrice != nil {
			total += *a.PurchasePrice
		}
	}
	return total, nil
}

// testAssetHandler wraps asset handler logic for testing
type testAssetHandler struct {
	assetRepo *mockAssetRepo
	orgID     uuid.UUID
}

func newTestAssetHandler() *testAssetHandler {
	return &testAssetHandler{
		assetRepo: newMockAssetRepo(),
		orgID:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testAssetHandler) listAssets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := domain.AssetFilter{
		Query: q.Get("q"),
	}

	if catID := q.Get("category_id"); catID != "" {
		if id, err := uuid.Parse(catID); err == nil {
			filter.CategoryID = &id
		}
	}

	page := domain.Pagination{Limit: limit, Offset: offset}
	assets, total, err := h.assetRepo.List(r.Context(), h.orgID, filter, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}

	if assets == nil {
		assets = []domain.Asset{}
	}

	// Convert to AssetWithImageURL
	assetsWithURLs := make([]AssetWithImageURL, len(assets))
	for i, asset := range assets {
		assetsWithURLs[i] = AssetWithImageURL{Asset: asset}
	}

	writeJSON(w, http.StatusOK, AssetListResponse{
		Assets: assetsWithURLs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *testAssetHandler) getAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	asset, err := h.assetRepo.GetByIDFull(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *testAssetHandler) createAsset(w http.ResponseWriter, r *http.Request) {
	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.CategoryID == "" {
		writeError(w, http.StatusBadRequest, "name and category_id are required")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category_id")
		return
	}

	asset := &domain.Asset{
		OrganizationID: h.orgID,
		CategoryID:     categoryID,
		Name:           req.Name,
		Description:    req.Description,
		Quantity:       req.Quantity,
		Attributes:     req.Attributes,
	}

	if asset.Quantity <= 0 {
		asset.Quantity = 1
	}

	if req.LocationID != nil {
		if id, err := uuid.Parse(*req.LocationID); err == nil {
			asset.LocationID = &id
		}
	}
	if req.PurchasePrice != nil {
		asset.PurchasePrice = req.PurchasePrice
	}

	if err := h.assetRepo.Create(r.Context(), asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create asset")
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

func (h *testAssetHandler) updateAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	var req UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	asset, err := h.assetRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category_id")
		return
	}

	asset.CategoryID = categoryID
	asset.Name = req.Name
	asset.Description = req.Description
	asset.Quantity = req.Quantity
	if asset.Quantity <= 0 {
		asset.Quantity = 1
	}

	if err := h.assetRepo.Update(r.Context(), asset); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update asset")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *testAssetHandler) deleteAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	if err := h.assetRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete asset")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *testAssetHandler) getAssetStats(w http.ResponseWriter, r *http.Request) {
	totalValue, err := h.assetRepo.GetTotalValue(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get asset stats")
		return
	}

	writeJSON(w, http.StatusOK, AssetStatsResponse{
		TotalValue: totalValue,
	})
}

func parseInt(s string) (int, error) {
	var n int
	err := json.Unmarshal([]byte(s), &n)
	return n, err
}

// Helper to create test router with chi URL params
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func createTestAsset(name string, categoryID uuid.UUID, price *float64) *domain.Asset {
	return &domain.Asset{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		CategoryID:     categoryID,
		Name:           name,
		Quantity:       1,
		PurchasePrice:  price,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_ListAssets_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	rec := httptest.NewRecorder()
	h.listAssets(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp AssetListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Assets) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp.Assets))
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func Test_ListAssets_WithAssets_ReturnsAll(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()
	h.assetRepo.addAsset(createTestAsset("Asset 1", catID, nil))
	h.assetRepo.addAsset(createTestAsset("Asset 2", catID, nil))
	h.assetRepo.addAsset(createTestAsset("Asset 3", catID, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	rec := httptest.NewRecorder()
	h.listAssets(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp AssetListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Assets) != 3 {
		t.Errorf("expected 3 assets, got %d", len(resp.Assets))
	}
	if resp.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Total)
	}
}

func Test_ListAssets_DefaultPagination(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/assets", nil)
	rec := httptest.NewRecorder()
	h.listAssets(rec, req)

	var resp AssetListResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", resp.Offset)
	}
}

func Test_GetAsset_ExistingAsset_ReturnsAsset(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()
	asset := createTestAsset("My Asset", catID, nil)
	h.assetRepo.addAsset(asset)

	req := httptest.NewRequest(http.MethodGet, "/api/assets/"+asset.ID.String(), nil)
	req = withChiURLParam(req, "id", asset.ID.String())
	rec := httptest.NewRecorder()
	h.getAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Asset
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "My Asset" {
		t.Errorf("expected name 'My Asset', got '%s'", resp.Name)
	}
}

func Test_GetAsset_NonExistentAsset_ReturnsNotFound(t *testing.T) {
	h := newTestAssetHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/assets/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getAsset(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetAsset_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/assets/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAsset_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()

	body := strings.NewReader(`{
		"name": "New Asset",
		"category_id": "` + catID.String() + `",
		"quantity": 5
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Asset
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "New Asset" {
		t.Errorf("expected name 'New Asset', got '%s'", resp.Name)
	}
	if resp.Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", resp.Quantity)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected ID to be generated")
	}
}

func Test_CreateAsset_MissingName_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()

	body := strings.NewReader(`{
		"category_id": "` + catID.String() + `"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAsset_MissingCategoryID_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	body := strings.NewReader(`{
		"name": "Asset without category"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAsset_InvalidCategoryID_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	body := strings.NewReader(`{
		"name": "Asset",
		"category_id": "not-a-uuid"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAsset_ZeroQuantity_DefaultsToOne(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()

	body := strings.NewReader(`{
		"name": "Asset",
		"category_id": "` + catID.String() + `",
		"quantity": 0
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	var resp domain.Asset
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Quantity != 1 {
		t.Errorf("expected quantity to default to 1, got %d", resp.Quantity)
	}
}

func Test_CreateAsset_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/assets", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateAsset_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()
	asset := createTestAsset("Original Name", catID, nil)
	h.assetRepo.addAsset(asset)

	newCatID := uuid.New()
	body := strings.NewReader(`{
		"name": "Updated Name",
		"category_id": "` + newCatID.String() + `",
		"quantity": 10
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/assets/"+asset.ID.String(), body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateAsset(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Asset
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", resp.Name)
	}
	if resp.Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", resp.Quantity)
	}
}

func Test_UpdateAsset_NonExistentAsset_ReturnsNotFound(t *testing.T) {
	h := newTestAssetHandler()
	nonExistentID := uuid.New()
	catID := uuid.New()

	body := strings.NewReader(`{
		"name": "Updated",
		"category_id": "` + catID.String() + `"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/assets/"+nonExistentID.String(), body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateAsset(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateAsset_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/assets/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteAsset_ExistingAsset_ReturnsNoContent(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()
	asset := createTestAsset("To Delete", catID, nil)
	h.assetRepo.addAsset(asset)

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/"+asset.ID.String(), nil)
	req = withChiURLParam(req, "id", asset.ID.String())
	rec := httptest.NewRecorder()
	h.deleteAsset(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	// Verify asset is deleted
	if _, exists := h.assetRepo.assets[asset.ID]; exists {
		t.Error("expected asset to be deleted from repository")
	}
}

func Test_DeleteAsset_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteAsset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_GetAssetStats_ReturnsCorrectTotalValue(t *testing.T) {
	h := newTestAssetHandler()
	catID := uuid.New()

	price1 := 100.50
	price2 := 200.75
	price3 := 50.25

	h.assetRepo.addAsset(createTestAsset("Asset 1", catID, &price1))
	h.assetRepo.addAsset(createTestAsset("Asset 2", catID, &price2))
	h.assetRepo.addAsset(createTestAsset("Asset 3", catID, &price3))
	h.assetRepo.addAsset(createTestAsset("Asset without price", catID, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/assets/stats", nil)
	rec := httptest.NewRecorder()
	h.getAssetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp AssetStatsResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	expectedTotal := price1 + price2 + price3
	if resp.TotalValue != expectedTotal {
		t.Errorf("expected total value %.2f, got %.2f", expectedTotal, resp.TotalValue)
	}
}

func Test_GetAssetStats_NoAssets_ReturnsZero(t *testing.T) {
	h := newTestAssetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/assets/stats", nil)
	rec := httptest.NewRecorder()
	h.getAssetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp AssetStatsResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.TotalValue != 0 {
		t.Errorf("expected total value 0, got %.2f", resp.TotalValue)
	}
}
