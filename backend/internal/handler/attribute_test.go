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

// mockAttributeRepo implements a minimal attribute repository for testing
type mockAttributeRepo struct {
	attributes  map[uuid.UUID]*domain.Attribute
	ListError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
}

func newMockAttributeRepo() *mockAttributeRepo {
	return &mockAttributeRepo{
		attributes: make(map[uuid.UUID]*domain.Attribute),
	}
}

func (r *mockAttributeRepo) addAttribute(a *domain.Attribute) {
	r.attributes[a.ID] = a
}

func (r *mockAttributeRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Attribute, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.attributes[id], nil
}

func (r *mockAttributeRepo) List(_ context.Context, _ uuid.UUID) ([]domain.Attribute, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	attrs := make([]domain.Attribute, 0, len(r.attributes))
	for _, a := range r.attributes {
		attrs = append(attrs, *a)
	}
	return attrs, nil
}

func (r *mockAttributeRepo) Create(_ context.Context, a *domain.Attribute) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = time.Now().UTC()
	r.attributes[a.ID] = a
	return nil
}

func (r *mockAttributeRepo) Update(_ context.Context, a *domain.Attribute) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	a.UpdatedAt = time.Now().UTC()
	r.attributes[a.ID] = a
	return nil
}

func (r *mockAttributeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.attributes, id)
	return nil
}

// testAttributeHandler wraps attribute handler logic for testing
type testAttributeHandler struct {
	attributeRepo *mockAttributeRepo
	orgID         uuid.UUID
}

func newTestAttributeHandler() *testAttributeHandler {
	return &testAttributeHandler{
		attributeRepo: newMockAttributeRepo(),
		orgID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testAttributeHandler) listAttributes(w http.ResponseWriter, r *http.Request) {
	attributes, err := h.attributeRepo.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attributes")
		return
	}
	writeJSON(w, http.StatusOK, attributes)
}

func (h *testAttributeHandler) getAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	attr, err := h.attributeRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if attr == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	writeJSON(w, http.StatusOK, attr)
}

func (h *testAttributeHandler) createAttribute(w http.ResponseWriter, r *http.Request) {
	var req CreateAttributeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.DataType == "" {
		writeError(w, http.StatusBadRequest, "data_type is required")
		return
	}

	switch req.DataType {
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeText, domain.AttributeTypeDate:
		// Valid
	default:
		writeError(w, http.StatusBadRequest, "invalid data_type: must be one of string, number, boolean, text, date")
		return
	}

	attr := &domain.Attribute{
		OrganizationID: h.orgID,
		Name:           req.Name,
		Key:            req.Key,
		DataType:       req.DataType,
	}

	if err := h.attributeRepo.Create(r.Context(), attr); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create attribute")
		return
	}

	writeJSON(w, http.StatusCreated, attr)
}

func (h *testAttributeHandler) updateAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	existing, err := h.attributeRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	var req UpdateAttributeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DataType == "" {
		writeError(w, http.StatusBadRequest, "data_type is required")
		return
	}

	switch req.DataType {
	case domain.AttributeTypeString, domain.AttributeTypeNumber, domain.AttributeTypeBoolean, domain.AttributeTypeText, domain.AttributeTypeDate:
		// Valid
	default:
		writeError(w, http.StatusBadRequest, "invalid data_type: must be one of string, number, boolean, text, date")
		return
	}

	existing.Name = req.Name
	existing.DataType = req.DataType

	if err := h.attributeRepo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update attribute")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (h *testAttributeHandler) deleteAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attribute ID")
		return
	}

	existing, err := h.attributeRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attribute")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "attribute not found")
		return
	}

	if existing.PluginID != nil {
		writeError(w, http.StatusForbidden, "cannot delete plugin-owned attribute")
		return
	}

	if err := h.attributeRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete attribute")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createTestAttribute(name, key string, dataType domain.AttributeDataType) *domain.Attribute {
	return &domain.Attribute{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:           name,
		Key:            key,
		DataType:       dataType,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_ListAttributes_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestAttributeHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/attributes", nil)
	rec := httptest.NewRecorder()
	h.listAttributes(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Attribute
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func Test_ListAttributes_WithAttributes_ReturnsAll(t *testing.T) {
	h := newTestAttributeHandler()
	h.attributeRepo.addAttribute(createTestAttribute("Color", "color", domain.AttributeTypeString))
	h.attributeRepo.addAttribute(createTestAttribute("Weight", "weight", domain.AttributeTypeNumber))
	h.attributeRepo.addAttribute(createTestAttribute("Is Active", "is_active", domain.AttributeTypeBoolean))

	req := httptest.NewRequest(http.MethodGet, "/api/attributes", nil)
	rec := httptest.NewRecorder()
	h.listAttributes(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Attribute
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(resp))
	}
}

func Test_GetAttribute_ExistingAttribute_ReturnsAttribute(t *testing.T) {
	h := newTestAttributeHandler()
	attr := createTestAttribute("Color", "color", domain.AttributeTypeString)
	h.attributeRepo.addAttribute(attr)

	req := httptest.NewRequest(http.MethodGet, "/api/attributes/"+attr.ID.String(), nil)
	req = withChiURLParam(req, "id", attr.ID.String())
	rec := httptest.NewRecorder()
	h.getAttribute(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Attribute
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Color" {
		t.Errorf("expected name 'Color', got '%s'", resp.Name)
	}
	if resp.Key != "color" {
		t.Errorf("expected key 'color', got '%s'", resp.Key)
	}
}

func Test_GetAttribute_NonExistentAttribute_ReturnsNotFound(t *testing.T) {
	h := newTestAttributeHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/attributes/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getAttribute(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetAttribute_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/attributes/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAttribute_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`{
		"name": "Size",
		"key": "size",
		"data_type": "string"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Attribute
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Size" {
		t.Errorf("expected name 'Size', got '%s'", resp.Name)
	}
	if resp.Key != "size" {
		t.Errorf("expected key 'size', got '%s'", resp.Key)
	}
	if resp.DataType != domain.AttributeTypeString {
		t.Errorf("expected data_type 'string', got '%s'", resp.DataType)
	}
}

func Test_CreateAttribute_AllDataTypes_Succeed(t *testing.T) {
	dataTypes := []domain.AttributeDataType{
		domain.AttributeTypeString,
		domain.AttributeTypeNumber,
		domain.AttributeTypeBoolean,
		domain.AttributeTypeText,
		domain.AttributeTypeDate,
	}

	for _, dt := range dataTypes {
		t.Run(string(dt), func(t *testing.T) {
			h := newTestAttributeHandler()

			body := strings.NewReader(`{
				"name": "Test",
				"key": "test",
				"data_type": "` + string(dt) + `"
			}`)

			req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.createAttribute(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status 201 for data_type '%s', got %d", dt, rec.Code)
			}
		})
	}
}

func Test_CreateAttribute_MissingName_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`{
		"key": "test",
		"data_type": "string"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAttribute_MissingKey_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`{
		"name": "Test",
		"data_type": "string"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAttribute_MissingDataType_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`{
		"name": "Test",
		"key": "test"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAttribute_InvalidDataType_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`{
		"name": "Test",
		"key": "test",
		"data_type": "invalid_type"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateAttribute_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/attributes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateAttribute_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestAttributeHandler()
	attr := createTestAttribute("Old Name", "old_key", domain.AttributeTypeString)
	h.attributeRepo.addAttribute(attr)

	body := strings.NewReader(`{
		"name": "New Name",
		"data_type": "number"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/attributes/"+attr.ID.String(), body)
	req = withChiURLParam(req, "id", attr.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateAttribute(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Attribute
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", resp.Name)
	}
	if resp.DataType != domain.AttributeTypeNumber {
		t.Errorf("expected data_type 'number', got '%s'", resp.DataType)
	}
	// Key should remain unchanged
	if resp.Key != "old_key" {
		t.Errorf("expected key to remain 'old_key', got '%s'", resp.Key)
	}
}

func Test_UpdateAttribute_NonExistentAttribute_ReturnsNotFound(t *testing.T) {
	h := newTestAttributeHandler()
	nonExistentID := uuid.New()

	body := strings.NewReader(`{
		"name": "Updated",
		"data_type": "string"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/attributes/"+nonExistentID.String(), body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateAttribute(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateAttribute_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/attributes/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateAttribute_InvalidDataType_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()
	attr := createTestAttribute("Test", "test", domain.AttributeTypeString)
	h.attributeRepo.addAttribute(attr)

	body := strings.NewReader(`{
		"name": "Test",
		"data_type": "invalid_type"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/attributes/"+attr.ID.String(), body)
	req = withChiURLParam(req, "id", attr.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteAttribute_ExistingAttribute_ReturnsNoContent(t *testing.T) {
	h := newTestAttributeHandler()
	attr := createTestAttribute("To Delete", "to_delete", domain.AttributeTypeString)
	h.attributeRepo.addAttribute(attr)

	req := httptest.NewRequest(http.MethodDelete, "/api/attributes/"+attr.ID.String(), nil)
	req = withChiURLParam(req, "id", attr.ID.String())
	rec := httptest.NewRecorder()
	h.deleteAttribute(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	if _, exists := h.attributeRepo.attributes[attr.ID]; exists {
		t.Error("expected attribute to be deleted from repository")
	}
}

func Test_DeleteAttribute_NonExistentAttribute_ReturnsNotFound(t *testing.T) {
	h := newTestAttributeHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/attributes/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.deleteAttribute(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_DeleteAttribute_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestAttributeHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/attributes/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteAttribute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteAttribute_PluginOwnedAttribute_ReturnsForbidden(t *testing.T) {
	h := newTestAttributeHandler()
	pluginID := "google_books"
	attr := &domain.Attribute{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PluginID:       &pluginID,
		Name:           "ISBN",
		Key:            "books.isbn",
		DataType:       domain.AttributeTypeString,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	h.attributeRepo.addAttribute(attr)

	req := httptest.NewRequest(http.MethodDelete, "/api/attributes/"+attr.ID.String(), nil)
	req = withChiURLParam(req, "id", attr.ID.String())
	rec := httptest.NewRecorder()
	h.deleteAttribute(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for plugin-owned attribute, got %d", rec.Code)
	}
}
