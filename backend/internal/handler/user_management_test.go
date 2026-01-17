package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
)

// mockSessionManager for testing user management
type mockSessionManager struct {
	session *mockSessionData
}

type mockSessionData struct {
	UserID uuid.UUID
	Email  string
	Name   string
	Role   domain.UserRole
}

func (m *mockSessionManager) GetSession(r *http.Request) (*mockSessionData, error) {
	if m.session == nil {
		return nil, http.ErrNoCookie
	}
	return m.session, nil
}

// testUserManagementHandler wraps user management logic for testing
type testUserManagementHandler struct {
	userRepo       *mockUserRepo
	sessionMgr     *mockSessionManager
	passwordMinLen int
	defaultOrgID   uuid.UUID
}

func newTestUserManagementHandler() *testUserManagementHandler {
	orgID := uuid.New()
	repo := newMockUserRepo()
	repo.defaultOrgID = orgID
	return &testUserManagementHandler{
		userRepo:       repo,
		sessionMgr:     &mockSessionManager{},
		passwordMinLen: 8,
		defaultOrgID:   orgID,
	}
}

func (h *testUserManagementHandler) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := h.sessionMgr.GetSession(r)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if session.Role != domain.UserRoleAdmin {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *testUserManagementHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.List(r.Context(), h.defaultOrgID)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = toUserResponse(&u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *testUserManagementHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *testUserManagementHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error":"email is required"}`, http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, `{"error":"password is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < h.passwordMinLen {
		http.Error(w, `{"error":"password too short"}`, http.StatusBadRequest)
		return
	}

	existing, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
		return
	}

	hash := "hashed:" + req.Password
	role := domain.UserRoleUser
	if req.Role == "admin" {
		role = domain.UserRoleAdmin
	}

	user := &domain.User{
		OrganizationID: h.defaultOrgID,
		Email:          req.Email,
		PasswordHash:   &hash,
		Role:           role,
	}
	if req.Name != "" {
		user.DisplayName = &req.Name
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *testUserManagementHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email != "" && req.Email != user.Email {
		existing, err := h.userRepo.GetByEmail(r.Context(), req.Email)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		if existing != nil && existing.ID != user.ID {
			http.Error(w, `{"error":"email already in use"}`, http.StatusConflict)
			return
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		user.DisplayName = &req.Name
	}

	if req.Role != "" {
		if req.Role == "admin" {
			user.Role = domain.UserRoleAdmin
		} else {
			user.Role = domain.UserRoleUser
		}
	}

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *testUserManagementHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	session, _ := h.sessionMgr.GetSession(r)
	if session != nil && session.UserID == id {
		http.Error(w, `{"error":"cannot delete your own account"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if err := h.userRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (h *testUserManagementHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, `{"error":"password is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < h.passwordMinLen {
		http.Error(w, `{"error":"password too short"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	hash := "hashed:" + req.Password
	if err := h.userRepo.UpdatePassword(r.Context(), id, hash); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// withUserMgmtChiURLParam helper for adding chi URL params
func withUserMgmtChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// Tests for RequireAdmin middleware

func Test_RequireAdmin_NoSession_ReturnsUnauthorized(t *testing.T) {
	h := newTestUserManagementHandler()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()

	h.RequireAdmin(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if nextCalled {
		t.Error("next handler should not be called")
	}
}

func Test_RequireAdmin_NonAdminUser_ReturnsForbidden(t *testing.T) {
	h := newTestUserManagementHandler()
	h.sessionMgr.session = &mockSessionData{
		UserID: uuid.New(),
		Role:   domain.UserRoleUser,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()

	h.RequireAdmin(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}
	if nextCalled {
		t.Error("next handler should not be called")
	}
}

func Test_RequireAdmin_AdminUser_CallsNext(t *testing.T) {
	h := newTestUserManagementHandler()
	h.sessionMgr.session = &mockSessionData{
		UserID: uuid.New(),
		Role:   domain.UserRoleAdmin,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()

	h.RequireAdmin(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Error("next handler should be called")
	}
}

// Tests for ListUsers

func Test_ListUsers_EmptyList_ReturnsEmptyArray(t *testing.T) {
	h := newTestUserManagementHandler()

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var users []UserResponse
	json.NewDecoder(rec.Body).Decode(&users)
	if len(users) != 0 {
		t.Errorf("expected empty array, got %d users", len(users))
	}
}

func Test_ListUsers_WithUsers_ReturnsUserList(t *testing.T) {
	h := newTestUserManagementHandler()

	name := "Test User"
	h.userRepo.addUser(&domain.User{
		ID:             uuid.New(),
		OrganizationID: h.defaultOrgID,
		Email:          "user1@example.com",
		DisplayName:    &name,
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var users []UserResponse
	json.NewDecoder(rec.Body).Decode(&users)
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Email != "user1@example.com" {
		t.Errorf("expected email user1@example.com, got %s", users[0].Email)
	}
}

// Tests for GetUser

func Test_GetUser_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	req := httptest.NewRequest(http.MethodGet, "/admin/users/invalid", nil)
	req = withUserMgmtChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_GetUser_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+id.String(), nil)
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_GetUser_Exists_ReturnsUser(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	name := "Test User"
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "test@example.com",
		DisplayName:    &name,
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users/"+id.String(), nil)
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var user UserResponse
	json.NewDecoder(rec.Body).Decode(&user)
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
}

// Tests for CreateUser

func Test_CreateUser_InvalidBody_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString("invalid json"))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateUser_MissingEmail_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateUser_MissingPassword_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateUser_PasswordTooShort_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"email":"test@example.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_CreateUser_EmailExists_ReturnsConflict(t *testing.T) {
	h := newTestUserManagementHandler()
	h.userRepo.addUser(&domain.User{
		ID:             uuid.New(),
		OrganizationID: h.defaultOrgID,
		Email:          "existing@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	body := `{"email":"existing@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}
}

func Test_CreateUser_ValidRequest_ReturnsCreated(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"email":"new@example.com","password":"password123","name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var user UserResponse
	json.NewDecoder(rec.Body).Decode(&user)
	if user.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", user.Email)
	}
	if user.Role != "user" {
		t.Errorf("expected role user, got %s", user.Role)
	}
}

func Test_CreateUser_AdminRole_SetsAdminRole(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"email":"admin@example.com","password":"password123","role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var user UserResponse
	json.NewDecoder(rec.Body).Decode(&user)
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

// Tests for UpdateUser

func Test_UpdateUser_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/users/invalid", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateUser_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+id.String(), bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_UpdateUser_InvalidBody_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+id.String(), bytes.NewBufferString("invalid"))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_UpdateUser_EmailTaken_ReturnsConflict(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})
	h.userRepo.addUser(&domain.User{
		ID:             uuid.New(),
		OrganizationID: h.defaultOrgID,
		Email:          "other@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	body := `{"email":"other@example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+id.String(), bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}
}

func Test_UpdateUser_ValidRequest_ReturnsUpdatedUser(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	body := `{"name":"Updated Name","role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+id.String(), bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var user UserResponse
	json.NewDecoder(rec.Body).Decode(&user)
	if *user.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", *user.Name)
	}
	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}
}

// Tests for DeleteUser

func Test_DeleteUser_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/invalid", nil)
	req = withUserMgmtChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteUser_SelfDelete_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	h.sessionMgr.session = &mockSessionData{
		UserID: id,
		Role:   domain.UserRoleAdmin,
	}
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "admin@example.com",
		Role:           domain.UserRoleAdmin,
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+id.String(), nil)
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_DeleteUser_NotFound_ReturnsNotFound(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+id.String(), nil)
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_DeleteUser_ValidRequest_ReturnsSuccess(t *testing.T) {
	h := newTestUserManagementHandler()
	adminID := uuid.New()
	h.sessionMgr.session = &mockSessionData{
		UserID: adminID,
		Role:   domain.UserRoleAdmin,
	}

	userID := uuid.New()
	h.userRepo.addUser(&domain.User{
		ID:             userID,
		OrganizationID: h.defaultOrgID,
		Email:          "user@example.com",
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/"+userID.String(), nil)
	req = withUserMgmtChiURLParam(req, "id", userID.String())
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify user was deleted
	if _, exists := h.userRepo.users[userID]; exists {
		t.Error("expected user to be deleted from repository")
	}
}

// Tests for ResetPassword

func Test_ResetPassword_InvalidID_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()

	body := `{"password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/invalid/reset-password", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", "invalid")
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_ResetPassword_InvalidBody_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id.String()+"/reset-password", bytes.NewBufferString("invalid"))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_ResetPassword_MissingPassword_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id.String()+"/reset-password", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_ResetPassword_PasswordTooShort_ReturnsBadRequest(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	body := `{"password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id.String()+"/reset-password", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_ResetPassword_UserNotFound_ReturnsNotFound(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()

	body := `{"password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id.String()+"/reset-password", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func Test_ResetPassword_ValidRequest_ReturnsSuccess(t *testing.T) {
	h := newTestUserManagementHandler()
	id := uuid.New()
	oldHash := "oldhash"
	h.userRepo.addUser(&domain.User{
		ID:             id,
		OrganizationID: h.defaultOrgID,
		Email:          "user@example.com",
		PasswordHash:   &oldHash,
		Role:           domain.UserRoleUser,
		CreatedAt:      time.Now(),
	})

	body := `{"password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+id.String()+"/reset-password", bytes.NewBufferString(body))
	req = withUserMgmtChiURLParam(req, "id", id.String())
	rec := httptest.NewRecorder()

	h.ResetPassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify password was updated
	user := h.userRepo.users[id]
	if user.PasswordHash == nil || *user.PasswordHash == oldHash {
		t.Error("expected password hash to be updated")
	}
}

// Tests for toUserResponse helper

func Test_toUserResponse_WithAllFields_ReturnsCorrectResponse(t *testing.T) {
	name := "Test User"
	hash := "somehash"
	oidc := "oidc-subject"
	user := &domain.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		DisplayName:  &name,
		Role:         domain.UserRoleAdmin,
		PasswordHash: &hash,
		OIDCSubject:  &oidc,
		CreatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	resp := toUserResponse(user)

	if resp.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp.Email)
	}
	if *resp.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", *resp.Name)
	}
	if resp.Role != "admin" {
		t.Errorf("expected role admin, got %s", resp.Role)
	}
	if !resp.HasPassword {
		t.Error("expected HasPassword to be true")
	}
	if !resp.HasOIDC {
		t.Error("expected HasOIDC to be true")
	}
}

func Test_toUserResponse_WithNilOptionalFields_HandlesNil(t *testing.T) {
	user := &domain.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Role:      domain.UserRoleUser,
		CreatedAt: time.Now(),
	}

	resp := toUserResponse(user)

	if resp.Name != nil {
		t.Errorf("expected nil name, got %v", resp.Name)
	}
	if resp.HasPassword {
		t.Error("expected HasPassword to be false")
	}
	if resp.HasOIDC {
		t.Error("expected HasOIDC to be false")
	}
}
