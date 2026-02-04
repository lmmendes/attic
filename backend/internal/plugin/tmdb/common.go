package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	baseURL      = "https://api.themoviedb.org/3"
	imageBaseURL = "https://image.tmdb.org/t/p"
	defaultLimit = 10
)

// APIKey can be set at build time via ldflags:
// go build -ldflags="-X github.com/lmmendes/attic/internal/plugin/tmdb.APIKey=your-key"
var APIKey = ""

const tmdbAPIKeyEnvVar = "ATTIC_TMDB_API_KEY"

// getAPIKey returns the API key, preferring environment variable over build-time value
func getAPIKey() string {
	if key := os.Getenv(tmdbAPIKeyEnvVar); key != "" {
		return key
	}
	return APIKey
}

// IsEnabled returns true if the TMDB API key is configured
func IsEnabled() bool {
	return getAPIKey() != ""
}

// GetDisabledReason returns the reason the plugin is disabled
func GetDisabledReason() string {
	if IsEnabled() {
		return ""
	}
	return "Missing API key: " + tmdbAPIKeyEnvVar
}

// Client wraps HTTP client with TMDB-specific functionality
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new TMDB API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// doRequest performs an authenticated request to the TMDB API
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	apiKey := getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	u, err := url.Parse(baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	if params == nil {
		params = url.Values{}
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// get performs a GET request and decodes the JSON response
func (c *Client) get(ctx context.Context, endpoint string, params url.Values, result any) error {
	resp, err := c.doRequest(ctx, endpoint, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// GetPosterURL returns the full URL for a poster image
func GetPosterURL(path string, size string) string {
	if path == "" {
		return ""
	}
	if size == "" {
		size = "w500"
	}
	return fmt.Sprintf("%s/%s%s", imageBaseURL, size, path)
}

// formatGenres converts genre objects to a comma-separated string
func formatGenres(genres []Genre) string {
	names := make([]string, len(genres))
	for i, g := range genres {
		names[i] = g.Name
	}
	return strings.Join(names, ", ")
}

// Common TMDB API response types

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProductionCompany struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SearchResultBase struct {
	ID           int     `json:"id"`
	Overview     string  `json:"overview"`
	PosterPath   *string `json:"poster_path"`
	VoteAverage  float64 `json:"vote_average"`
	OriginalLang string  `json:"original_language"`
}
