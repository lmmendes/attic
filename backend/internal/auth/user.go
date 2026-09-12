package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
)

type userContextKey string

const (
	DomainUserContextKey userContextKey = "domain_user"
)

// UserProvisioner handles automatic user creation
type UserProvisioner struct {
	userRepo *repository.UserRepository
	orgID    uuid.UUID
}

// NewUserProvisioner creates a new user provisioner
func NewUserProvisioner(userRepo *repository.UserRepository, orgID uuid.UUID) *UserProvisioner {
	return &UserProvisioner{
		userRepo: userRepo,
		orgID:    orgID,
	}
}

// Provision is middleware that ensures a domain user exists for the authenticated user
func (p *UserProvisioner) Provision(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			// No authenticated user, continue without provisioning
			next.ServeHTTP(w, r)
			return
		}

		// Get or create the domain user
		user, created, err := p.userRepo.GetOrCreate(
			r.Context(),
			p.orgID,
			claims.Subject,
			claims.Email,
			claimsDisplayName(claims),
		)
		if err != nil {
			slog.Error("failed to provision user", "error", err, "subject", claims.Subject)
			http.Error(w, `{"error":"failed to provision user"}`, http.StatusInternalServerError)
			return
		}

		if created {
			slog.Info("provisioned new user", "user_id", user.ID, "email", user.Email)
		}

		// Add domain user to context
		ctx := context.WithValue(r.Context(), DomainUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoadLocalUser loads the existing account identified by an authenticated local
// session. Unlike OIDC provisioning, this must never create an account.
func (p *UserProvisioner) LoadLocalUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		id, err := uuid.Parse(claims.Subject)
		if err != nil {
			http.Error(w, `{"error":"invalid session"}`, http.StatusUnauthorized)
			return
		}
		user, err := p.userRepo.GetByID(r.Context(), id)
		if err != nil {
			slog.Error("failed to load local user", "error", err, "user_id", id)
			http.Error(w, `{"error":"failed to load user"}`, http.StatusInternalServerError)
			return
		}
		if user == nil || user.OrganizationID != p.orgID {
			http.Error(w, `{"error":"invalid session"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), DomainUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser extracts the domain user from context
func GetUser(ctx context.Context) *domain.User {
	user, ok := ctx.Value(DomainUserContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

func claimsDisplayName(c *Claims) string {
	if c.Name != "" {
		return c.Name
	}
	return c.DisplayName
}
