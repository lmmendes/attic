package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_NewOAuthHandler_Disabled_ReturnsHandler(t *testing.T) {
	ctx := context.Background()
	cfg := OAuthConfig{
		Disabled: true,
		BaseURL:  "http://localhost:3000",
	}

	handler, err := NewOAuthHandler(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected handler to be created")
	}
	if !handler.disabled {
		t.Error("expected handler to be disabled")
	}
}

func Test_OAuthHandler_Login_Disabled_RedirectsToHome(t *testing.T) {
	handler := &OAuthHandler{disabled: true, baseURL: "http://localhost:3000"}

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	if rec.Header().Get("Location") != "/" {
		t.Errorf("expected redirect to '/', got '%s'", rec.Header().Get("Location"))
	}
}

func Test_OAuthHandler_Callback_Disabled_RedirectsToHome(t *testing.T) {
	handler := &OAuthHandler{disabled: true, baseURL: "http://localhost:3000"}

	req := httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	if rec.Header().Get("Location") != "/" {
		t.Errorf("expected redirect to '/', got '%s'", rec.Header().Get("Location"))
	}
}

func Test_OAuthHandler_Logout_Disabled_RedirectsToHome(t *testing.T) {
	handler := &OAuthHandler{disabled: true, baseURL: "http://localhost:3000"}

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	// Should clear the session cookie
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("expected session cookie to be set")
	} else if sessionCookie.MaxAge != -1 {
		t.Error("expected session cookie to be cleared (MaxAge=-1)")
	}
}

func Test_OAuthHandler_GetSession_Disabled_ReturnsMockSession(t *testing.T) {
	handler := &OAuthHandler{disabled: true, baseURL: "http://localhost:3000"}

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	rec := httptest.NewRecorder()

	handler.GetSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]any
	json.NewDecoder(rec.Body).Decode(&response)

	if response["authenticated"] != true {
		t.Error("expected authenticated to be true")
	}

	user := response["user"].(map[string]any)
	if user["email"] != "dev@example.com" {
		t.Errorf("expected dev email, got %v", user["email"])
	}
}

func Test_OAuthHandler_GetSession_NoSession_ReturnsUnauthenticated(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		baseURL:  "http://localhost:3000",
		secret:   make([]byte, 32),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	rec := httptest.NewRecorder()

	handler.GetSession(rec, req)

	var response map[string]any
	json.NewDecoder(rec.Body).Decode(&response)

	if response["authenticated"] != false {
		t.Error("expected authenticated to be false")
	}
}

func Test_OAuthHandler_GetSession_ExpiredSession_ReturnsUnauthenticated(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		baseURL:  "http://localhost:3000",
		secret:   make([]byte, 32),
	}

	// Create an expired session cookie
	session := Session{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired
		Subject:     "user-123",
		Email:       "test@example.com",
	}
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})
	rec := httptest.NewRecorder()

	handler.GetSession(rec, req)

	var response map[string]any
	json.NewDecoder(rec.Body).Decode(&response)

	if response["authenticated"] != false {
		t.Error("expected authenticated to be false for expired session")
	}
}

func Test_OAuthHandler_GetSession_ValidSession_ReturnsAuthenticated(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		baseURL:  "http://localhost:3000",
		secret:   make([]byte, 32),
	}

	// Create a valid session cookie
	session := Session{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour), // Not expired
		Subject:     "user-123",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})
	rec := httptest.NewRecorder()

	handler.GetSession(rec, req)

	var response map[string]any
	json.NewDecoder(rec.Body).Decode(&response)

	if response["authenticated"] != true {
		t.Error("expected authenticated to be true")
	}

	user := response["user"].(map[string]any)
	if user["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", user["email"])
	}
	if user["sub"] != "user-123" {
		t.Errorf("expected subject user-123, got %v", user["sub"])
	}
}

func Test_OAuthHandler_GetAccessToken_NoSession_ReturnsEmpty(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token := handler.GetAccessToken(req)

	if token != "" {
		t.Errorf("expected empty token, got '%s'", token)
	}
}

func Test_OAuthHandler_GetAccessToken_ValidSession_ReturnsToken(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	session := Session{
		AccessToken: "my-access-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})

	token := handler.GetAccessToken(req)

	if token != "my-access-token" {
		t.Errorf("expected 'my-access-token', got '%s'", token)
	}
}

func Test_OAuthHandler_setSessionCookie_EncodesSession(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	session := &Session{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Subject:      "user-789",
		Email:        "test@example.com",
		Name:         "Test User",
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	err := handler.setSessionCookie(rec, req, session)
	if err != nil {
		t.Fatalf("failed to set session cookie: %v", err)
	}

	// Find the session cookie
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set")
	}

	// Decode and verify
	data, err := base64.StdEncoding.DecodeString(sessionCookie.Value)
	if err != nil {
		t.Fatalf("failed to decode cookie: %v", err)
	}

	var decoded Session
	json.Unmarshal(data, &decoded)

	if decoded.AccessToken != "access-123" {
		t.Errorf("expected access token 'access-123', got '%s'", decoded.AccessToken)
	}
	if decoded.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", decoded.Email)
	}
}

func Test_OAuthHandler_getSessionFromCookie_DecodesSession(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	session := Session{
		AccessToken: "token-123",
		Subject:     "sub-456",
		Email:       "decode@example.com",
	}
	data, _ := json.Marshal(session)
	encoded := base64.StdEncoding.EncodeToString(data)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})

	decoded, err := handler.getSessionFromCookie(req)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	if decoded.AccessToken != "token-123" {
		t.Errorf("expected access token 'token-123', got '%s'", decoded.AccessToken)
	}
	if decoded.Email != "decode@example.com" {
		t.Errorf("expected email 'decode@example.com', got '%s'", decoded.Email)
	}
}

func Test_OAuthHandler_getSessionFromCookie_InvalidBase64_ReturnsError(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: "not-valid-base64!!!",
	})

	_, err := handler.getSessionFromCookie(req)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func Test_OAuthHandler_getSessionFromCookie_InvalidJSON_ReturnsError(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		secret:   make([]byte, 32),
	}

	// Valid base64 but invalid JSON
	encoded := base64.StdEncoding.EncodeToString([]byte("not json"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})

	_, err := handler.getSessionFromCookie(req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func Test_generateRandomString_ReturnsCorrectLength(t *testing.T) {
	lengths := []int{8, 16, 32, 64}

	for _, length := range lengths {
		result := generateRandomString(length)
		if len(result) != length {
			t.Errorf("expected length %d, got %d", length, len(result))
		}
	}
}

func Test_generateRandomString_ReturnsUniqueStrings(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		s := generateRandomString(32)
		if seen[s] {
			t.Error("generated duplicate random string")
		}
		seen[s] = true
	}
}

func Test_Session_Struct_Fields(t *testing.T) {
	expiry := time.Now().Add(1 * time.Hour)
	session := Session{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    expiry,
		Subject:      "sub",
		Email:        "email@test.com",
		Name:         "Test Name",
	}

	if session.AccessToken != "access" {
		t.Error("AccessToken not set correctly")
	}
	if session.RefreshToken != "refresh" {
		t.Error("RefreshToken not set correctly")
	}
	if session.Subject != "sub" {
		t.Error("Subject not set correctly")
	}
	if session.Email != "email@test.com" {
		t.Error("Email not set correctly")
	}
	if session.Name != "Test Name" {
		t.Error("Name not set correctly")
	}
}
