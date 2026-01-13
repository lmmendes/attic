package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/mendelui/attic/internal/domain"
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
	verifier       *oidc.IDTokenVerifier
	disabled       bool
	oidcEnabled    bool
	oauth          *OAuthHandler
	sessionManager *SessionManager
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
	var tokenString string

	// First, try Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			tokenString = parts[1]
		}
	}

	// If no header, try session cookie
	if tokenString == "" && m.oauth != nil {
		tokenString = m.oauth.GetAccessToken(r)
	}

	if tokenString == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Verify the token
	idToken, err := m.verifier.Verify(r.Context(), tokenString)
	if err != nil {
		slog.Error("token verification failed", "error", err)
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}

	// Extract claims
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("failed to parse claims", "error", err)
		http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
		return
	}

	claims.Subject = idToken.Subject

	// Add claims to context
	ctx := context.WithValue(r.Context(), UserContextKey, &claims)
	next.ServeHTTP(w, r.WithContext(ctx))
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
