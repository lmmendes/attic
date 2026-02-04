package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lmmendes/attic/internal/domain"
)

const (
	SeriesPluginID = "tmdb_series"
)

// SeriesPlugin implements the TMDB TV Series import plugin
type SeriesPlugin struct {
	client *Client
}

// NewSeriesPlugin creates a new TMDB TV Series plugin
func NewSeriesPlugin() *SeriesPlugin {
	return &SeriesPlugin{
		client: NewClient(),
	}
}

// ID returns the plugin identifier
func (p *SeriesPlugin) ID() string {
	return SeriesPluginID
}

// Name returns the display name
func (p *SeriesPlugin) Name() string {
	return "TMDB TV Series"
}

// Description returns the plugin description
func (p *SeriesPlugin) Description() string {
	return "Import TV series from The Movie Database (TMDB)"
}

// Enabled returns true if the TMDB API key is configured
func (p *SeriesPlugin) Enabled() bool {
	return IsEnabled()
}

// DisabledReason returns the reason the plugin is disabled
func (p *SeriesPlugin) DisabledReason() string {
	return GetDisabledReason()
}

// CategoryName returns the category this plugin manages
func (p *SeriesPlugin) CategoryName() string {
	return "TV Series"
}

// CategoryDescription returns the category description
func (p *SeriesPlugin) CategoryDescription() string {
	return "TV series imported from TMDB"
}

// Attributes returns the attributes this plugin provides
func (p *SeriesPlugin) Attributes() []domain.PluginAttribute {
	return []domain.PluginAttribute{
		{Key: "series.first_air_date", Name: "First Air Date", DataType: domain.AttributeTypeDate, Required: false},
		{Key: "series.last_air_date", Name: "Last Air Date", DataType: domain.AttributeTypeDate, Required: false},
		{Key: "series.genres", Name: "Genres", DataType: domain.AttributeTypeString, Required: false},
		{Key: "series.rating", Name: "Rating", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "series.seasons", Name: "Number of Seasons", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "series.episodes", Name: "Number of Episodes", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "series.language", Name: "Original Language", DataType: domain.AttributeTypeString, Required: false},
		{Key: "series.status", Name: "Status", DataType: domain.AttributeTypeString, Required: false},
	}
}

// SearchFields returns the available search fields
func (p *SeriesPlugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{
		{Key: "name", Label: "Title"},
	}
}

// Search searches for TV series using the TMDB API
func (p *SeriesPlugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = defaultLimit
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("include_adult", "false")

	var apiResp seriesSearchResponse
	if err := p.client.get(ctx, "/search/tv", params, &apiResp); err != nil {
		return nil, fmt.Errorf("searching TV series: %w", err)
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
			Title:      item.Name,
		}

		// Build subtitle with year
		var subtitleParts []string
		if item.FirstAirDate != "" && len(item.FirstAirDate) >= 4 {
			subtitleParts = append(subtitleParts, fmt.Sprintf("(%s)", item.FirstAirDate[:4]))
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

// Fetch retrieves full TV series data by external ID
func (p *SeriesPlugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	endpoint := fmt.Sprintf("/tv/%s", url.PathEscape(externalID))

	var series seriesDetails
	if err := p.client.get(ctx, endpoint, nil, &series); err != nil {
		return nil, fmt.Errorf("fetching TV series: %w", err)
	}

	data := &domain.ImportData{
		Name:       series.Name,
		ExternalID: externalID,
		Attributes: make(map[string]any),
	}

	// Description (overview)
	if series.Overview != "" {
		data.Description = &series.Overview
	}

	// Poster image
	if series.PosterPath != nil && *series.PosterPath != "" {
		posterURL := GetPosterURL(*series.PosterPath, "w500")
		data.ImageURL = &posterURL
	}

	// Attributes
	if series.FirstAirDate != "" {
		data.Attributes["series.first_air_date"] = series.FirstAirDate
	}

	if series.LastAirDate != "" {
		data.Attributes["series.last_air_date"] = series.LastAirDate
	}

	if len(series.Genres) > 0 {
		data.Attributes["series.genres"] = formatGenres(series.Genres)
	}

	if series.VoteAverage > 0 {
		data.Attributes["series.rating"] = series.VoteAverage
	}

	if series.NumberOfSeasons > 0 {
		data.Attributes["series.seasons"] = series.NumberOfSeasons
	}

	if series.NumberOfEpisodes > 0 {
		data.Attributes["series.episodes"] = series.NumberOfEpisodes
	}

	if series.OriginalLanguage != "" {
		data.Attributes["series.language"] = series.OriginalLanguage
	}

	if series.Status != "" {
		data.Attributes["series.status"] = series.Status
	}

	return data, nil
}

// TMDB TV Series API response types

type seriesSearchResponse struct {
	Page         int                  `json:"page"`
	TotalPages   int                  `json:"total_pages"`
	TotalResults int                  `json:"total_results"`
	Results      []seriesSearchResult `json:"results"`
}

type seriesSearchResult struct {
	SearchResultBase
	Name         string `json:"name"`
	FirstAirDate string `json:"first_air_date"`
}

type seriesDetails struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	Overview         string  `json:"overview"`
	PosterPath       *string `json:"poster_path"`
	FirstAirDate     string  `json:"first_air_date"`
	LastAirDate      string  `json:"last_air_date"`
	Genres           []Genre `json:"genres"`
	VoteAverage      float64 `json:"vote_average"`
	NumberOfSeasons  int     `json:"number_of_seasons"`
	NumberOfEpisodes int     `json:"number_of_episodes"`
	OriginalLanguage string  `json:"original_language"`
	Status           string  `json:"status"`
}
