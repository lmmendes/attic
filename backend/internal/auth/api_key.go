package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type apiKeyContextKey struct{}

func GetAPIKey(ctx context.Context) *domain.APIKey {
	key, _ := ctx.Value(apiKeyContextKey{}).(*domain.APIKey)
	return key
}

type APIKeyStore interface {
	Authenticate(context.Context, string) (*domain.APIKey, error)
}
type APIKeyUserStore interface {
	GetByID(context.Context, uuid.UUID) (*domain.User, error)
}

type APIKeyAuthenticator struct {
	Keys           APIKeyStore
	Users          APIKeyUserStore
	OrganizationID uuid.UUID
}

// Authenticate gives explicit credentials precedence over any browser session.
// With no Authorization header it delegates to the existing browser middleware.
func (a *APIKeyAuthenticator) Authenticate(browser func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fallback := browser(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.Header.Values("Authorization")) == 0 {
				fallback.ServeHTTP(w, r)
				return
			}
			values := r.Header.Values("Authorization")
			parts := strings.Fields(values[0])
			if len(values) != 1 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				rejectKey(w)
				return
			}
			key, err := a.Keys.Authenticate(r.Context(), parts[1])
			if err != nil {
				http.Error(w, `{"error":"authentication unavailable"}`, 503)
				return
			}
			if key == nil || key.OrganizationID != a.OrganizationID {
				rejectKey(w)
				return
			}
			user, err := a.Users.GetByID(r.Context(), key.OwnerID)
			if err != nil {
				http.Error(w, `{"error":"authentication unavailable"}`, 503)
				return
			}
			if user == nil || user.OrganizationID != key.OrganizationID {
				rejectKey(w)
				return
			}
			ctx := context.WithValue(r.Context(), apiKeyContextKey{}, key)
			ctx = context.WithValue(ctx, DomainUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
func rejectKey(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
}

func RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetAPIKey(r.Context()) == nil {
			rejectKey(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func InteractiveOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetAPIKey(r.Context()) != nil {
			http.Error(w, `{"error":"interactive account authentication required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
