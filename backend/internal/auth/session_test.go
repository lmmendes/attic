package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

func Test_NewSessionManager_CreatesManager(t *testing.T) {
	manager := NewSessionManager("test-secret", 24)

	if manager == nil {
		t.Fatal("expected non-nil session manager")
	}
	if manager.durationHours != 24 {
		t.Errorf("expected duration 24, got %d", manager.durationHours)
	}
}

func Test_NewSessionManager_PadsShortSecret(t *testing.T) {
	manager := NewSessionManager("short", 24)

	if manager == nil {
		t.Fatal("expected non-nil session manager")
	}
	if len(manager.secret) != 32 {
		t.Errorf("expected secret length 32, got %d", len(manager.secret))
	}
}

func Test_CreateSession_SetsSessionCookie(t *testing.T) {
	manager := NewSessionManager("test-secret-key-32-bytes-long!!", 24)
	displayName := "Test User"
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		DisplayName: &displayName,
		Role:        domain.UserRoleUser,
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	err := manager.CreateSession(rec, req, user)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "attic_session" {
			found = true
			if c.Value == "" {
				t.Error("expected non-empty cookie value")
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly to be true")
			}
			if c.Path != "/" {
				t.Errorf("expected path '/', got '%s'", c.Path)
			}
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func Test_GetSession_ValidSession_ReturnsSession(t *testing.T) {
	manager := NewSessionManager("test-secret-key-32-bytes-long!!", 24)
	displayName := "Test User"
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		DisplayName: &displayName,
		Role:        domain.UserRoleAdmin,
	}

	// Create session
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	manager.CreateSession(createRec, createReq, user)

	// Get the cookie
	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Retrieve session
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(sessionCookie)

	session, err := manager.GetSession(getReq)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, session.UserID)
	}
	if session.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, session.Email)
	}
	if session.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", session.Name)
	}
	if session.Role != domain.UserRoleAdmin {
		t.Errorf("expected role admin, got %s", session.Role)
	}
}

func Test_GetSession_NoSession_ReturnsError(t *testing.T) {
	manager := NewSessionManager("test-secret", 24)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	session, err := manager.GetSession(req)

	if err == nil {
		t.Error("expected error when no session cookie")
	}
	if session != nil {
		t.Error("expected nil session when no cookie")
	}
}

func Test_GetSession_InvalidCookie_ReturnsError(t *testing.T) {
	manager := NewSessionManager("test-secret", 24)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "attic_session",
		Value: "invalid-base64-!!!",
	})

	session, err := manager.GetSession(req)

	if err == nil {
		t.Error("expected error for invalid cookie")
	}
	if session != nil {
		t.Error("expected nil session for invalid cookie")
	}
}

func Test_ClearSession_RemovesCookie(t *testing.T) {
	manager := NewSessionManager("test-secret", 24)

	rec := httptest.NewRecorder()
	manager.ClearSession(rec)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "attic_session" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("expected MaxAge -1 to clear cookie, got %d", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("expected empty value, got '%s'", c.Value)
			}
		}
	}
	if !found {
		t.Error("expected session cookie in response")
	}
}

func Test_GetSessionInfo_Authenticated_ReturnsUserInfo(t *testing.T) {
	manager := NewSessionManager("test-secret-key-32-bytes-long!!", 24)
	displayName := "Test User"
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		DisplayName: &displayName,
		Role:        domain.UserRoleUser,
	}

	// Create session
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	manager.CreateSession(createRec, createReq, user)

	// Get the cookie
	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Get session info
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(sessionCookie)

	info := manager.GetSessionInfo(getReq)

	if info["authenticated"] != true {
		t.Error("expected authenticated to be true")
	}
	userInfo := info["user"].(map[string]any)
	if userInfo["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%v'", userInfo["email"])
	}
}

func Test_GetSessionInfo_NotAuthenticated_ReturnsFalse(t *testing.T) {
	manager := NewSessionManager("test-secret", 24)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	info := manager.GetSessionInfo(req)

	if info["authenticated"] != false {
		t.Error("expected authenticated to be false")
	}
	if info["user"] != nil {
		t.Error("expected user to be nil when not authenticated")
	}
}

func Test_Session_ExpirationIsSet(t *testing.T) {
	manager := NewSessionManager("test-secret-key-32-bytes-long!!", 2) // 2 hours
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  domain.UserRoleUser,
	}

	// Create session
	createReq := httptest.NewRequest(http.MethodPost, "/", nil)
	createRec := httptest.NewRecorder()
	manager.CreateSession(createRec, createReq, user)

	// Get the cookie
	var sessionCookie *http.Cookie
	for _, c := range createRec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	// Retrieve session
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(sessionCookie)

	session, _ := manager.GetSession(getReq)

	// Check expiration is roughly 2 hours from now
	expectedExpiry := time.Now().Add(2 * time.Hour)
	diff := session.ExpiresAt.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, session.ExpiresAt)
	}
}

func Test_CreateSession_UserWithoutDisplayName_HandlesNil(t *testing.T) {
	manager := NewSessionManager("test-secret-key-32-bytes-long!!", 24)
	user := &domain.User{
		ID:          uuid.New(),
		Email:       "test@example.com",
		DisplayName: nil, // No display name
		Role:        domain.UserRoleUser,
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	err := manager.CreateSession(rec, req, user)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify we can retrieve the session
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "attic_session" {
			sessionCookie = c
			break
		}
	}

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.AddCookie(sessionCookie)

	session, _ := manager.GetSession(getReq)
	if session.Name != "" {
		t.Errorf("expected empty name for nil display name, got '%s'", session.Name)
	}
}
