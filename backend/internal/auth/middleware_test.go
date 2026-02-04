package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

// Tests for disabled middleware

func Test_Middleware_Disabled_AllowsAllRequests(t *testing.T) {
	m := &Middleware{disabled: true}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func Test_Middleware_Disabled_SetsDevUserClaims(t *testing.T) {
	m := &Middleware{disabled: true}

	var claims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims = GetClaims(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if claims == nil {
		t.Fatal("expected claims to be set")
	}
	if claims.Email != "dev@example.com" {
		t.Errorf("expected dev email, got %s", claims.Email)
	}
	if claims.Subject != "dev-user" {
		t.Errorf("expected dev-user subject, got %s", claims.Subject)
	}
}

// Tests for local authentication

func Test_Middleware_Local_ValidSession_AllowsRequest(t *testing.T) {
	sm := NewSessionManager("test-secret-key-32-bytes-long!!", 24)
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: sm,
	}

	// Create a session
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	sm.CreateSession(createRec, createReq, user)

	// Get the cookie
	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	nextCalled := false
	var claims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		claims = GetClaims(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	if claims == nil {
		t.Fatal("expected claims to be set")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", claims.Email)
	}
}

func Test_Middleware_Local_NoSession_ReturnsUnauthorized(t *testing.T) {
	sm := NewSessionManager("test-secret", 24)
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: sm,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_Middleware_Local_InvalidSession_ReturnsUnauthorized(t *testing.T) {
	sm := NewSessionManager("test-secret", 24)
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: sm,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "attic_session",
		Value: "invalid-session-data",
	})
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_Middleware_Local_NoSessionManager_ReturnsInternalError(t *testing.T) {
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: nil,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

// Tests for GetClaims

func Test_GetClaims_WithValidContext_ReturnsClaims(t *testing.T) {
	claims := &Claims{
		Subject: "user-123",
		Email:   "test@example.com",
	}
	ctx := context.WithValue(context.Background(), UserContextKey, claims)

	result := GetClaims(ctx)

	if result == nil {
		t.Fatal("expected claims to be returned")
	}
	if result.Subject != "user-123" {
		t.Errorf("expected subject user-123, got %s", result.Subject)
	}
}

func Test_GetClaims_WithEmptyContext_ReturnsNil(t *testing.T) {
	ctx := context.Background()

	result := GetClaims(ctx)

	if result != nil {
		t.Error("expected nil claims for empty context")
	}
}

func Test_GetClaims_WithWrongType_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserContextKey, "not a claims struct")

	result := GetClaims(ctx)

	if result != nil {
		t.Error("expected nil claims for wrong type")
	}
}

// Tests for Optional middleware

func Test_Optional_NoAuthHeader_AllowsRequest(t *testing.T) {
	m := &Middleware{disabled: false, oidcEnabled: false}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	m.Optional(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called without auth header")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func Test_Optional_WithAuthHeader_ValidatesToken(t *testing.T) {
	sm := NewSessionManager("test-secret", 24)
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: sm,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	m.Optional(next).ServeHTTP(rec, req)

	// Should try to validate the token and fail
	if nextCalled {
		t.Error("expected next handler NOT to be called with invalid token")
	}
}

// Tests for RequireAdmin middleware

func Test_RequireAdmin_NoSession_ReturnsUnauthorized(t *testing.T) {
	sm := NewSessionManager("test-secret", 24)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	RequireAdmin(sm)(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func Test_RequireAdmin_NonAdminUser_ReturnsForbidden(t *testing.T) {
	sm := NewSessionManager("test-secret-key-32-bytes-long!!", 24)

	// Create a non-admin session
	user := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  domain.UserRoleUser,
	}
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	sm.CreateSession(createRec, createReq, user)

	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()

	RequireAdmin(sm)(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called for non-admin")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}
}

func Test_RequireAdmin_AdminUser_AllowsRequest(t *testing.T) {
	sm := NewSessionManager("test-secret-key-32-bytes-long!!", 24)

	// Create an admin session
	user := &domain.User{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Role:  domain.UserRoleAdmin,
	}
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	sm.CreateSession(createRec, createReq, user)

	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()

	RequireAdmin(sm)(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called for admin")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// Tests for SetOAuthHandler and SetSessionManager

func Test_SetOAuthHandler_SetsHandler(t *testing.T) {
	m := &Middleware{}

	if m.oauth != nil {
		t.Error("expected oauth to be nil initially")
	}

	m.SetOAuthHandler(&OAuthHandler{})

	if m.oauth == nil {
		t.Error("expected oauth to be set")
	}
}

func Test_SetSessionManager_SetsManager(t *testing.T) {
	m := &Middleware{}

	if m.sessionManager != nil {
		t.Error("expected sessionManager to be nil initially")
	}

	m.SetSessionManager(NewSessionManager("test", 24))

	if m.sessionManager == nil {
		t.Error("expected sessionManager to be set")
	}
}

// Test for NewMiddleware with disabled config

func Test_NewMiddleware_Disabled_ReturnsDisabledMiddleware(t *testing.T) {
	cfg := Config{Disabled: true}

	m, err := NewMiddleware(context.Background(), cfg)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !m.disabled {
		t.Error("expected middleware to be disabled")
	}
}

// Tests for Claims struct

func Test_Claims_FieldsAreAccessible(t *testing.T) {
	claims := Claims{
		Subject:     "user-123",
		Email:       "test@example.com",
		Name:        "Test User",
		DisplayName: "testuser",
	}

	if claims.Subject != "user-123" {
		t.Errorf("expected subject user-123, got %s", claims.Subject)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", claims.Email)
	}
	if claims.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", claims.Name)
	}
	if claims.DisplayName != "testuser" {
		t.Errorf("expected displayName testuser, got %s", claims.DisplayName)
	}
}

// Test for session expiration

func Test_Middleware_Local_ExpiredSession_ReturnsUnauthorized(t *testing.T) {
	sm := NewSessionManager("test-secret-key-32-bytes-long!!", 0) // 0 hours = immediate expiration
	m := &Middleware{
		disabled:       false,
		oidcEnabled:    false,
		sessionManager: sm,
	}

	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	sm.CreateSession(createRec, createReq, user)

	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Wait a moment for "expiration"
	time.Sleep(10 * time.Millisecond)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()

	m.Authenticate(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called for expired session")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}
