package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/lmmendes/attic/internal/domain"
	"golang.org/x/oauth2"
)

const (
	stateCookieName   = "oauth_state"
	sessionCookieName = "session"
	cookieMaxAge      = 24 * time.Hour
)

// Session contains OIDC credentials stored server-side.
type Session = domain.OIDCSession

// OIDCSessionStore persists OIDC sessions using a hash of the browser session ID.
type OIDCSessionStore interface {
	Create(context.Context, string, *Session) error
	Get(context.Context, string) (*Session, error)
	Delete(context.Context, string) error
	DeleteExpired(context.Context) error
}

// OAuthHandler handles OAuth login flow
type OAuthHandler struct {
	provider           *oidc.Provider
	oauth2Config       oauth2.Config
	verifier           *oidc.IDTokenVerifier
	baseURL            string
	sessions           OIDCSessionStore
	disabled           bool
	endSessionEndpoint string
}

// OAuthConfig for OAuth handler
type OAuthConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	BaseURL      string
	SessionStore OIDCSessionStore
	Disabled     bool
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(ctx context.Context, cfg OAuthConfig) (*OAuthHandler, error) {
	if cfg.Disabled {
		return &OAuthHandler{baseURL: cfg.BaseURL, disabled: true}, nil
	}
	if cfg.SessionStore == nil {
		return nil, errors.New("OIDC session store is required")
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.BaseURL + "/auth/oidc/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	// Extract end_session_endpoint from OIDC discovery document
	var providerClaims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&providerClaims); err != nil {
		slog.Warn("failed to extract end_session_endpoint from OIDC discovery", "error", err)
	}

	return &OAuthHandler{
		provider: provider, oauth2Config: oauth2Config, verifier: verifier,
		baseURL: cfg.BaseURL, sessions: cfg.SessionStore,
		endSessionEndpoint: providerClaims.EndSessionEndpoint,
	}, nil
}

// Login redirects to the OAuth provider
func (h *OAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.disabled {
		// In disabled mode, just redirect to home
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Generate state for CSRF protection
	state := generateRandomString(32)

	// Store state in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to OAuth provider
	authURL := h.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.disabled {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		slog.Error("missing state cookie", "error", err)
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		slog.Error("state mismatch")
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for error from provider
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		slog.Error("OAuth error", "error", errMsg, "description", errDesc)
		http.Error(w, "OAuth error: "+errMsg, http.StatusBadRequest)
		return
	}

	// Exchange code for tokens
	code := r.URL.Query().Get("code")
	token, err := h.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("failed to exchange code", "error", err)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		slog.Error("no id_token in response")
		http.Error(w, "No ID token", http.StatusInternalServerError)
		return
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		slog.Error("failed to verify id_token", "error", err)
		http.Error(w, "Invalid ID token", http.StatusInternalServerError)
		return
	}

	// Extract claims
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("failed to parse claims", "error", err)
		http.Error(w, "Invalid claims", http.StatusInternalServerError)
		return
	}

	// Create session
	session := Session{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		ExpiresAt:    oidcSessionExpiry(time.Now(), token.Expiry, idToken.Expiry),
		Subject:      idToken.Subject,
		Email:        claims.Email,
		Name:         claims.Name,
	}

	// Persist the session and send only its opaque ID to the browser
	if err := h.setSessionCookie(w, r, &session); err != nil {
		slog.Error("failed to set session cookie", "error", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// Logout clears the session
func (h *OAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Read session before clearing so we can pass id_token_hint to the provider
	session, _ := h.getSessionFromCookie(r)

	if err := h.ClearSession(w, r); err != nil {
		slog.Error("failed to revoke OIDC session", "error", err)
		http.Error(w, "Failed to revoke session", http.StatusInternalServerError)
		return
	}

	if h.disabled {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	postLogoutRedirect := h.baseURL + "/login?logout=true"

	params := url.Values{}
	params.Set("post_logout_redirect_uri", postLogoutRedirect)
	params.Set("client_id", h.oauth2Config.ClientID)
	if session != nil && session.IDToken != "" {
		params.Set("id_token_hint", session.IDToken)
	}

	var logoutURL string
	if h.endSessionEndpoint != "" {
		logoutURL = h.endSessionEndpoint + "?" + params.Encode()
	} else {
		// Fallback: derive logout URL from auth endpoint (e.g. Keycloak)
		authURL := h.provider.Endpoint().AuthURL
		if len(authURL) > 4 {
			logoutURL = authURL[:len(authURL)-4] + "logout?" + params.Encode()
		} else {
			logoutURL = postLogoutRedirect
		}
	}

	http.Redirect(w, r, logoutURL, http.StatusTemporaryRedirect)
}

// GetSession writes the current session info as JSON to the response
func (h *OAuthHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.GetSessionInfo(r))
}

// GetSessionInfo returns session data as a map without writing to the response
func (h *OAuthHandler) GetSessionInfo(r *http.Request) map[string]any {
	if h.disabled {
		return map[string]any{
			"authenticated": true,
			"oidc_enabled":  true,
			"user": map[string]string{
				"sub":   "dev-user",
				"email": "dev@example.com",
				"name":  "Development User",
			},
		}
	}

	session, err := h.getSessionFromCookie(r)
	if err != nil || session == nil {
		return map[string]any{
			"authenticated": false,
			"oidc_enabled":  true,
		}
	}

	return map[string]any{
		"authenticated": true,
		"oidc_enabled":  true,
		"user": map[string]string{
			"sub":   session.Subject,
			"email": session.Email,
			"name":  session.Name,
		},
		"expires_at": session.ExpiresAt,
	}
}

// GetAccessToken retrieves the access token from the server-side session
func (h *OAuthHandler) GetAccessToken(r *http.Request) string {
	session, err := h.getSessionFromCookie(r)
	if err != nil || session == nil {
		return ""
	}
	return session.AccessToken
}

// GetIDToken retrieves the ID token from the server-side session
func (h *OAuthHandler) GetIDToken(r *http.Request) string {
	session, err := h.getSessionFromCookie(r)
	if err != nil || session == nil {
		return ""
	}
	return session.IDToken
}

func oidcSessionExpiry(now, accessExpiry, idExpiry time.Time) time.Time {
	expiry := now.Add(cookieMaxAge)
	for _, candidate := range []time.Time{accessExpiry, idExpiry} {
		if !candidate.IsZero() && candidate.Before(expiry) {
			expiry = candidate
		}
	}
	return expiry
}

func sessionIDHash(id string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil || len(raw) != 32 || base64.RawURLEncoding.EncodeToString(raw) != id {
		return "", errors.New("invalid OIDC session ID")
	}
	digest := sha256.Sum256([]byte(id))
	return hex.EncodeToString(digest[:]), nil
}

func (h *OAuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, session *Session) error {
	if h.sessions == nil {
		return errors.New("OIDC session store is required")
	}
	if !time.Now().Before(session.ExpiresAt) {
		return errors.New("OIDC session is expired")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := sessionIDHash(id)
	if err != nil {
		return err
	}
	if err := h.sessions.Create(r.Context(), hash, session); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: id, Path: "/",
		MaxAge: max(1, int(time.Until(session.ExpiresAt).Seconds())), Expires: session.ExpiresAt,
		HttpOnly: true, Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (h *OAuthHandler) getSessionFromCookie(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	hash, err := sessionIDHash(cookie.Value)
	if err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("OIDC session store is required")
	}
	session, err := h.sessions.Get(r.Context(), hash)
	if err != nil {
		slog.Error("failed to load OIDC session", "error", err)
		return nil, err
	}
	if session == nil || !time.Now().Before(session.ExpiresAt) {
		return nil, errors.New("OIDC session missing or expired")
	}
	return session, nil
}

// ClearSession revokes the server-side session and clears the browser cookie.
func (h *OAuthHandler) ClearSession(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: isSecureRequest(r), SameSite: http.SameSiteLaxMode,
	})
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	hash, err := sessionIDHash(cookie.Value)
	if err != nil {
		return nil
	} // Legacy or malformed cookies have no stored session.
	if h.sessions == nil {
		return errors.New("OIDC session store is required")
	}
	if err := h.sessions.Delete(r.Context(), hash); err != nil {
		return fmt.Errorf("deleting OIDC session: %w", err)
	}
	return nil
}

// isSecureRequest checks if the request originated over HTTPS,
// accounting for reverse proxies that set X-Forwarded-Proto.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}
