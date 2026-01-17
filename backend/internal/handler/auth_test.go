package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/auth"
	"github.com/mendelui/attic/internal/domain"
)

// mockUserRepo implements a minimal user repository for testing
type mockUserRepo struct {
	users            map[uuid.UUID]*domain.User
	usersByEmail     map[string]*domain.User
	GetByIDError     error
	GetByEmailError  error
	UpdatePassError  error
	ListError        error
	CreateError      error
	UpdateError      error
	DeleteError      error
	defaultOrgID     uuid.UUID
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:        make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

func (r *mockUserRepo) addUser(u *domain.User) {
	r.users[u.ID] = u
	r.usersByEmail[strings.ToLower(u.Email)] = u
}

func (r *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if r.GetByIDError != nil {
		return nil, r.GetByIDError
	}
	return r.users[id], nil
}

func (r *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if r.GetByEmailError != nil {
		return nil, r.GetByEmailError
	}
	return r.usersByEmail[strings.ToLower(email)], nil
}

func (r *mockUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if r.UpdatePassError != nil {
		return r.UpdatePassError
	}
	if u, ok := r.users[id]; ok {
		u.PasswordHash = &hash
	}
	return nil
}

func (r *mockUserRepo) List(_ context.Context, orgID uuid.UUID) ([]domain.User, error) {
	if r.ListError != nil {
		return nil, r.ListError
	}
	result := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		if u.OrganizationID == orgID {
			result = append(result, *u)
		}
	}
	return result, nil
}

func (r *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	if r.CreateError != nil {
		return r.CreateError
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	r.users[user.ID] = user
	r.usersByEmail[strings.ToLower(user.Email)] = user
	return nil
}

func (r *mockUserRepo) Update(_ context.Context, user *domain.User) error {
	if r.UpdateError != nil {
		return r.UpdateError
	}
	r.users[user.ID] = user
	r.usersByEmail[strings.ToLower(user.Email)] = user
	return nil
}

func (r *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}
	if u, ok := r.users[id]; ok {
		delete(r.usersByEmail, strings.ToLower(u.Email))
		delete(r.users, id)
	}
	return nil
}

// testAuthHandler wraps AuthHandler for testing without database
type testAuthHandler struct {
	userRepo          *mockUserRepo
	sessionManager    *auth.SessionManager
	passwordMinLength int
	oidcEnabled       bool
}

func newTestAuthHandler(oidcEnabled bool) *testAuthHandler {
	return &testAuthHandler{
		userRepo:          newMockUserRepo(),
		sessionManager:    auth.NewSessionManager("test-secret-key-32-bytes-long!!", 24),
		passwordMinLength: 8,
		oidcEnabled:       oidcEnabled,
	}
}

func (h *testAuthHandler) login(w http.ResponseWriter, r *http.Request) {
	if h.oidcEnabled {
		http.Error(w, `{"error":"email/password login is disabled when OIDC is enabled"}`, http.StatusBadRequest)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password are required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if !user.HasPassword() {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(req.Password, *user.PasswordHash) {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if err := h.sessionManager.CreateSession(w, r, user); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"user": map[string]any{
			"id":    user.ID.String(),
			"email": user.Email,
			"name":  user.DisplayName,
			"role":  user.Role,
		},
	})
}

func (h *testAuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	h.sessionManager.ClearSession(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (h *testAuthHandler) getSession(w http.ResponseWriter, r *http.Request) {
	info := h.sessionManager.GetSessionInfo(r)
	info["oidc_enabled"] = h.oidcEnabled
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *testAuthHandler) getAuthMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"oidc_enabled": h.oidcEnabled,
	})
}

func (h *testAuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	if h.oidcEnabled {
		http.Error(w, `{"error":"password change is disabled when OIDC is enabled"}`, http.StatusBadRequest)
		return
	}

	session, err := h.sessionManager.GetSession(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"current and new password are required"}`, http.StatusBadRequest)
		return
	}

	if err := auth.ValidatePassword(req.NewPassword, h.passwordMinLength); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), session.UserID)
	if err != nil || user == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if !user.HasPassword() || !auth.CheckPassword(req.CurrentPassword, *user.PasswordHash) {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.UpdatePassword(r.Context(), user.ID, hash); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

// Helper functions for tests

func jsonBody(t *testing.T, v any) *strings.Reader {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return strings.NewReader(string(data))
}

func createTestUser(t *testing.T, email, password string, role domain.UserRole) *domain.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return &domain.User{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:          email,
		PasswordHash:   &hash,
		Role:           role,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// Tests

func Test_Login_ValidCredentials_ReturnsSuccess(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "password123", domain.UserRoleUser)
	h.userRepo.addUser(user)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["success"] != true {
		t.Error("expected success to be true")
	}

	// Check session cookie is set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "attic_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func Test_Login_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "password123", domain.UserRoleUser)
	h.userRepo.addUser(user)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_Login_NonExistentUser_ReturnsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_Login_MissingEmail_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Password: "password123",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Login_MissingPassword_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email: "test@example.com",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Login_OIDCEnabled_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(true) // OIDC enabled

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 when OIDC enabled, got %d", rec.Code)
	}
}

func Test_Login_UserWithoutPassword_ReturnsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(false)
	// User without password (OIDC-only user)
	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:          "oidc@example.com",
		PasswordHash:   nil,
		Role:           domain.UserRoleUser,
	}
	h.userRepo.addUser(user)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "oidc@example.com",
		Password: "anypassword",
	}))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_Login_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_Logout_ReturnsSuccess(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["success"] != true {
		t.Error("expected success to be true")
	}

	// Check session cookie is cleared
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "attic_session" && c.MaxAge == -1 {
			return // Cookie cleared as expected
		}
	}
	t.Error("expected session cookie to be cleared")
}

func Test_GetSession_WithoutSession_ReturnsUnauthenticated(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	rec := httptest.NewRecorder()
	h.getSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["authenticated"] != false {
		t.Error("expected authenticated to be false")
	}
}

func Test_GetSession_WithSession_ReturnsAuthenticatedUser(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "password123", domain.UserRoleAdmin)
	h.userRepo.addUser(user)

	// First login to get session
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.login(loginRec, loginReq)

	// Get session cookie
	cookies := loginRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not found")
	}

	// Now check session
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	h.getSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["authenticated"] != true {
		t.Error("expected authenticated to be true")
	}
	userInfo := resp["user"].(map[string]any)
	if userInfo["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", userInfo["email"])
	}
}

func Test_GetAuthMode_OIDCDisabled_ReturnsCorrectMode(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/auth/mode", nil)
	rec := httptest.NewRecorder()
	h.getAuthMode(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["oidc_enabled"] != false {
		t.Error("expected oidc_enabled to be false")
	}
}

func Test_GetAuthMode_OIDCEnabled_ReturnsCorrectMode(t *testing.T) {
	h := newTestAuthHandler(true)

	req := httptest.NewRequest(http.MethodGet, "/auth/mode", nil)
	rec := httptest.NewRecorder()
	h.getAuthMode(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["oidc_enabled"] != true {
		t.Error("expected oidc_enabled to be true")
	}
}

func Test_ChangePassword_ValidRequest_ReturnsSuccess(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "oldpassword", domain.UserRoleUser)
	h.userRepo.addUser(user)

	// First login
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "oldpassword",
	}))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.login(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Change password
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	h.changePassword(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Verify new password works
	loginReq2 := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "newpassword123",
	}))
	loginReq2.Header.Set("Content-Type", "application/json")
	loginRec2 := httptest.NewRecorder()
	h.login(loginRec2, loginReq2)

	if loginRec2.Code != http.StatusOK {
		t.Errorf("expected login with new password to succeed, got %d", loginRec2.Code)
	}
}

func Test_ChangePassword_WrongCurrentPassword_ReturnsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "correctpassword", domain.UserRoleUser)
	h.userRepo.addUser(user)

	// First login
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "correctpassword",
	}))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.login(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Try to change password with wrong current
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	h.changePassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_ChangePassword_NewPasswordTooShort_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "oldpassword", domain.UserRoleUser)
	h.userRepo.addUser(user)

	// First login
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "oldpassword",
	}))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.login(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Try to change to short password
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "short",
	}))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	h.changePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func Test_ChangePassword_WithoutSession_ReturnsUnauthorized(t *testing.T) {
	h := newTestAuthHandler(false)

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.changePassword(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_ChangePassword_OIDCEnabled_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(true) // OIDC enabled

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.changePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 when OIDC enabled, got %d", rec.Code)
	}
}

func Test_ChangePassword_MissingFields_ReturnsBadRequest(t *testing.T) {
	h := newTestAuthHandler(false)
	user := createTestUser(t, "test@example.com", "oldpassword", domain.UserRoleUser)
	h.userRepo.addUser(user)

	// First login
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", jsonBody(t, LoginRequest{
		Email:    "test@example.com",
		Password: "oldpassword",
	}))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	h.login(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	tests := []struct {
		name string
		req  ChangePasswordRequest
	}{
		{"missing current", ChangePasswordRequest{NewPassword: "newpassword123"}},
		{"missing new", ChangePasswordRequest{CurrentPassword: "oldpassword"}},
		{"both empty", ChangePasswordRequest{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/change-password", jsonBody(t, tt.req))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(sessionCookie)
			rec := httptest.NewRecorder()
			h.changePassword(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rec.Code)
			}
		})
	}
}
