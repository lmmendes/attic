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

// mockConditionRepo implements a minimal condition repository for testing
type mockConditionRepo struct {
	conditions  map[uuid.UUID]*domain.Condition
	ListError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
}

func newMockConditionRepo() *mockConditionRepo {
	return &mockConditionRepo{
		conditions: make(map[uuid.UUID]*domain.Condition),
	}
}

func (r *mockConditionRepo) addCondition(c *domain.Condition) {
	r.conditions[c.ID] = c
}

func (r *mockConditionRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Condition, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}
	return r.conditions[id], nil
}

func (r *mockConditionRepo) List(_ context.Context, _ uuid.UUID) ([]domain.Condition, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	conds := make([]domain.Condition, 0, len(r.conditions))
	for _, c := range r.conditions {
		conds = append(conds, *c)
	}
	return conds, nil
}

func (r *mockConditionRepo) Create(_ context.Context, c *domain.Condition) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = time.Now().UTC()
	r.conditions[c.ID] = c
	return nil
}

func (r *mockConditionRepo) Update(_ context.Context, c *domain.Condition) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	c.UpdatedAt = time.Now().UTC()
	r.conditions[c.ID] = c
	return nil
}

func (r *mockConditionRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	delete(r.conditions, id)
	return nil
}

// testConditionHandler wraps condition handler logic for testing
type testConditionHandler struct {
	conditionRepo *mockConditionRepo
	orgID         uuid.UUID
}

func newTestConditionHandler() *testConditionHandler {
	return &testConditionHandler{
		conditionRepo: newMockConditionRepo(),
		orgID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

func (h *testConditionHandler) listConditions(w http.ResponseWriter, r *http.Request) {
	conditions, err := h.conditionRepo.List(r.Context(), h.orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conditions")
		return
	}

	if conditions == nil {
		conditions = []domain.Condition{}
	}

	writeJSON(w, http.StatusOK, conditions)
}

func (h *testConditionHandler) getCondition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	cond, err := h.conditionRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get condition")
		return
	}
	if cond == nil {
		writeError(w, http.StatusNotFound, "condition not found")
		return
	}

	writeJSON(w, http.StatusOK, cond)
}

func (h *testConditionHandler) createCondition(w http.ResponseWriter, r *http.Request) {
	var req CreateConditionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" || req.Label == "" {
		writeError(w, http.StatusBadRequest, "code and label are required")
		return
	}

	cond := &domain.Condition{
		OrganizationID: h.orgID,
		Code:           req.Code,
		Label:          req.Label,
		Description:    req.Description,
		SortOrder:      req.SortOrder,
	}

	if err := h.conditionRepo.Create(r.Context(), cond); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create condition")
		return
	}

	writeJSON(w, http.StatusCreated, cond)
}

func (h *testConditionHandler) updateCondition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	var req UpdateConditionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cond, err := h.conditionRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get condition")
		return
	}
	if cond == nil {
		writeError(w, http.StatusNotFound, "condition not found")
		return
	}

	cond.Code = req.Code
	cond.Label = req.Label
	cond.Description = req.Description
	cond.SortOrder = req.SortOrder

	if err := h.conditionRepo.Update(r.Context(), cond); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update condition")
		return
	}

	writeJSON(w, http.StatusOK, cond)
}

func (h *testConditionHandler) deleteCondition(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid condition ID")
		return
	}

	if err := h.conditionRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete condition")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createTestCondition(code, label string, sortOrder int) *domain.Condition {
	return &domain.Condition{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Code:           code,
		Label:          label,
		SortOrder:      sortOrder,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_ListConditions_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestConditionHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/conditions", nil)
	rec := httptest.NewRecorder()
	h.listConditions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Condition
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func Test_ListConditions_WithConditions_ReturnsAll(t *testing.T) {
	h := newTestConditionHandler()
	h.conditionRepo.addCondition(createTestCondition("NEW", "New", 1))
	h.conditionRepo.addCondition(createTestCondition("GOOD", "Good", 2))
	h.conditionRepo.addCondition(createTestCondition("FAIR", "Fair", 3))

	req := httptest.NewRequest(http.MethodGet, "/api/conditions", nil)
	rec := httptest.NewRecorder()
	h.listConditions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp []domain.Condition
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Errorf("expected 3 conditions, got %d", len(resp))
	}
}

func Test_GetCondition_ExistingCondition_ReturnsCondition(t *testing.T) {
	h := newTestConditionHandler()
	cond := createTestCondition("NEW", "New", 1)
	h.conditionRepo.addCondition(cond)

	req := httptest.NewRequest(http.MethodGet, "/api/conditions/"+cond.ID.String(), nil)
	req = withChiURLParam(req, "id", cond.ID.String())
	rec := httptest.NewRecorder()
	h.getCondition(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Condition
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "NEW" {
		t.Errorf("expected code 'NEW', got '%s'", resp.Code)
	}
}

func Test_GetCondition_NonExistentCondition_ReturnsNotFound(t *testing.T) {
	h := newTestConditionHandler()

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/conditions/"+nonExistentID.String(), nil)
	req = withChiURLParam(req, "id", nonExistentID.String())
	rec := httptest.NewRecorder()
	h.getCondition(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetCondition_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/conditions/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.getCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCondition_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestConditionHandler()

	body := strings.NewReader(`{
		"code": "EXCELLENT",
		"label": "Excellent",
		"description": "Like new condition",
		"sort_order": 0
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/conditions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCondition(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp domain.Condition
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "EXCELLENT" {
		t.Errorf("expected code 'EXCELLENT', got '%s'", resp.Code)
	}
	if resp.Label != "Excellent" {
		t.Errorf("expected label 'Excellent', got '%s'", resp.Label)
	}
}

func Test_CreateCondition_MissingCode_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	body := strings.NewReader(`{
		"label": "Excellent"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/conditions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCondition_MissingLabel_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	body := strings.NewReader(`{
		"code": "EXCELLENT"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/conditions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateCondition_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/conditions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.createCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateCondition_ValidRequest_ReturnsUpdated(t *testing.T) {
	h := newTestConditionHandler()
	cond := createTestCondition("OLD", "Old Label", 1)
	h.conditionRepo.addCondition(cond)

	body := strings.NewReader(`{
		"code": "UPDATED",
		"label": "Updated Label",
		"sort_order": 5
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/conditions/"+cond.ID.String(), body)
	req = withChiURLParam(req, "id", cond.ID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateCondition(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp domain.Condition
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != "UPDATED" {
		t.Errorf("expected code 'UPDATED', got '%s'", resp.Code)
	}
	if resp.SortOrder != 5 {
		t.Errorf("expected sort_order 5, got %d", resp.SortOrder)
	}
}

func Test_UpdateCondition_NonExistentCondition_ReturnsNotFound(t *testing.T) {
	h := newTestConditionHandler()
	nonExistentID := uuid.New()

	body := strings.NewReader(`{
		"code": "UPDATED",
		"label": "Updated"
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/conditions/"+nonExistentID.String(), body)
	req = withChiURLParam(req, "id", nonExistentID.String())
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.updateCondition(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateCondition_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	req := httptest.NewRequest(http.MethodPut, "/api/conditions/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.updateCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteCondition_ExistingCondition_ReturnsNoContent(t *testing.T) {
	h := newTestConditionHandler()
	cond := createTestCondition("TO_DELETE", "To Delete", 1)
	h.conditionRepo.addCondition(cond)

	req := httptest.NewRequest(http.MethodDelete, "/api/conditions/"+cond.ID.String(), nil)
	req = withChiURLParam(req, "id", cond.ID.String())
	rec := httptest.NewRecorder()
	h.deleteCondition(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	if _, exists := h.conditionRepo.conditions[cond.ID]; exists {
		t.Error("expected condition to be deleted from repository")
	}
}

func Test_DeleteCondition_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestConditionHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/conditions/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.deleteCondition(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
