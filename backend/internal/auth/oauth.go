package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	stateCookieName   = "oauth_state"
	sessionCookieName = "session"
	cookieMaxAge      = 24 * time.Hour
)

// Session represents a user session stored in a cookie
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Subject      string    `json:"sub"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
}

// OAuthHandler handles OAuth login flow
type OAuthHandler struct {
	provider           *oidc.Provider
	oauth2Config       oauth2.Config
	verifier           *oidc.IDTokenVerifier
	baseURL            string
	secret             []byte
	disabled           bool
	endSessionEndpoint string
}

// OAuthConfig for OAuth handler
type OAuthConfig struct {
	IssuerURL     string
	ClientID      string
	ClientSecret  string
	BaseURL       string
	SessionSecret string
	Disabled      bool
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(ctx context.Context, cfg OAuthConfig) (*OAuthHandler, error) {
	if cfg.Disabled {
		return &OAuthHandler{
			nil,
			oauth2.Config{},
			nil,
			cfg.BaseURL,
			nil,
			true,
			"",
		}, nil
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

	secret := []byte(cfg.SessionSecret)
	if len(secret) < 32 {
		// Pad secret if too short
		padded := make([]byte, 32)
		copy(padded, secret)
		secret = padded
	}

	// Extract end_session_endpoint from OIDC discovery document
	var providerClaims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&providerClaims); err != nil {
		slog.Warn("failed to extract end_session_endpoint from OIDC discovery", "error", err)
	}

	return &OAuthHandler{
		provider,
		oauth2Config,
		verifier,
		cfg.BaseURL,
		secret[:32],
		false,
		providerClaims.EndSessionEndpoint,
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
		ExpiresAt:    token.Expiry,
		Subject:      idToken.Subject,
		Email:        claims.Email,
		Name:         claims.Name,
	}

	// Store session in cookie
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

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

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

// GetSession returns the current session info
func (h *OAuthHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	if h.disabled {
		// Return mock session
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": true,
			"user": map[string]string{
				"sub":   "dev-user",
				"email": "dev@example.com",
				"name":  "Development User",
			},
		})
		return
	}

	session, err := h.getSessionFromCookie(r)
	if err != nil || session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": false,
		})
		return
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// TODO: Implement token refresh
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"user": map[string]string{
			"sub":   session.Subject,
			"email": session.Email,
			"name":  session.Name,
		},
		"expires_at": session.ExpiresAt,
	})
}

// GetAccessToken extracts access token from session cookie
func (h *OAuthHandler) GetAccessToken(r *http.Request) string {
	session, err := h.getSessionFromCookie(r)
	if err != nil || session == nil {
		return ""
	}
	return session.AccessToken
}

func (h *OAuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   int(cookieMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (h *OAuthHandler) getSessionFromCookie(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
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
