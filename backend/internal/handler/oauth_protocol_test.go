package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

type oauthTestUsers struct {
	user *domain.User
}

func (u oauthTestUsers) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return u.user, nil
}

func (u oauthTestUsers) GetByEmail(context.Context, string) (*domain.User, error) {
	return u.user, nil
}

func (u oauthTestUsers) GetOrCreate(context.Context, uuid.UUID, string, string, string) (*domain.User, bool, error) {
	return u.user, false, nil
}

func Test_OAuthProtocolHandler_Metadata_AdvertisesSecureCodeFlow(t *testing.T) {
	handler := &OAuthProtocolHandler{baseURL: "https://attic.example.com"}
	recorder := httptest.NewRecorder()
	handler.Metadata(recorder, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["authorization_endpoint"] != "https://attic.example.com/oauth/authorize" {
		t.Fatalf("unexpected authorization endpoint: %v", response["authorization_endpoint"])
	}
	methods := response["code_challenge_methods_supported"].([]any)
	if len(methods) != 1 || methods[0] != "S256" {
		t.Fatalf("expected only S256 PKCE, got %v", methods)
	}
}

func Test_OAuthProtocolHandler_AuthMethods_AdvertisesNativePasswordEndpoint(t *testing.T) {
	handler := &OAuthProtocolHandler{baseURL: "https://attic.example.com"}
	recorder := httptest.NewRecorder()

	handler.AuthMethods(recorder, httptest.NewRequest(http.MethodGet, "/auth/methods", nil))

	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["password_token_endpoint"] != "https://attic.example.com/oauth/password" {
		t.Fatalf("unexpected password endpoint: %v", response["password_token_endpoint"])
	}
}

func Test_OAuthProtocolHandler_PasswordToken_RejectsInvalidCredentials(t *testing.T) {
	handler := &OAuthProtocolHandler{users: oauthTestUsers{}}
	request := httptest.NewRequest(http.MethodPost, "/oauth/password", strings.NewReader(`{"client_id":"attic-mobile","email":"user@example.com","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.PasswordToken(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "invalid_credentials") {
		t.Fatalf("expected invalid_credentials, got %s", recorder.Body.String())
	}
}

func Test_ValidateAuthorizationRequest_RejectsUnregisteredRedirect(t *testing.T) {
	values := validAuthorizationValues()
	values.Set("redirect_uri", "malicious.example:/callback")

	if _, err := validateAuthorizationRequest(values); err == nil {
		t.Fatal("expected unregistered redirect URI to be rejected")
	}
}

func Test_VerifyPKCE_ValidS256Verifier_ReturnsTrue(t *testing.T) {
	verifier := strings.Repeat("a", 43)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])

	if !verifyPKCE(verifier, challenge) {
		t.Fatal("expected verifier to match challenge")
	}
	if verifyPKCE(strings.Repeat("b", 43), challenge) {
		t.Fatal("expected different verifier to be rejected")
	}
}

func Test_OAuthProtocolHandler_Token_RejectsPasswordGrant(t *testing.T) {
	handler := &OAuthProtocolHandler{}
	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("grant_type=password"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.Token(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "unsupported_grant_type") {
		t.Fatalf("expected password grant to be rejected, got %s", recorder.Body.String())
	}
}

func validAuthorizationValues() url.Values {
	digest := sha256.Sum256([]byte(strings.Repeat("v", 43)))
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {mobileClientID},
		"redirect_uri":          {mobileRedirectURI},
		"code_challenge_method": {"S256"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"state":                 {"request-state"},
	}
}
