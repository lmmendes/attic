package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mendelui/attic/internal/domain"
)

const (
	MoviesPluginID = "tmdb_movies"
)

// MoviesPlugin implements the TMDB Movies import plugin
type MoviesPlugin struct {
	client *Client
}

// NewMoviesPlugin creates a new TMDB Movies plugin
func NewMoviesPlugin() *MoviesPlugin {
	return &MoviesPlugin{
		client: NewClient(),
	}
}

// ID returns the plugin identifier
func (p *MoviesPlugin) ID() string {
	return MoviesPluginID
}

// Name returns the display name
func (p *MoviesPlugin) Name() string {
	return "TMDB Movies"
}

// Description returns the plugin description
func (p *MoviesPlugin) Description() string {
	return "Import movies from The Movie Database (TMDB)"
}

// Enabled returns true if the TMDB API key is configured
func (p *MoviesPlugin) Enabled() bool {
	return IsEnabled()
}

// DisabledReason returns the reason the plugin is disabled
func (p *MoviesPlugin) DisabledReason() string {
	return GetDisabledReason()
}

// CategoryName returns the category this plugin manages
func (p *MoviesPlugin) CategoryName() string {
	return "Movies"
}

// CategoryDescription returns the category description
func (p *MoviesPlugin) CategoryDescription() string {
	return "Movies imported from TMDB"
}

// Attributes returns the attributes this plugin provides
func (p *MoviesPlugin) Attributes() []domain.PluginAttribute {
	return []domain.PluginAttribute{
		{Key: "movies.release_date", Name: "Release Date", DataType: domain.AttributeTypeDate, Required: false},
		{Key: "movies.genres", Name: "Genres", DataType: domain.AttributeTypeString, Required: false},
		{Key: "movies.rating", Name: "Rating", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "movies.runtime", Name: "Runtime (minutes)", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "movies.language", Name: "Original Language", DataType: domain.AttributeTypeString, Required: false},
		{Key: "movies.status", Name: "Status", DataType: domain.AttributeTypeString, Required: false},
		{Key: "movies.tagline", Name: "Tagline", DataType: domain.AttributeTypeString, Required: false},
		{Key: "movies.budget", Name: "Budget", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "movies.revenue", Name: "Revenue", DataType: domain.AttributeTypeNumber, Required: false},
	}
}

// SearchFields returns the available search fields
func (p *MoviesPlugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{
		{Key: "title", Label: "Title"},
	}
}

// Search searches for movies using the TMDB API
func (p *MoviesPlugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = defaultLimit
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("include_adult", "false")

	var apiResp movieSearchResponse
	if err := p.client.get(ctx, "/search/movie", params, &apiResp); err != nil {
		return nil, fmt.Errorf("searching movies: %w", err)
	}

	// Limit results
	items := apiResp.Results
	if len(items) > limit {
		items = items[:limit]
	}

	results := make([]domain.SearchResult, 0, len(items))
	for _, item := range items {
		result := domain.SearchResult{
			ExternalID: fmt.Sprintf("%d", item.ID),
			Title:      item.Title,
		}

		// Build subtitle with year
		var subtitleParts []string
		if item.ReleaseDate != "" && len(item.ReleaseDate) >= 4 {
			subtitleParts = append(subtitleParts, fmt.Sprintf("(%s)", item.ReleaseDate[:4]))
		}
		if item.VoteAverage > 0 {
			subtitleParts = append(subtitleParts, fmt.Sprintf("★ %.1f", item.VoteAverage))
		}
		result.Subtitle = strings.Join(subtitleParts, " ")

		// Get poster thumbnail
		if item.PosterPath != nil && *item.PosterPath != "" {
			posterURL := GetPosterURL(*item.PosterPath, "w185")
			result.ImageURL = &posterURL
		}

		results = append(results, result)
	}

	return results, nil
}

// Fetch retrieves full movie data by external ID
func (p *MoviesPlugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	endpoint := fmt.Sprintf("/movie/%s", url.PathEscape(externalID))

	var movie movieDetails
	if err := p.client.get(ctx, endpoint, nil, &movie); err != nil {
		return nil, fmt.Errorf("fetching movie: %w", err)
	}

	data := &domain.ImportData{
		Name:       movie.Title,
		ExternalID: externalID,
		Attributes: make(map[string]any),
	}

	// Description (overview)
	if movie.Overview != "" {
		data.Description = &movie.Overview
	}

	// Poster image
	if movie.PosterPath != nil && *movie.PosterPath != "" {
		posterURL := GetPosterURL(*movie.PosterPath, "w500")
		data.ImageURL = &posterURL
	}

	// Attributes
	if movie.ReleaseDate != "" {
		data.Attributes["movies.release_date"] = movie.ReleaseDate
	}

	if len(movie.Genres) > 0 {
		data.Attributes["movies.genres"] = formatGenres(movie.Genres)
	}

	if movie.VoteAverage > 0 {
		data.Attributes["movies.rating"] = movie.VoteAverage
	}

	if movie.Runtime > 0 {
		data.Attributes["movies.runtime"] = movie.Runtime
	}

	if movie.OriginalLanguage != "" {
		data.Attributes["movies.language"] = movie.OriginalLanguage
	}

	if movie.Status != "" {
		data.Attributes["movies.status"] = movie.Status
	}

	if movie.Tagline != "" {
		data.Attributes["movies.tagline"] = movie.Tagline
	}

	if movie.Budget > 0 {
		data.Attributes["movies.budget"] = movie.Budget
	}

	if movie.Revenue > 0 {
		data.Attributes["movies.revenue"] = movie.Revenue
	}

	return data, nil
}

// TMDB Movie API response types

type movieSearchResponse struct {
	Page         int                 `json:"page"`
	TotalPages   int                 `json:"total_pages"`
	TotalResults int                 `json:"total_results"`
	Results      []movieSearchResult `json:"results"`
}

type movieSearchResult struct {
	SearchResultBase
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}

type movieDetails struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Overview         string    `json:"overview"`
	PosterPath       *string   `json:"poster_path"`
	ReleaseDate      string    `json:"release_date"`
	Genres           []Genre   `json:"genres"`
	VoteAverage      float64   `json:"vote_average"`
	Runtime          int       `json:"runtime"`
	OriginalLanguage string    `json:"original_language"`
	Status           string    `json:"status"`
	Tagline          string    `json:"tagline"`
	Budget           int64     `json:"budget"`
	Revenue          int64     `json:"revenue"`
}
