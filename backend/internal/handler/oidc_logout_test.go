package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/lmmendes/attic/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

type logoutSessionStore struct {
	deleted string
	err     error
}

func (s *logoutSessionStore) Create(context.Context, string, *auth.Session) error { return nil }
func (s *logoutSessionStore) Get(context.Context, string) (*auth.Session, error)  { return nil, nil }
func (s *logoutSessionStore) Delete(_ context.Context, hash string) error {
	s.deleted = hash
	return s.err
}
func (s *logoutSessionStore) DeleteExpired(context.Context) error { return nil }

func Test_AuthHandler_OIDCLogout(t *testing.T) {
	var issuer string
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	}))
	defer discovery.Close()
	issuer = discovery.URL
	for _, failing := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "database_failure"}[failing], func(t *testing.T) {
			store := &logoutSessionStore{}
			if failing {
				store.err = errors.New("database unavailable")
			}
			oauth, err := auth.NewOAuthHandler(context.Background(), auth.OAuthConfig{IssuerURL: issuer, ClientID: "attic", SessionStore: store})
			if err != nil {
				t.Fatal(err)
			}
			h := &AuthHandler{oidcEnabled: true, oauthHandler: oauth}
			id := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
			hash := sha256.Sum256([]byte(id))
			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session", Value: id})
			rec := httptest.NewRecorder()
			h.Logout(rec, req)
			want := http.StatusOK
			if failing {
				want = http.StatusInternalServerError
			}
			if rec.Code != want || store.deleted != hex.EncodeToString(hash[:]) {
				t.Fatalf("logout status %d, deleted %q", rec.Code, store.deleted)
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].MaxAge != -1 {
				t.Fatal("OIDC cookie not cleared")
			}
			if !failing {
				var response map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatal(err)
				}
				if response["success"] != true {
					t.Fatal("success response changed")
				}
			}
		})
	}
}
