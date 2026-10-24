package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type testKeyStore struct{ key *domain.APIKey }

func (s *testKeyStore) Authenticate(_ context.Context, secret string) (*domain.APIKey, error) {
	if secret != "valid" {
		return nil, nil
	}
	return s.key, nil
}

type testKeyUsers struct{ user *domain.User }

func (s *testKeyUsers) GetByID(context.Context, uuid.UUID) (*domain.User, error) { return s.user, nil }
func TestAPIKeyAuthenticationPrecedence(t *testing.T) {
	user := &domain.User{ID: uuid.New(), OrganizationID: uuid.New(), Role: domain.UserRoleAdmin}
	keys := &testKeyStore{&domain.APIKey{OwnerID: user.ID, OrganizationID: user.OrganizationID}}
	users := &testKeyUsers{user}
	a := APIKeyAuthenticator{keys, users, user.OrganizationID}
	disabled, err := NewMiddleware(context.Background(), Config{Disabled: true, DisabledUser: user})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUser(r.Context()) == nil {
			t.Fatal("missing domain user")
		}
		w.WriteHeader(204)
	})
	for _, header := range []string{"Bearer valid", "Bearer invalid", "Basic value", "", "Bearer"} {
		request := httptest.NewRequest("GET", "/", nil)
		request.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		a.Authenticate(disabled.Authenticate)(endpoint).ServeHTTP(response, request)
		want := 401
		if header == "Bearer valid" {
			want = 204
		}
		if response.Code != want {
			t.Fatalf("%q: got %d want %d", header, response.Code, want)
		}
	}
	request := httptest.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()
	a.Authenticate(disabled.Authenticate)(endpoint).ServeHTTP(response, request)
	if response.Code != 204 {
		t.Fatal("disabled auth changed")
	}
	request.Header.Set("Authorization", "Bearer valid")
	for _, badUser := range []*domain.User{nil, {ID: user.ID, OrganizationID: uuid.New()}} {
		users.user = badUser
		response = httptest.NewRecorder()
		a.Authenticate(disabled.Authenticate)(endpoint).ServeHTTP(response, request)
		if response.Code != 401 {
			t.Fatal("deleted or foreign user accepted")
		}
	}
}

func TestAPIKeyBypassesLocalAndOIDCProvisioning(t *testing.T) {
	user := &domain.User{ID: uuid.New(), OrganizationID: uuid.New(), Role: domain.UserRoleUser}
	key := &domain.APIKey{OwnerID: user.ID, OrganizationID: user.OrganizationID}
	authenticator := APIKeyAuthenticator{Keys: &testKeyStore{key}, Users: &testKeyUsers{user}, OrganizationID: user.OrganizationID}
	provisioner := NewUserProvisioner(nil, user.OrganizationID)
	for _, oidc := range []bool{false, true} {
		browser := &Middleware{oidcEnabled: oidc}
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUser(r.Context()) != user || GetClaims(r.Context()) != nil {
				t.Fatal("API key used browser claims")
			}
			w.WriteHeader(204)
		})
		var next http.Handler = provisioner.LoadLocalUser(endpoint)
		if oidc {
			next = provisioner.Provision(endpoint)
		}
		request := httptest.NewRequest("GET", "/api/assets", nil)
		request.Header.Set("Authorization", "Bearer valid")
		response := httptest.NewRecorder()
		authenticator.Authenticate(browser.Authenticate)(next).ServeHTTP(response, request)
		if response.Code != 204 {
			t.Fatalf("oidc=%v got %d", oidc, response.Code)
		}
	}
}
