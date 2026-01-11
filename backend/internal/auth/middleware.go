package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
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

// Middleware handles JWT validation
type Middleware struct {
	verifier *oidc.IDTokenVerifier
	disabled bool
}

// Config for auth middleware
type Config struct {
	IssuerURL string
	ClientID  string
	Disabled  bool // For development without auth
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(ctx context.Context, cfg Config) (*Middleware, error) {
	if cfg.Disabled {
		slog.Warn("authentication is DISABLED - all requests will use a mock user")
		return &Middleware{disabled: true}, nil
	}

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

	return &Middleware{verifier: verifier}, nil
}

// Authenticate is HTTP middleware that validates JWT tokens
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

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

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
	})
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
