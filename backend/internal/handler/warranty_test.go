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

// mockWarrantyRepo implements a minimal warranty repository for testing
type mockWarrantyRepo struct {
	warranties  map[uuid.UUID]*domain.Warranty // keyed by AssetID
	ListError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
}

func newMockWarrantyRepo() *mockWarrantyRepo {
	return &mockWarrantyRepo{
		warranties: make(map[uuid.UUID]*domain.Warranty),
	}
}

func (r *mockWarrantyRepo) addWarranty(w *domain.Warranty) {
	r.warranties[w.AssetID] = w
}

func (r *mockWarrantyRepo) GetByAssetID(_ context.Context, assetID uuid.UUID) (*domain.Warranty, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.warranties[assetID], nil
}

func (r *mockWarrantyRepo) List(_ context.Context, _ uuid.UUID) ([]domain.WarrantyWithAsset, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	result := make([]domain.WarrantyWithAsset, 0, len(r.warranties))
	for _, w := range r.warranties {
		result = append(result, domain.WarrantyWithAsset{
			Warranty:  *w,
			AssetName: "Test Asset",
		})
	}
	return result, nil
}

func (r *mockWarrantyRepo) ListExpiring(_ context.Context, _ uuid.UUID, days int) ([]domain.Warranty, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, days)
	result := make([]domain.Warranty, 0)
	for _, w := range r.warranties {
		if w.EndDate != nil && w.EndDate.Before(cutoff) && w.EndDate.After(now) {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (r *mockWarrantyRepo) Create(_ context.Context, w *domain.Warranty) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	w.CreatedAt = time.Now().UTC()
	w.UpdatedAt = time.Now().UTC()
	r.warranties[w.AssetID] = w
	return nil
}

func (r *mockWarrantyRepo) Update(_ context.Context, w *domain.Warranty) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	w.UpdatedAt = time.Now().UTC()
	r.warranties[w.AssetID] = w
	return nil
}

func (r *mockWarrantyRepo) Delete(_ context.Context, assetID uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.warranties, assetID)
	return nil
}

// testWarrantyHandler wraps warranty handler logic for testing
type testWarrantyHandler struct {
	warrantyRepo *mockWarrantyRepo
	assetRepo    *mockAssetRepo
	orgID        uuid.UUID
}

func newTestWarrantyHandler() *testWarrantyHandler {
	return &testWarrantyHandler{
		warrantyRepo: newMockWarrantyRepo(),
		assetRepo:    newMockAssetRepo(),
		orgID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testWarrantyHandler) getWarranty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	warranty, err := h.warrantyRepo.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get warranty")
		return
	}
	if warranty == nil {
		writeError(w, http.StatusNotFound, "warranty not found")
		return
	}

	writeJSON(w, http.StatusOK, warranty)
}

func (h *testWarrantyHandler) listWarranties(w http.ResponseWriter, r *http.Request) {
	warranties, err := h.warrantyRepo.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warranties")
		return
	}

	if warranties == nil {
		warranties = []domain.WarrantyWithAsset{}
	}

	writeJSON(w, http.StatusOK, warranties)
}

func (h *testWarrantyHandler) listExpiringWarranties(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := parseInt(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	warranties, err := h.warrantyRepo.ListExpiring(r.Context(), h.orgID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list warranties")
		return
	}

	if warranties == nil {
		warranties = []domain.Warranty{}
	}

	writeJSON(w, http.StatusOK, warranties)
}

func (h *testWarrantyHandler) createWarranty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	// Check if asset exists
	asset, err := h.assetRepo.GetByID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check asset")
		return
	}
	if asset == nil {
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}

	// Check if warranty already exists
	existing, err := h.warrantyRepo.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing warranty")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "warranty already exists for this asset")
		return
	}

	var req CreateWarrantyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warranty := &domain.Warranty{
		AssetID:  assetID,
		Provider: req.Provider,
		Notes:    req.Notes,
	}

	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			warranty.StartDate = &t
		}
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			warranty.EndDate = &t
		}
	}

	if err := h.warrantyRepo.Create(r.Context(), warranty); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create warranty")
		return
	}

	writeJSON(w, http.StatusCreated, warranty)
}

func (h *testWarrantyHandler) updateWarranty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	warranty, err := h.warrantyRepo.GetByAssetID(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get warranty")
		return
	}
	if warranty == nil {
		writeError(w, http.StatusNotFound, "warranty not found")
		return
	}

	var req UpdateWarrantyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	warranty.Provider = req.Provider
	warranty.Notes = req.Notes

	if req.StartDate != nil {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			warranty.StartDate = &t
		}
	} else {
		warranty.StartDate = nil
	}
	if req.EndDate != nil {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			warranty.EndDate = &t
		}
	} else {
		warranty.EndDate = nil
	}

	if err := h.warrantyRepo.Update(r.Context(), warranty); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update warranty")
		return
	}

	writeJSON(w, http.StatusOK, warranty)
}

func (h *testWarrantyHandler) deleteWarranty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}

	if err := h.warrantyRepo.Delete(r.Context(), assetID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete warranty")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createTestWarranty(assetID uuid.UUID, provider string, endDate *time.Time) *domain.Warranty {
	return &domain.Warranty{
		ID:        uuid.New(),
		AssetID:   assetID,
		Provider:  &provider,
		EndDate:   endDate,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// Tests

func Test_ListWarranties_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestWarrantyHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/warranties", nil)
	rec := httptest.NewRecorder()
	h.listWarranties(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.WarrantyWithAsset
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func Test_ListWarranties_WithWarranties_ReturnsAll(t *testing.T) {
	h := newTestWarrantyHandler()
	asset1 := createTestAsset("Asset 1", uuid.New(), nil)
	asset2 := createTestAsset("Asset 2", uuid.New(), nil)
	h.assetRepo.addAsset(asset1)
	h.assetRepo.addAsset(asset2)

	endDate := time.Now().AddDate(1, 0, 0)
	h.warrantyRepo.addWarranty(createTestWarranty(asset1.ID, "Provider A", &endDate))
	h.warrantyRepo.addWarranty(createTestWarranty(asset2.ID, "Provider B", &endDate))

	req := httptest.NewRequest(http.MethodGet, "/api/warranties", nil)
	rec := httptest.NewRecorder()
	h.listWarranties(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.WarrantyWithAsset
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 warranties, got %d", len(resp))
	}
}

func Test_ListExpiringWarranties_DefaultDays_Returns30Days(t *testing.T) {
	h := newTestWarrantyHandler()
	asset1 := createTestAsset("Asset 1", uuid.New(), nil)
	asset2 := createTestAsset("Asset 2", uuid.New(), nil)
	h.assetRepo.addAsset(asset1)
	h.assetRepo.addAsset(asset2)

	// Warranty expiring in 15 days (should be included)
	expiring := time.Now().AddDate(0, 0, 15)
	h.warrantyRepo.addWarranty(createTestWarranty(asset1.ID, "Provider A", &expiring))

	// Warranty expiring in 60 days (should not be included)
	notExpiring := time.Now().AddDate(0, 0, 60)
	h.warrantyRepo.addWarranty(createTestWarranty(asset2.ID, "Provider B", &notExpiring))

	req := httptest.NewRequest(http.MethodGet, "/api/warranties/expiring", nil)
	rec := httptest.NewRecorder()
	h.listExpiringWarranties(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 expiring warranty, got %d", len(resp))
	}
}

func Test_ListExpiringWarranties_CustomDays(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	// Warranty expiring in 45 days
	expiring := time.Now().AddDate(0, 0, 45)
	h.warrantyRepo.addWarranty(createTestWarranty(asset.ID, "Provider", &expiring))

	req := httptest.NewRequest(http.MethodGet, "/api/warranties/expiring?days=60", nil)
	rec := httptest.NewRecorder()
	h.listExpiringWarranties(rec, req)

	var resp []domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 expiring warranty with days=60, got %d", len(resp))
	}
}

func Test_GetWarranty_ExistingWarranty_ReturnsWarranty(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	endDate := time.Now().AddDate(1, 0, 0)
	warranty := createTestWarranty(asset.ID, "Test Provider", &endDate)
	h.warrantyRepo.addWarranty(warranty)

	req := httptest.NewRequest(http.MethodGet, "/api/assets/"+asset.ID.String()+"/warranty", nil)
	req = withChiURLParam(req, "id", asset.ID.String())
	rec := httptest.NewRecorder()
	h.getWarranty(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if *resp.Provider != "Test Provider" {
		t.Errorf("expected provider 'Test Provider', got '%s'", *resp.Provider)
	}
}

func Test_GetWarranty_NonExistentWarranty_ReturnsNotFound(t *testing.T) {
	h := newTestWarrantyHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/assets/"+nonExistentID.String()+"/warranty", nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getWarranty(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetWarranty_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestWarrantyHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/assets/not-a-uuid/warranty", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getWarranty(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateWarranty_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	body := strings.NewReader(`{
		"provider": "Manufacturer Warranty",
		"start_date": "2024-01-01",
		"end_date": "2026-01-01",
		"notes": "Extended warranty"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets/"+asset.ID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createWarranty(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if *resp.Provider != "Manufacturer Warranty" {
		t.Errorf("expected provider 'Manufacturer Warranty', got '%s'", *resp.Provider)
	}
}

func Test_CreateWarranty_AssetNotFound_ReturnsNotFound(t *testing.T) {
	h := newTestWarrantyHandler()

	nonExistentID := uuid.New()
	body := strings.NewReader(`{
		"provider": "Test"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets/"+nonExistentID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createWarranty(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_CreateWarranty_AlreadyExists_ReturnsConflict(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	// Add existing warranty
	endDate := time.Now().AddDate(1, 0, 0)
	h.warrantyRepo.addWarranty(createTestWarranty(asset.ID, "Existing", &endDate))

	body := strings.NewReader(`{
		"provider": "New Provider"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/assets/"+asset.ID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createWarranty(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}
}

func Test_CreateWarranty_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestWarrantyHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/assets/not-a-uuid/warranty", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.createWarranty(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateWarranty_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/assets/"+asset.ID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createWarranty(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateWarranty_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	endDate := time.Now().AddDate(1, 0, 0)
	warranty := createTestWarranty(asset.ID, "Old Provider", &endDate)
	h.warrantyRepo.addWarranty(warranty)

	body := strings.NewReader(`{
		"provider": "New Provider",
		"end_date": "2027-01-01"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/assets/"+asset.ID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateWarranty(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if *resp.Provider != "New Provider" {
		t.Errorf("expected provider 'New Provider', got '%s'", *resp.Provider)
	}
}

func Test_UpdateWarranty_NonExistentWarranty_ReturnsNotFound(t *testing.T) {
	h := newTestWarrantyHandler()

	nonExistentID := uuid.New()
	body := strings.NewReader(`{
		"provider": "Updated"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/assets/"+nonExistentID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateWarranty(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateWarranty_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestWarrantyHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/assets/not-a-uuid/warranty", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateWarranty(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateWarranty_ClearDates_SetsToNil(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	startDate := time.Now()
	endDate := time.Now().AddDate(1, 0, 0)
	warranty := &domain.Warranty{
		ID:        uuid.New(),
		AssetID:   asset.ID,
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	h.warrantyRepo.addWarranty(warranty)

	body := strings.NewReader(`{
		"provider": "Provider Only"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/assets/"+asset.ID.String()+"/warranty", body)
	req = withChiURLParam(req, "id", asset.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateWarranty(rec, req)

	var resp domain.Warranty
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.StartDate != nil {
		t.Error("expected start_date to be nil after update")
	}
	if resp.EndDate != nil {
		t.Error("expected end_date to be nil after update")
	}
}

func Test_DeleteWarranty_ExistingWarranty_ReturnsNoContent(t *testing.T) {
	h := newTestWarrantyHandler()
	asset := createTestAsset("Asset", uuid.New(), nil)
	h.assetRepo.addAsset(asset)

	endDate := time.Now().AddDate(1, 0, 0)
	h.warrantyRepo.addWarranty(createTestWarranty(asset.ID, "Provider", &endDate))

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/"+asset.ID.String()+"/warranty", nil)
	req = withChiURLParam(req, "id", asset.ID.String())
	rec := httptest.NewRecorder()
	h.deleteWarranty(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	if _, exists := h.warrantyRepo.warranties[asset.ID]; exists {
		t.Error("expected warranty to be deleted from repository")
	}
}

func Test_DeleteWarranty_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestWarrantyHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/not-a-uuid/warranty", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteWarranty(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
