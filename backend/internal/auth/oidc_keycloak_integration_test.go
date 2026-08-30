package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type keycloakProvider struct {
	container    testcontainers.Container
	issuerURL    string
	clientID     string
	clientSecret string
	username     string
	password     string
}

type keycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

var (
	keycloakOnce     sync.Once
	keycloakInstance *keycloakProvider
	keycloakErr      error
)

func TestMain(m *testing.M) {
	exitCode := m.Run()

	if keycloakInstance != nil && keycloakInstance.container != nil {
		_ = keycloakInstance.container.Terminate(context.Background())
	}

	os.Exit(exitCode)
}

func Test_Middleware_OIDC_KeycloakSession_UsesIDTokenFromCookie(t *testing.T) {
	ctx := context.Background()
	provider, err := getKeycloakProvider()
	if err != nil {
		t.Fatalf("failed to start keycloak provider: %v", err)
	}

	token, err := provider.getUserToken(ctx)
	if err != nil {
		t.Fatalf("failed to get user token: %v", err)
	}

	middleware, err := NewMiddleware(ctx, Config{
		IssuerURL:   provider.issuerURL,
		ClientID:    provider.clientID,
		OIDCEnabled: true,
	})
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	middleware.SetOAuthHandler(&OAuthHandler{})

	session := Session{
		AccessToken: "definitely-not-a-valid-jwt",
		IDToken:     token.IDToken,
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
		Email:       "attic@example.com",
		Name:        "Attic User",
	}
	cookieValue, err := encodeSessionCookie(session)
	if err != nil {
		t.Fatalf("failed to encode session cookie: %v", err)
	}

	var claims *Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims = GetClaims(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: cookieValue})
	rec := httptest.NewRecorder()

	middleware.Authenticate(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body: %s", rec.Code, rec.Body.String())
	}
	if claims == nil {
		t.Fatal("expected claims in request context")
	}
	if claims.Subject == "" {
		t.Fatal("expected non-empty subject from ID token")
	}
	if claims.Email != "attic@example.com" {
		t.Fatalf("expected email attic@example.com, got %q", claims.Email)
	}
}

func getKeycloakProvider() (*keycloakProvider, error) {
	keycloakOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				keycloakErr = fmt.Errorf("keycloak container startup panic: %v", r)
			}
		}()
		keycloakInstance, keycloakErr = startKeycloak(context.Background())
	})

	return keycloakInstance, keycloakErr
}

func startKeycloak(ctx context.Context) (*keycloakProvider, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "quay.io/keycloak/keycloak:26.2.0",
			ExposedPorts: []string{"8080/tcp"},
			Env: map[string]string{
				"KC_BOOTSTRAP_ADMIN_USERNAME": "admin",
				"KC_BOOTSTRAP_ADMIN_PASSWORD": "admin",
				"KC_HTTP_ENABLED":             "true",
			},
			Cmd: []string{"start-dev", "--http-port=8080"},
			WaitingFor: wait.ForHTTP("/realms/master/.well-known/openid-configuration").
				WithPort("8080/tcp").
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("start keycloak container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("resolve keycloak host: %w", err)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("resolve keycloak port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	adminToken, err := getKeycloakAdminToken(ctx, baseURL)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	if err := bootstrapKeycloakRealm(ctx, baseURL, adminToken); err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	return &keycloakProvider{
		container:    container,
		issuerURL:    baseURL + "/realms/attic",
		clientID:     "attic-client",
		clientSecret: "attic-secret",
		username:     "attic-user",
		password:     "attic-password",
	}, nil
}

func getKeycloakAdminToken(ctx context.Context, baseURL string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", "admin")
	form.Set("password", "admin")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/realms/master/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create admin token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute admin token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("admin token request failed with status %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode admin token response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("admin token response missing access_token")
	}

	return payload.AccessToken, nil
}

func bootstrapKeycloakRealm(ctx context.Context, baseURL, adminToken string) error {
	realmPayload := map[string]any{
		"realm":   "attic",
		"enabled": true,
		"clients": []map[string]any{
			{
				"clientId":                  "attic-client",
				"name":                      "Attic",
				"enabled":                   true,
				"protocol":                  "openid-connect",
				"publicClient":              false,
				"secret":                    "attic-secret",
				"directAccessGrantsEnabled": true,
				"standardFlowEnabled":       true,
				"serviceAccountsEnabled":    true,
				"redirectUris":              []string{"http://app.localtest.me:8080/auth/oidc/callback"},
				"webOrigins":                []string{"*"},
				"defaultClientScopes":       []string{"profile", "email"},
				"fullScopeAllowed":          true,
			},
		},
		"users": []map[string]any{
			{
				"username":      "attic-user",
				"enabled":       true,
				"emailVerified": true,
				"email":         "attic@example.com",
				"firstName":     "Attic",
				"lastName":      "User",
				"credentials": []map[string]any{
					{
						"type":      "password",
						"value":     "attic-password",
						"temporary": false,
					},
				},
			},
		},
	}

	body, err := json.Marshal(realmPayload)
	if err != nil {
		return fmt.Errorf("marshal realm payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/admin/realms", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create realm bootstrap request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute realm bootstrap request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("realm bootstrap failed with status %d", resp.StatusCode)
	}

	return nil
}

func (p *keycloakProvider) getUserToken(ctx context.Context) (*keycloakTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("username", p.username)
	form.Set("password", p.password)
	form.Set("scope", "openid profile email")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.issuerURL+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create user token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute user token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user token request failed with status %d", resp.StatusCode)
	}

	var payload keycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode user token response: %w", err)
	}
	if payload.IDToken == "" {
		return nil, fmt.Errorf("user token response missing id_token")
	}

	return &payload, nil
}

func encodeSessionCookie(session Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
