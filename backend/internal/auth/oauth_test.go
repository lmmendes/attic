package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

	req := httptest.NewRequest(http.MethodGet, "/auth/oidc/callback", nil)
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
		sessions: newTestOIDCStore(),
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
		sessions: newTestOIDCStore(),
	}

	// Create an expired session cookie
	session := Session{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired
		Subject:     "user-123",
		Email:       "test@example.com",
	}
	encoded := seedOIDCSession(t, handler, session)

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
		sessions: newTestOIDCStore(),
	}

	// Create a valid session cookie
	session := Session{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour), // Not expired
		Subject:     "user-123",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	encoded := seedOIDCSession(t, handler, session)

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
		sessions: newTestOIDCStore(),
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
		sessions: newTestOIDCStore(),
	}

	session := Session{
		AccessToken: "my-access-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	encoded := seedOIDCSession(t, handler, session)

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

func Test_OAuthHandler_GetIDToken_ValidSession_ReturnsToken(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		sessions: newTestOIDCStore(),
	}

	session := Session{
		IDToken:   "my-id-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	encoded := seedOIDCSession(t, handler, session)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  sessionCookieName,
		Value: encoded,
	})

	token := handler.GetIDToken(req)

	if token != "my-id-token" {
		t.Errorf("expected 'my-id-token', got '%s'", token)
	}
}

func Test_OAuthHandler_setSessionCookie_PersistsSession(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		sessions: newTestOIDCStore(),
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

	if len(sessionCookie.Value) != 43 {
		t.Fatalf("expected opaque ID, got %d bytes", len(sessionCookie.Value))
	}
	lookup := httptest.NewRequest(http.MethodGet, "/", nil)
	lookup.AddCookie(sessionCookie)
	decoded, err := handler.getSessionFromCookie(lookup)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.AccessToken != "access-123" || decoded.Email != "test@example.com" {
		t.Fatalf("unexpected session: %+v", decoded)
	}
}

func Test_OAuthHandler_getSessionFromCookie_LoadsSession(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		sessions: newTestOIDCStore(),
	}

	session := Session{
		AccessToken: "token-123",
		ExpiresAt:   time.Now().Add(time.Hour),
		Subject:     "sub-456",
		Email:       "decode@example.com",
	}
	encoded := seedOIDCSession(t, handler, session)

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
		sessions: newTestOIDCStore(),
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

func Test_OAuthHandler_getSessionFromCookie_InvalidID_ReturnsError(t *testing.T) {
	handler := &OAuthHandler{
		disabled: false,
		sessions: newTestOIDCStore(),
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
		t.Error("expected error for invalid ID")
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

// testOIDCStore mimics persistence without exposing its contents in cookies.
type testOIDCStore struct {
	mu       sync.Mutex
	sessions map[string]Session
	err      error
	cleanups chan struct{}
}

func newTestOIDCStore() *testOIDCStore { return &testOIDCStore{sessions: map[string]Session{}} }
func (s *testOIDCStore) Create(_ context.Context, hash string, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.sessions[hash] = *session
	return nil
}
func (s *testOIDCStore) Get(_ context.Context, hash string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	session, ok := s.sessions[hash]
	if !ok {
		return nil, nil
	}
	return &session, nil
}
func (s *testOIDCStore) Delete(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	delete(s.sessions, hash)
	return nil
}
func (s *testOIDCStore) DeleteExpired(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cleanups != nil {
		select {
		case s.cleanups <- struct{}{}:
		case <-ctx.Done():
		}
	}
	if s.err != nil {
		return s.err
	}
	for hash, session := range s.sessions {
		if !time.Now().Before(session.ExpiresAt) {
			delete(s.sessions, hash)
		}
	}
	return nil
}
func seedOIDCSession(t *testing.T, h *OAuthHandler, session Session) string {
	t.Helper()
	id := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
	hash, err := sessionIDHash(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.sessions.Create(context.Background(), hash, &session); err != nil {
		t.Fatal(err)
	}
	return id
}

func Test_OIDCSession_LargeTokensAndRevocation(t *testing.T) {
	store := newTestOIDCStore()
	h := &OAuthHandler{sessions: store}
	session := &Session{AccessToken: strings.Repeat("a", 5000), RefreshToken: strings.Repeat("r", 5000), IDToken: strings.Repeat("i", 5000), ExpiresAt: time.Now().Add(time.Hour)}
	req := httptest.NewRequest(http.MethodGet, "https://attic.example/auth/oidc/callback", nil)
	rec := httptest.NewRecorder()
	if err := h.setSessionCookie(rec, req, session); err != nil {
		t.Fatal(err)
	}
	cookie := rec.Result().Cookies()[0]
	if len(cookie.String()) >= 4096 || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("invalid cookie attributes: %s", cookie)
	}
	req.AddCookie(cookie)
	second := &OAuthHandler{sessions: store}
	if second.GetSessionInfo(req)["authenticated"] != true || second.GetIDToken(req) != session.IDToken {
		t.Fatal("session not shared across handlers")
	}
	hash, _ := sessionIDHash(cookie.Value)
	if hash == cookie.Value {
		t.Fatal("raw session ID stored")
	}
	if err := second.ClearSession(httptest.NewRecorder(), req); err != nil {
		t.Fatal(err)
	}
	if h.GetSessionInfo(req)["authenticated"] != false {
		t.Fatal("revoked session accepted")
	}
}

func Test_OIDCSession_StoreFailures(t *testing.T) {
	store := newTestOIDCStore()
	h := &OAuthHandler{sessions: store}
	session := Session{ExpiresAt: time.Now().Add(time.Hour)}
	id := seedOIDCSession(t, h, session)
	store.err = errors.New("database unavailable")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	rec := httptest.NewRecorder()
	if err := h.setSessionCookie(rec, req, &session); err == nil || len(rec.Result().Cookies()) != 0 {
		t.Fatal("cookie issued despite failed persistence")
	}
	if h.GetSessionInfo(req)["authenticated"] != false {
		t.Fatal("database failure authenticated")
	}
	rec = httptest.NewRecorder()
	if err := h.ClearSession(rec, req); err == nil {
		t.Fatal("expected revocation failure")
	}
	if rec.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("cookie not cleared on failure")
	}
}

func Test_OIDCSession_RejectsLegacyAndUnknownCookies(t *testing.T) {
	h := &OAuthHandler{sessions: newTestOIDCStore()}
	data, _ := json.Marshal(Session{ExpiresAt: time.Now().Add(time.Hour)})
	for _, id := range []string{base64.StdEncoding.EncodeToString(data), "invalid", base64.RawURLEncoding.EncodeToString(make([]byte, 32))} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
		if _, err := h.getSessionFromCookie(req); err == nil {
			t.Fatal("accepted invalid session")
		}
	}
}

func Test_OIDCSessionExpiry(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct{ access, id, want time.Time }{
		{now.Add(time.Hour), now.Add(2 * time.Hour), now.Add(time.Hour)},
		{now.Add(2 * time.Hour), now.Add(time.Hour), now.Add(time.Hour)},
		{time.Time{}, now.Add(48 * time.Hour), now.Add(cookieMaxAge)},
	} {
		if got := oidcSessionExpiry(now, tc.access, tc.id); !got.Equal(tc.want) {
			t.Fatalf("got %s, want %s", got, tc.want)
		}
	}
}

func Test_OIDCSessionCleanup_RetriesAndStops(t *testing.T) {
	store := newTestOIDCStore()
	store.err = errors.New("temporary failure")
	store.cleanups = make(chan struct{}, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); runOIDCSessionCleanup(ctx, store, 10*time.Millisecond) }()
	wait := func(ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("cleanup timed out")
		}
	}
	wait(store.cleanups)
	store.mu.Lock()
	store.err = nil
	store.sessions["expired"] = Session{ExpiresAt: time.Now().Add(-time.Hour)}
	store.sessions["valid"] = Session{ExpiresAt: time.Now().Add(time.Hour)}
	store.mu.Unlock()
	wait(store.cleanups)
	store.mu.Lock()
	_, expired := store.sessions["expired"]
	_, valid := store.sessions["valid"]
	store.mu.Unlock()
	if expired || !valid {
		t.Fatal("cleanup did not preserve only active sessions")
	}
	cancel()
	wait(done)
}
