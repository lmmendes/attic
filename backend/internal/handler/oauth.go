package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

const (
	mobileClientID       = "attic-mobile"
	mobileRedirectURI    = "com.lmmendes.attic:/oauth2redirect"
	accessTokenLifetime  = 15 * time.Minute
	refreshTokenLifetime = 30 * 24 * time.Hour
	authorizationCodeTTL = 2 * time.Minute
)

type OAuthProtocolHandler struct {
	baseURL          string
	oidcEnabled      bool
	defaultOrgID     uuid.UUID
	users            oauthUserRepository
	tokens           *repository.OAuthRepository
	sessions         *auth.SessionManager
	oauth            *auth.OAuthHandler
	passwordAttempts *passwordAttemptLimiter
}

type oauthUserRepository interface {
	GetByID(context.Context, uuid.UUID) (*domain.User, error)
	GetByEmail(context.Context, string) (*domain.User, error)
	GetOrCreate(context.Context, uuid.UUID, string, string, string) (*domain.User, bool, error)
}

func NewOAuthProtocolHandler(baseURL string, oidcEnabled bool, defaultOrgID uuid.UUID, users oauthUserRepository, tokens *repository.OAuthRepository, sessions *auth.SessionManager, oauth *auth.OAuthHandler) *OAuthProtocolHandler {
	return &OAuthProtocolHandler{
		baseURL: strings.TrimRight(baseURL, "/"), oidcEnabled: oidcEnabled, defaultOrgID: defaultOrgID,
		users: users, tokens: tokens, sessions: sessions, oauth: oauth, passwordAttempts: newPasswordAttemptLimiter(),
	}
}

func (h *OAuthProtocolHandler) Metadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                h.baseURL,
		"authorization_endpoint":                h.baseURL + "/oauth/authorize",
		"token_endpoint":                        h.baseURL + "/oauth/token",
		"revocation_endpoint":                   h.baseURL + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

func (h *OAuthProtocolHandler) AuthMethods(w http.ResponseWriter, _ *http.Request) {
	methods := []string{"password"}
	if h.oidcEnabled {
		methods = []string{"oidc"}
	}
	response := map[string]any{
		"issuer":                 h.baseURL,
		"authorization_endpoint": h.baseURL + "/oauth/authorize",
		"token_endpoint":         h.baseURL + "/oauth/token",
		"client_id":              mobileClientID,
		"redirect_uri":           mobileRedirectURI,
		"methods":                methods,
	}
	if !h.oidcEnabled {
		response["password_token_endpoint"] = h.baseURL + "/oauth/password"
	}
	writeJSON(w, http.StatusOK, response)
}

type passwordTokenRequest struct {
	ClientID string `json:"client_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PasswordToken authenticates Attic's built-in first-party native client.
// It is deliberately separate from /oauth/token: OAuth's password grant stays
// disabled, while OIDC and passkeys continue to use Authorization Code + PKCE.
func (h *OAuthProtocolHandler) PasswordToken(w http.ResponseWriter, r *http.Request) {
	if h.oidcEnabled {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_authentication_method", "password authentication is disabled")
		return
	}
	var request passwordTokenRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&request); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if request.ClientID != mobileClientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "unknown client")
		return
	}
	if strings.TrimSpace(request.Email) == "" || request.Password == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "email and password are required")
		return
	}
	now := time.Now().UTC()
	if !h.passwordAttempts.allow(r, request.Email, now) {
		w.Header().Set("Retry-After", "900")
		writeOAuthError(w, http.StatusTooManyRequests, "temporarily_unavailable", "too many failed sign-in attempts; try again later")
		return
	}
	user, err := h.users.GetByEmail(r.Context(), request.Email)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to authenticate")
		return
	}
	var passwordHash *string
	if user != nil {
		passwordHash = user.PasswordHash
	}
	if user == nil || !auth.CheckPasswordHash(request.Password, passwordHash) {
		h.passwordAttempts.failure(r, request.Email, now)
		writeOAuthError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	h.passwordAttempts.success(r, request.Email)
	h.issueTokens(w, r, user.ID, mobileClientID, "attic offline_access")
}

func (h *OAuthProtocolHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	request, err := validateAuthorizationRequest(r.URL.Query())
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.currentUser(r)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to read login session")
		return
	}
	if user == nil {
		returnTo := r.URL.RequestURI()
		http.Redirect(w, r, "/login?return_to="+url.QueryEscape(returnTo), http.StatusFound)
		return
	}

	code, err := randomToken("attic_code_")
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue authorization code")
		return
	}
	if err := h.tokens.CreateAuthorizationCode(r.Context(), code, repository.AuthorizationCode{
		UserID: user.ID, ClientID: request.clientID, RedirectURI: request.redirectURI,
		CodeChallenge: request.codeChallenge, Scope: request.scope,
	}, time.Now().UTC().Add(authorizationCodeTTL)); err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue authorization code")
		return
	}

	redirect, _ := url.Parse(request.redirectURI)
	query := redirect.Query()
	query.Set("code", code)
	if request.state != "" {
		query.Set("state", request.state)
	}
	redirect.RawQuery = query.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (h *OAuthProtocolHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		h.exchangeAuthorizationCode(w, r)
	case "refresh_token":
		h.refresh(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant type is not supported")
	}
}

func (h *OAuthProtocolHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err == nil {
		_ = h.tokens.RevokeRefreshToken(r.Context(), r.Form.Get("token"))
	}
	w.WriteHeader(http.StatusOK)
}

type authorizationRequest struct {
	clientID      string
	redirectURI   string
	codeChallenge string
	scope         string
	state         string
}

func validateAuthorizationRequest(values url.Values) (*authorizationRequest, error) {
	if values.Get("response_type") != "code" {
		return nil, fmt.Errorf("response_type must be code")
	}
	if values.Get("client_id") != mobileClientID {
		return nil, fmt.Errorf("unknown client_id")
	}
	if values.Get("redirect_uri") != mobileRedirectURI {
		return nil, fmt.Errorf("redirect_uri is not registered")
	}
	if values.Get("code_challenge_method") != "S256" {
		return nil, fmt.Errorf("code_challenge_method must be S256")
	}
	challenge := values.Get("code_challenge")
	decodedChallenge, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(decodedChallenge) != sha256.Size {
		return nil, fmt.Errorf("invalid code_challenge")
	}
	if values.Get("state") == "" {
		return nil, fmt.Errorf("state is required")
	}
	return &authorizationRequest{
		clientID: mobileClientID, redirectURI: mobileRedirectURI, codeChallenge: challenge,
		scope: values.Get("scope"), state: values.Get("state"),
	}, nil
}

func (h *OAuthProtocolHandler) currentUser(r *http.Request) (*domain.User, error) {
	if h.oidcEnabled {
		if h.oauth == nil {
			return nil, nil
		}
		session, err := h.oauth.GetSessionData(r)
		if err != nil || session == nil || time.Now().UTC().After(session.ExpiresAt) {
			return nil, nil
		}
		user, _, err := h.users.GetOrCreate(r.Context(), h.defaultOrgID, session.Subject, session.Email, session.Name)
		return user, err
	}
	session, err := h.sessions.GetSession(r)
	if err != nil || session == nil {
		return nil, nil
	}
	return h.users.GetByID(r.Context(), session.UserID)
}

func (h *OAuthProtocolHandler) exchangeAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code, err := h.tokens.ConsumeAuthorizationCode(r.Context(), r.Form.Get("code"))
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to exchange code")
		return
	}
	if code == nil || code.ClientID != r.Form.Get("client_id") || code.RedirectURI != r.Form.Get("redirect_uri") || !verifyPKCE(r.Form.Get("code_verifier"), code.CodeChallenge) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid")
		return
	}
	h.issueTokens(w, r, code.UserID, code.ClientID, code.Scope)
}

func (h *OAuthProtocolHandler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Form.Get("client_id") != mobileClientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "unknown client")
		return
	}
	accessToken, err := randomToken("attic_at_")
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to refresh token")
		return
	}
	refreshToken, err := randomToken("attic_rt_")
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to refresh token")
		return
	}
	accessExpiresAt := time.Now().UTC().Add(accessTokenLifetime)
	session, err := h.tokens.RotateRefreshToken(r.Context(), r.Form.Get("refresh_token"), accessToken, refreshToken, accessExpiresAt)
	if err != nil || session == nil || session.ClientID != mobileClientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid")
		return
	}
	writeTokenResponse(w, accessToken, refreshToken, session.Scope)
}

func (h *OAuthProtocolHandler) issueTokens(w http.ResponseWriter, r *http.Request, userID uuid.UUID, clientID, scope string) {
	accessToken, err := randomToken("attic_at_")
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	refreshToken, err := randomToken("attic_rt_")
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	now := time.Now().UTC()
	if err := h.tokens.CreateSession(r.Context(), userID, clientID, scope, accessToken, refreshToken, now.Add(accessTokenLifetime), now.Add(refreshTokenLifetime)); err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}
	writeTokenResponse(w, accessToken, refreshToken, scope)
}

func writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken, scope string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken, "token_type": "Bearer",
		"expires_in": int(accessTokenLifetime.Seconds()), "refresh_token": refreshToken, "scope": scope,
	})
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func randomToken(prefix string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func verifyPKCE(verifier, expectedChallenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(challenge), []byte(expectedChallenge)) == 1
}
