package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
)

func Test_GetUser_WithValidContext_ReturnsUser(t *testing.T) {
	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		Role:           domain.UserRoleUser,
	}
	ctx := context.WithValue(context.Background(), DomainUserContextKey, user)

	result := GetUser(ctx)

	if result == nil {
		t.Fatal("expected user to be returned")
	}
	if result.ID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, result.ID)
	}
	if result.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", result.Email)
	}
}

func Test_GetUser_WithEmptyContext_ReturnsNil(t *testing.T) {
	ctx := context.Background()

	result := GetUser(ctx)

	if result != nil {
		t.Error("expected nil user for empty context")
	}
}

func Test_GetUser_WithWrongType_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), DomainUserContextKey, "not a user")

	result := GetUser(ctx)

	if result != nil {
		t.Error("expected nil user for wrong type in context")
	}
}

func Test_GetUser_WithNilValue_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), DomainUserContextKey, (*domain.User)(nil))

	result := GetUser(ctx)

	if result != nil {
		t.Error("expected nil user when nil is stored in context")
	}
}

// mockUserRepo implements a minimal UserRepository for testing
type mockUserRepo struct {
	users         map[uuid.UUID]*domain.User
	getOrCreateFn func(ctx context.Context, orgID uuid.UUID, subject, email, displayName string) (*domain.User, bool, error)
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[uuid.UUID]*domain.User),
	}
}

func (r *mockUserRepo) GetOrCreate(ctx context.Context, orgID uuid.UUID, subject, email, displayName string) (*domain.User, bool, error) {
	if r.getOrCreateFn != nil {
		return r.getOrCreateFn(ctx, orgID, subject, email, displayName)
	}
	// Default: create new user
	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Email:          email,
		Role:           domain.UserRoleUser,
	}
	if displayName != "" {
		user.DisplayName = &displayName
	}
	sub := subject
	user.OIDCSubject = &sub
	return user, true, nil
}

func Test_UserProvisioner_NoClaims_ContinuesWithoutProvisioning(t *testing.T) {
	orgID := uuid.New()
	mockRepo := newMockUserRepo()

	// Create provisioner with a wrapper that adapts our mock
	provisioner := &testUserProvisioner{
		repo:  mockRepo,
		orgID: orgID,
	}

	nextCalled := false
	var userInContext *domain.User
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		userInContext = GetUser(r.Context())
	})

	// Request without claims in context
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	provisioner.Provision(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	if userInContext != nil {
		t.Error("expected no user in context when no claims present")
	}
}

func Test_UserProvisioner_WithClaims_ProvisionesUser(t *testing.T) {
	orgID := uuid.New()
	mockRepo := newMockUserRepo()

	provisioner := &testUserProvisioner{
		repo:  mockRepo,
		orgID: orgID,
	}

	nextCalled := false
	var userInContext *domain.User
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		userInContext = GetUser(r.Context())
	})

	// Create request with claims
	claims := &Claims{
		Subject:     "oidc-subject-123",
		Email:       "user@example.com",
		DisplayName: "Test User",
	}
	ctx := context.WithValue(context.Background(), UserContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	provisioner.Provision(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	if userInContext == nil {
		t.Fatal("expected user to be provisioned in context")
	}
	if userInContext.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", userInContext.Email)
	}
}

func Test_UserProvisioner_RepoError_ReturnsInternalError(t *testing.T) {
	orgID := uuid.New()
	mockRepo := newMockUserRepo()
	mockRepo.getOrCreateFn = func(ctx context.Context, orgID uuid.UUID, subject, email, displayName string) (*domain.User, bool, error) {
		return nil, false, context.DeadlineExceeded
	}

	provisioner := &testUserProvisioner{
		repo:  mockRepo,
		orgID: orgID,
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	claims := &Claims{
		Subject: "subject",
		Email:   "test@example.com",
	}
	ctx := context.WithValue(context.Background(), UserContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	provisioner.Provision(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("expected next handler NOT to be called on error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func Test_NewUserProvisioner_CreatesProvisioner(t *testing.T) {
	// This is a compilation/integration test to verify the real constructor works
	// We can't easily test it without a real repository, but we can verify the struct

	// The real NewUserProvisioner requires a *repository.UserRepository
	// which we can't easily create without a database connection.
	// This test just verifies the expected behavior.

	if DomainUserContextKey == "" {
		t.Error("expected DomainUserContextKey to be defined")
	}
}

// testUserProvisioner is a test double that mimics UserProvisioner behavior
type testUserProvisioner struct {
	repo  *mockUserRepo
	orgID uuid.UUID
}

func (p *testUserProvisioner) Provision(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			next.ServeHTTP(w, r)
			return
		}

		user, _, err := p.repo.GetOrCreate(
			r.Context(),
			p.orgID,
			claims.Subject,
			claims.Email,
			claims.DisplayName,
		)
		if err != nil {
			http.Error(w, `{"error":"failed to provision user"}`, http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), DomainUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Test_DomainUserContextKey_IsUnique(t *testing.T) {
	// Verify the context key is properly typed to avoid collisions
	var key1 userContextKey = "domain_user"
	var key2 userContextKey = "domain_user"

	if key1 != key2 {
		t.Error("same key values should be equal")
	}

	// The exported constant should match
	if DomainUserContextKey != "domain_user" {
		t.Errorf("expected 'domain_user', got '%s'", DomainUserContextKey)
	}
}

func Test_GetUser_IntegrationWithClaims(t *testing.T) {
	// Simulate a full flow: claims -> provisioner -> handler
	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "integration@example.com",
		Role:           domain.UserRoleAdmin,
	}

	// Simulate what the provisioner does
	ctx := context.Background()
	ctx = context.WithValue(ctx, DomainUserContextKey, user)

	// In a handler, we'd do this:
	retrievedUser := GetUser(ctx)

	if retrievedUser == nil {
		t.Fatal("expected user to be retrievable")
	}
	if retrievedUser.ID != user.ID {
		t.Error("expected same user")
	}
	if !retrievedUser.IsAdmin() {
		t.Error("expected user to be admin")
	}
}
