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

// mockLocationRepo implements a minimal location repository for testing
type mockLocationRepo struct {
	locations   map[uuid.UUID]*domain.Location
	ListError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
}

func newMockLocationRepo() *mockLocationRepo {
	return &mockLocationRepo{
		locations: make(map[uuid.UUID]*domain.Location),
	}
}

func (r *mockLocationRepo) addLocation(l *domain.Location) {
	r.locations[l.ID] = l
}

func (r *mockLocationRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Location, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.locations[id], nil
}

func (r *mockLocationRepo) List(_ context.Context, _ uuid.UUID) ([]domain.Location, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	locs := make([]domain.Location, 0, len(r.locations))
	for _, l := range r.locations {
		locs = append(locs, *l)
	}
	return locs, nil
}

func (r *mockLocationRepo) ListTree(_ context.Context, _ uuid.UUID) ([]domain.Location, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	roots := make([]domain.Location, 0)
	for _, l := range r.locations {
		if l.ParentID == nil {
			loc := *l
			for _, child := range r.locations {
				if child.ParentID != nil && *child.ParentID == l.ID {
					loc.Children = append(loc.Children, *child)
				}
			}
			roots = append(roots, loc)
		}
	}
	return roots, nil
}

func (r *mockLocationRepo) Create(_ context.Context, l *domain.Location) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	l.CreatedAt = time.Now().UTC()
	l.UpdatedAt = time.Now().UTC()
	r.locations[l.ID] = l
	return nil
}

func (r *mockLocationRepo) Update(_ context.Context, l *domain.Location) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	l.UpdatedAt = time.Now().UTC()
	r.locations[l.ID] = l
	return nil
}

func (r *mockLocationRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.locations, id)
	return nil
}

// testLocationHandler wraps location handler logic for testing
type testLocationHandler struct {
	locationRepo *mockLocationRepo
	orgID        uuid.UUID
}

func newTestLocationHandler() *testLocationHandler {
	return &testLocationHandler{
		locationRepo: newMockLocationRepo(),
		orgID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testLocationHandler) listLocations(w http.ResponseWriter, r *http.Request) {
	tree := r.URL.Query().Get("tree") == "true"

	var locations []domain.Location
	var err error

	if tree {
		locations, err = h.locationRepo.ListTree(r.Context(), h.orgID)
	} else {
		locations, err = h.locationRepo.List(r.Context(), h.orgID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list locations")
		return
	}

	if locations == nil {
		locations = []domain.Location{}
	}

	writeJSON(w, http.StatusOK, locations)
}

func (h *testLocationHandler) getLocation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	loc, err := h.locationRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	writeJSON(w, http.StatusOK, loc)
}

func (h *testLocationHandler) createLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	loc := &domain.Location{
		OrganizationID: h.orgID,
		Name:           req.Name,
		Description:    req.Description,
	}

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		loc.ParentID = &parentID
	}

	if err := h.locationRepo.Create(r.Context(), loc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create location")
		return
	}

	writeJSON(w, http.StatusCreated, loc)
}

func (h *testLocationHandler) updateLocation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	loc, err := h.locationRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	loc.Name = req.Name
	loc.Description = req.Description

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		loc.ParentID = &parentID
	} else {
		loc.ParentID = nil
	}

	if err := h.locationRepo.Update(r.Context(), loc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update location")
		return
	}

	writeJSON(w, http.StatusOK, loc)
}

func (h *testLocationHandler) deleteLocation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location ID")
		return
	}

	if err := h.locationRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete location")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createTestLocation(name string, parentID *uuid.UUID) *domain.Location {
	return &domain.Location{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ParentID:       parentID,
		Name:           name,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_ListLocations_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestLocationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
	rec := httptest.NewRecorder()
	h.listLocations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func Test_ListLocations_WithLocations_ReturnsAll(t *testing.T) {
	h := newTestLocationHandler()
	h.locationRepo.addLocation(createTestLocation("Office", nil))
	h.locationRepo.addLocation(createTestLocation("Warehouse", nil))
	h.locationRepo.addLocation(createTestLocation("Storage Room", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
	rec := httptest.NewRecorder()
	h.listLocations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Errorf("expected 3 locations, got %d", len(resp))
	}
}

func Test_ListLocations_TreeMode_ReturnsHierarchy(t *testing.T) {
	h := newTestLocationHandler()
	office := createTestLocation("Office", nil)
	h.locationRepo.addLocation(office)

	desk1 := createTestLocation("Desk 1", &office.ID)
	h.locationRepo.addLocation(desk1)

	desk2 := createTestLocation("Desk 2", &office.ID)
	h.locationRepo.addLocation(desk2)

	warehouse := createTestLocation("Warehouse", nil)
	h.locationRepo.addLocation(warehouse)

	req := httptest.NewRequest(http.MethodGet, "/api/locations?tree=true", nil)
	rec := httptest.NewRecorder()
	h.listLocations(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 root locations, got %d", len(resp))
	}
}

func Test_GetLocation_ExistingLocation_ReturnsLocation(t *testing.T) {
	h := newTestLocationHandler()
	loc := createTestLocation("Office", nil)
	h.locationRepo.addLocation(loc)

	req := httptest.NewRequest(http.MethodGet, "/api/locations/"+loc.ID.String(), nil)
	req = withChiURLParam(req, "id", loc.ID.String())
	rec := httptest.NewRecorder()
	h.getLocation(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Office" {
		t.Errorf("expected name 'Office', got '%s'", resp.Name)
	}
}

func Test_GetLocation_NonExistentLocation_ReturnsNotFound(t *testing.T) {
	h := newTestLocationHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/locations/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getLocation(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetLocation_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/locations/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateLocation_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestLocationHandler()

	body := strings.NewReader(`{
		"name": "New Office",
		"description": "Main office building"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/locations", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createLocation(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "New Office" {
		t.Errorf("expected name 'New Office', got '%s'", resp.Name)
	}
}

func Test_CreateLocation_WithParent_SetsParentID(t *testing.T) {
	h := newTestLocationHandler()
	parent := createTestLocation("Building A", nil)
	h.locationRepo.addLocation(parent)

	body := strings.NewReader(`{
		"name": "Room 101",
		"parent_id": "` + parent.ID.String() + `"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/locations", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createLocation(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var resp domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.ParentID == nil || *resp.ParentID != parent.ID {
		t.Error("expected parent_id to be set")
	}
}

func Test_CreateLocation_MissingName_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	body := strings.NewReader(`{
		"description": "Location without name"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/locations", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateLocation_InvalidParentID_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	body := strings.NewReader(`{
		"name": "Location",
		"parent_id": "not-a-uuid"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/locations", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateLocation_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/locations", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateLocation_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestLocationHandler()
	loc := createTestLocation("Old Name", nil)
	h.locationRepo.addLocation(loc)

	body := strings.NewReader(`{
		"name": "New Name",
		"description": "Updated description"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/locations/"+loc.ID.String(), body)
	req = withChiURLParam(req, "id", loc.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateLocation(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", resp.Name)
	}
}

func Test_UpdateLocation_NonExistentLocation_ReturnsNotFound(t *testing.T) {
	h := newTestLocationHandler()
	nonExistentID := uuid.New()

	body := strings.NewReader(`{
		"name": "Updated"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/locations/"+nonExistentID.String(), body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateLocation(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateLocation_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/locations/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateLocation_ClearParent_SetsParentToNil(t *testing.T) {
	h := newTestLocationHandler()
	parent := createTestLocation("Parent", nil)
	h.locationRepo.addLocation(parent)

	child := createTestLocation("Child", &parent.ID)
	h.locationRepo.addLocation(child)

	body := strings.NewReader(`{
		"name": "Former Child"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/locations/"+child.ID.String(), body)
	req = withChiURLParam(req, "id", child.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateLocation(rec, req)

	var resp domain.Location
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.ParentID != nil {
		t.Error("expected parent_id to be nil after update")
	}
}

func Test_DeleteLocation_ExistingLocation_ReturnsNoContent(t *testing.T) {
	h := newTestLocationHandler()
	loc := createTestLocation("To Delete", nil)
	h.locationRepo.addLocation(loc)

	req := httptest.NewRequest(http.MethodDelete, "/api/locations/"+loc.ID.String(), nil)
	req = withChiURLParam(req, "id", loc.ID.String())
	rec := httptest.NewRecorder()
	h.deleteLocation(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	if _, exists := h.locationRepo.locations[loc.ID]; exists {
		t.Error("expected location to be deleted from repository")
	}
}

func Test_DeleteLocation_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestLocationHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/locations/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteLocation(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
