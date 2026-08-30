package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	twitchTokenURL          = "https://id.twitch.tv/oauth2/token"
	clientIDEnvVar          = "ATTIC_IGDB_CLIENT_ID"
	clientSecretEnvVar      = "ATTIC_IGDB_CLIENT_SECRET"
	tokenRefreshLeeway      = 5 * time.Minute
	defaultTokenHTTPTimeout = 10 * time.Second
)

// ClientID and ClientSecret can be set at build time via ldflags:
// go build -ldflags="-X github.com/lmmendes/attic/internal/plugin/igdb.ClientID=... -X github.com/lmmendes/attic/internal/plugin/igdb.ClientSecret=..."
var (
	ClientID     = ""
	ClientSecret = ""
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// tokenSource manages the Twitch OAuth token used to authenticate IGDB requests.
// Tokens are cached in memory and refreshed automatically before expiry.
type tokenSource struct {
	httpClient *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newTokenSource() *tokenSource {
	return &tokenSource{
		httpClient: &http.Client{Timeout: defaultTokenHTTPTimeout},
	}
}

// Token returns a valid access token, fetching a new one when necessary.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token != "" && time.Until(t.expiresAt) > tokenRefreshLeeway {
		return t.token, nil
	}

	clientID, clientSecret := credentials()
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("IGDB credentials not configured")
	}

	tok, err := t.requestToken(ctx, clientID, clientSecret)
	if err != nil {
		return "", err
	}

	t.token = tok.AccessToken
	t.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return t.token, nil
}

// Invalidate forces the next Token() call to fetch a fresh token.
// Use this after receiving a 401 from the IGDB API.
func (t *tokenSource) Invalidate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.token = ""
	t.expiresAt = time.Time{}
}

// IsEnabled returns true if both IGDB credentials are configured.
func IsEnabled() bool {
	id, secret := credentials()
	return id != "" && secret != ""
}

// GetDisabledReason returns the reason the plugin is disabled.
func GetDisabledReason() string {
	if IsEnabled() {
		return ""
	}
	id, secret := credentials()
	missing := make([]string, 0, 2)
	if id == "" {
		missing = append(missing, clientIDEnvVar)
	}
	if secret == "" {
		missing = append(missing, clientSecretEnvVar)
	}
	return "Missing IGDB credentials: " + strings.Join(missing, ", ")
}

// Private helpers

func credentials() (string, string) {
	id := os.Getenv(clientIDEnvVar)
	if id == "" {
		id = ClientID
	}
	secret := os.Getenv(clientSecretEnvVar)
	if secret == "" {
		secret = ClientSecret
	}
	return id, secret
}

func (t *tokenSource) requestToken(ctx context.Context, clientID, clientSecret string) (*tokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, twitchTokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned empty token")
	}

	return &tok, nil
}
