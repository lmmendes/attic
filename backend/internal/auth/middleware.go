package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/lmmendes/attic/internal/domain"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// Claims represents the JWT claims we care about
type Claims struct {
	Subject     string `json:"sub"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"preferred_username"`
}

// Middleware handles authentication (both OIDC and local)
type Middleware struct {
	verifier        *oidc.IDTokenVerifier
	idTokenVerifier *oidc.IDTokenVerifier
	disabled        bool
	oidcEnabled     bool
	oauth           *OAuthHandler
	sessionManager  *SessionManager
}

// Config for auth middleware
type Config struct {
	IssuerURL   string
	ClientID    string
	Disabled    bool // For development without auth
	OIDCEnabled bool // Whether OIDC is the auth method
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(ctx context.Context, cfg Config) (*Middleware, error) {
	if cfg.Disabled {
		slog.Warn("authentication is DISABLED - all requests will use a mock user")
		return &Middleware{disabled: true}, nil
	}

	m := &Middleware{
		oidcEnabled: cfg.OIDCEnabled,
	}

	// Only initialize OIDC if enabled
	if cfg.OIDCEnabled {
		provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
		if err != nil {
			return nil, err
		}

		verifier := provider.Verifier(&oidc.Config{
			ClientID:                   cfg.ClientID,
			SkipClientIDCheck:          true, // Keycloak access tokens use 'azp' not 'aud'
			SkipExpiryCheck:            false,
			SkipIssuerCheck:            false,
			InsecureSkipSignatureCheck: false,
		})
		m.verifier = verifier

		// Separate verifier for ID tokens from session cookies.
		// Expiry is skipped because the session manages its own lifetime
		// via ExpiresAt (derived from the access token expiry).
		idTokenVerifier := provider.Verifier(&oidc.Config{
			ClientID:        cfg.ClientID,
			SkipExpiryCheck: true,
		})
		m.idTokenVerifier = idTokenVerifier
	}

	return m, nil
}

// SetOAuthHandler sets the OAuth handler for cookie-based auth (OIDC mode)
func (m *Middleware) SetOAuthHandler(oauth *OAuthHandler) {
	m.oauth = oauth
}

// SetSessionManager sets the session manager for local auth
func (m *Middleware) SetSessionManager(sm *SessionManager) {
	m.sessionManager = sm
}

// Authenticate is HTTP middleware that validates authentication
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.disabled {
			// Use mock user for development
			claims := &Claims{
				Subject:     "dev-user",
				Email:       "dev@example.com",
				Name:        "Development User",
				DisplayName: "devuser",
			}
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if m.oidcEnabled {
			// OIDC authentication
			m.authenticateOIDC(w, r, next)
		} else {
			// Local (email/password) authentication
			m.authenticateLocal(w, r, next)
		}
	})
}

// authenticateOIDC handles OIDC-based authentication
func (m *Middleware) authenticateOIDC(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// First, try Authorization header (Bearer token from API clients)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			m.authenticateOIDCBearer(w, r, next, parts[1])
			return
		}
	}

	// Fall back to session cookie (browser-based flow)
	if m.oauth != nil {
		m.authenticateOIDCSession(w, r, next)
		return
	}

	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
}

// authenticateOIDCBearer verifies a Bearer token from the Authorization header.
// The token is expected to be a JWT (e.g. from providers that issue JWT access tokens).
func (m *Middleware) authenticateOIDCBearer(w http.ResponseWriter, r *http.Request, next http.Handler, tokenString string) {
	idToken, err := m.verifier.Verify(r.Context(), tokenString)
	if err != nil {
		slog.Error("token verification failed", "error", err)
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("failed to parse claims", "error", err)
		http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
		return
	}

	claims.Subject = idToken.Subject

	ctx := context.WithValue(r.Context(), UserContextKey, &claims)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// authenticateOIDCSession verifies the session cookie using the stored ID token.
// RFC-compliant OIDC providers may issue opaque access tokens, so the ID token
// (which is always a signed JWT) is used for verification instead.
func (m *Middleware) authenticateOIDCSession(w http.ResponseWriter, r *http.Request, next http.Handler) {
	session, err := m.oauth.getSessionFromCookie(r)
	if err != nil || session == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Check session expiry (derived from the access token lifetime)
	if time.Now().After(session.ExpiresAt) {
		http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
		return
	}

	// Use the ID token for verification — it is always a signed JWT per OIDC spec.
	// This avoids the "compact JWS format must have three parts" error that occurs
	// when providers (e.g. Authelia, Keycloak) issue opaque access tokens.
	if session.IDToken != "" && m.idTokenVerifier != nil {
		idToken, err := m.idTokenVerifier.Verify(r.Context(), session.IDToken)
		if err != nil {
			slog.Error("id token verification failed", "error", err)
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		var claims Claims
		if err := idToken.Claims(&claims); err != nil {
			slog.Error("failed to parse claims", "error", err)
			http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		claims.Subject = idToken.Subject

		// Supplement missing claims from session data
		if claims.Email == "" {
			claims.Email = session.Email
		}
		if claims.DisplayName == "" {
			claims.DisplayName = session.Name
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &claims)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	// Fallback: try verifying the access token as a JWT (backward compat
	// for providers that issue JWT access tokens and sessions without an ID token)
	if session.AccessToken != "" {
		idToken, err := m.verifier.Verify(r.Context(), session.AccessToken)
		if err != nil {
			slog.Error("token verification failed", "error", err)
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		var claims Claims
		if err := idToken.Claims(&claims); err != nil {
			slog.Error("failed to parse claims", "error", err)
			http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		claims.Subject = idToken.Subject

		if claims.Email == "" {
			claims.Email = session.Email
		}
		if claims.DisplayName == "" {
			claims.DisplayName = session.Name
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &claims)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
}

// authenticateLocal handles local (email/password) authentication
func (m *Middleware) authenticateLocal(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if m.sessionManager == nil {
		http.Error(w, `{"error":"session manager not configured"}`, http.StatusInternalServerError)
		return
	}

	session, err := m.sessionManager.GetSession(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Convert local session to claims for compatibility
	claims := &Claims{
		Subject:     session.UserID.String(),
		Email:       session.Email,
		Name:        session.Name,
		DisplayName: session.Name,
	}

	// Add claims to context
	ctx := context.WithValue(r.Context(), UserContextKey, claims)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// GetClaims extracts claims from context
func GetClaims(ctx context.Context) *Claims {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// Optional returns middleware that allows unauthenticated requests
func (m *Middleware) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// If header is present, validate it
		m.Authenticate(next).ServeHTTP(w, r)
	})
}

// RequireAdmin middleware checks if the user has admin role
func RequireAdmin(sessionManager *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check domain user from context first (used by OIDC via UserProvisioner)
			if user := GetUser(r.Context()); user != nil {
				if user.Role != domain.UserRoleAdmin {
					http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Fall back to local session
			session, err := sessionManager.GetSession(r)
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
}
