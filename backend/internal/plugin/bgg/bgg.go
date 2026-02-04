package bgg

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lmmendes/attic/internal/domain"
)

const (
	PluginID        = "bgg_boardgames"
	baseURL         = "https://boardgamegeek.com/xmlapi2"
	defaultLimit    = 10
	bggAPIKeyEnvVar = "ATTIC_BGG_API_KEY"
)

// APIKey can be set at build time via ldflags:
// go build -ldflags="-X github.com/lmmendes/attic/internal/plugin/bgg.APIKey=your-key"
var APIKey = ""

// getAPIKey returns the API key, preferring environment variable over build-time value
func getAPIKey() string {
	if key := os.Getenv(bggAPIKeyEnvVar); key != "" {
		return key
	}
	return APIKey
}

// Plugin implements the BoardGameGeek import plugin
type Plugin struct {
	client       *http.Client
	lastRequest  time.Time
	rateLimitMu  sync.Mutex
}

// New creates a new BGG plugin
func New() *Plugin {
	return &Plugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the display name
func (p *Plugin) Name() string {
	return "BoardGameGeek"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Import board games from BoardGameGeek"
}

// Enabled returns true if the BGG API key is configured
func (p *Plugin) Enabled() bool {
	return getAPIKey() != ""
}

// DisabledReason returns the reason the plugin is disabled
func (p *Plugin) DisabledReason() string {
	if p.Enabled() {
		return ""
	}
	return "Missing API key: " + bggAPIKeyEnvVar
}

// CategoryName returns the category this plugin manages
func (p *Plugin) CategoryName() string {
	return "Board Games"
}

// CategoryDescription returns the category description
func (p *Plugin) CategoryDescription() string {
	return "Board games imported from BoardGameGeek"
}

// Attributes returns the attributes this plugin provides
func (p *Plugin) Attributes() []domain.PluginAttribute {
	return []domain.PluginAttribute{
		{Key: "boardgames.year_published", Name: "Year Published", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.min_players", Name: "Min Players", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.max_players", Name: "Max Players", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.playing_time", Name: "Playing Time (min)", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.min_playtime", Name: "Min Playtime (min)", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.max_playtime", Name: "Max Playtime (min)", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.min_age", Name: "Minimum Age", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.rating", Name: "BGG Rating", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.weight", Name: "Complexity/Weight", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "boardgames.designers", Name: "Designers", DataType: domain.AttributeTypeString, Required: false},
		{Key: "boardgames.publishers", Name: "Publishers", DataType: domain.AttributeTypeString, Required: false},
		{Key: "boardgames.categories", Name: "Categories", DataType: domain.AttributeTypeString, Required: false},
		{Key: "boardgames.mechanics", Name: "Mechanics", DataType: domain.AttributeTypeString, Required: false},
	}
}

// SearchFields returns the available search fields
func (p *Plugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{
		{Key: "name", Label: "Name"},
	}
}

// Search searches for board games using the BGG API
func (p *Plugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	apiKey := getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("BGG API key not configured")
	}

	// Rate limiting
	p.waitForRateLimit()

	// Build URL
	u, _ := url.Parse(baseURL + "/search")
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "boardgame")
	u.RawQuery = params.Encode()

	// Make request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var searchResp searchResponse
	if err := xml.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Limit results
	itemCount := min(limit, len(searchResp.Items))
	if itemCount == 0 {
		return []domain.SearchResult{}, nil
	}

	// Collect IDs for batch thumbnail fetch
	ids := make([]string, itemCount)
	for i := 0; i < itemCount; i++ {
		ids[i] = searchResp.Items[i].ID
	}

	// Fetch thumbnails in batch (single API call)
	thumbnails := p.fetchThumbnails(ctx, apiKey, ids)

	// Convert to SearchResult
	results := make([]domain.SearchResult, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		item := searchResp.Items[i]

		result := domain.SearchResult{
			ExternalID: item.ID,
			Title:      item.Name.Value,
		}

		// Build subtitle with year
		if item.YearPublished.Value != "" {
			result.Subtitle = fmt.Sprintf("(%s)", item.YearPublished.Value)
		}

		// Add thumbnail if available
		if thumb, ok := thumbnails[item.ID]; ok && thumb != "" {
			result.ImageURL = &thumb
		}

		results = append(results, result)
	}

	return results, nil
}

// Fetch retrieves full board game data by external ID
func (p *Plugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	apiKey := getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("BGG API key not configured")
	}

	// Rate limiting
	p.waitForRateLimit()

	// Build URL
	u, _ := url.Parse(baseURL + "/thing")
	params := url.Values{}
	params.Set("id", externalID)
	params.Set("stats", "1")
	u.RawQuery = params.Encode()

	// Make request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("board game not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var thingResp thingResponse
	if err := xml.NewDecoder(resp.Body).Decode(&thingResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(thingResp.Items) == 0 {
		return nil, fmt.Errorf("board game not found")
	}

	item := thingResp.Items[0]

	// Find primary name
	name := ""
	for _, n := range item.Names {
		if n.Type == "primary" {
			name = n.Value
			break
		}
	}
	if name == "" && len(item.Names) > 0 {
		name = item.Names[0].Value
	}

	// Build ImportData
	data := &domain.ImportData{
		Name:       name,
		ExternalID: externalID,
		Attributes: make(map[string]any),
	}

	// Description
	if item.Description != "" {
		desc := cleanDescription(item.Description)
		data.Description = &desc
	}

	// Image
	if item.Image != "" {
		data.ImageURL = &item.Image
	} else if item.Thumbnail != "" {
		data.ImageURL = &item.Thumbnail
	}

	// Attributes
	if item.YearPublished.Value != "" {
		if year, err := strconv.Atoi(item.YearPublished.Value); err == nil {
			data.Attributes["boardgames.year_published"] = year
		}
	}

	if item.MinPlayers.Value != "" {
		if v, err := strconv.Atoi(item.MinPlayers.Value); err == nil {
			data.Attributes["boardgames.min_players"] = v
		}
	}

	if item.MaxPlayers.Value != "" {
		if v, err := strconv.Atoi(item.MaxPlayers.Value); err == nil {
			data.Attributes["boardgames.max_players"] = v
		}
	}

	if item.PlayingTime.Value != "" {
		if v, err := strconv.Atoi(item.PlayingTime.Value); err == nil {
			data.Attributes["boardgames.playing_time"] = v
		}
	}

	if item.MinPlayTime.Value != "" {
		if v, err := strconv.Atoi(item.MinPlayTime.Value); err == nil {
			data.Attributes["boardgames.min_playtime"] = v
		}
	}

	if item.MaxPlayTime.Value != "" {
		if v, err := strconv.Atoi(item.MaxPlayTime.Value); err == nil {
			data.Attributes["boardgames.max_playtime"] = v
		}
	}

	if item.MinAge.Value != "" {
		if v, err := strconv.Atoi(item.MinAge.Value); err == nil {
			data.Attributes["boardgames.min_age"] = v
		}
	}

	// Statistics
	if item.Statistics.Ratings.Average.Value != "" {
		if v, err := strconv.ParseFloat(item.Statistics.Ratings.Average.Value, 64); err == nil {
			data.Attributes["boardgames.rating"] = v
		}
	}

	if item.Statistics.Ratings.AverageWeight.Value != "" {
		if v, err := strconv.ParseFloat(item.Statistics.Ratings.AverageWeight.Value, 64); err == nil {
			data.Attributes["boardgames.weight"] = v
		}
	}

	// Links (designers, publishers, categories, mechanics)
	designers := extractLinks(item.Links, "boardgamedesigner")
	if len(designers) > 0 {
		data.Attributes["boardgames.designers"] = strings.Join(designers, ", ")
	}

	publishers := extractLinks(item.Links, "boardgamepublisher")
	if len(publishers) > 0 {
		data.Attributes["boardgames.publishers"] = strings.Join(publishers, ", ")
	}

	categories := extractLinks(item.Links, "boardgamecategory")
	if len(categories) > 0 {
		data.Attributes["boardgames.categories"] = strings.Join(categories, ", ")
	}

	mechanics := extractLinks(item.Links, "boardgamemechanic")
	if len(mechanics) > 0 {
		data.Attributes["boardgames.mechanics"] = strings.Join(mechanics, ", ")
	}

	return data, nil
}

// waitForRateLimit ensures we don't exceed BGG's rate limits (~5 seconds between requests)
func (p *Plugin) waitForRateLimit() {
	p.rateLimitMu.Lock()
	defer p.rateLimitMu.Unlock()

	elapsed := time.Since(p.lastRequest)
	if elapsed < 5*time.Second {
		time.Sleep(5*time.Second - elapsed)
	}
	p.lastRequest = time.Now()
}

// fetchThumbnails fetches thumbnails for multiple IDs in a single API call
func (p *Plugin) fetchThumbnails(ctx context.Context, apiKey string, ids []string) map[string]string {
	thumbnails := make(map[string]string)
	if len(ids) == 0 {
		return thumbnails
	}

	// Rate limiting
	p.waitForRateLimit()

	// Build URL with comma-separated IDs
	u, _ := url.Parse(baseURL + "/thing")
	params := url.Values{}
	params.Set("id", strings.Join(ids, ","))
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return thumbnails
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return thumbnails
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return thumbnails
	}

	var thingResp thingResponse
	if err := xml.NewDecoder(resp.Body).Decode(&thingResp); err != nil {
		return thumbnails
	}

	for _, item := range thingResp.Items {
		if item.Thumbnail != "" {
			thumbnails[item.ID] = item.Thumbnail
		} else if item.Image != "" {
			thumbnails[item.ID] = item.Image
		}
	}

	return thumbnails
}

// extractLinks extracts values from links of a specific type
func extractLinks(links []link, linkType string) []string {
	var values []string
	for _, l := range links {
		if l.Type == linkType {
			values = append(values, l.Value)
		}
	}
	return values
}

// cleanDescription removes HTML entities and basic HTML tags from description
func cleanDescription(desc string) string {
	// Replace common HTML entities
	replacer := strings.NewReplacer(
		"&#10;", "\n",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", "\"",
		"&apos;", "'",
		"&mdash;", "—",
		"&ndash;", "–",
	)
	return replacer.Replace(desc)
}

// BGG XML API response types

type searchResponse struct {
	XMLName xml.Name     `xml:"items"`
	Total   int          `xml:"total,attr"`
	Items   []searchItem `xml:"item"`
}

type searchItem struct {
	Type          string       `xml:"type,attr"`
	ID            string       `xml:"id,attr"`
	Name          valueAttr    `xml:"name"`
	YearPublished valueAttr    `xml:"yearpublished"`
}

type thingResponse struct {
	XMLName xml.Name    `xml:"items"`
	Items   []thingItem `xml:"item"`
}

type thingItem struct {
	Type          string      `xml:"type,attr"`
	ID            string      `xml:"id,attr"`
	Thumbnail     string      `xml:"thumbnail"`
	Image         string      `xml:"image"`
	Names         []nameAttr  `xml:"name"`
	Description   string      `xml:"description"`
	YearPublished valueAttr   `xml:"yearpublished"`
	MinPlayers    valueAttr   `xml:"minplayers"`
	MaxPlayers    valueAttr   `xml:"maxplayers"`
	PlayingTime   valueAttr   `xml:"playingtime"`
	MinPlayTime   valueAttr   `xml:"minplaytime"`
	MaxPlayTime   valueAttr   `xml:"maxplaytime"`
	MinAge        valueAttr   `xml:"minage"`
	Links         []link      `xml:"link"`
	Statistics    statistics  `xml:"statistics"`
}

type valueAttr struct {
	Value string `xml:"value,attr"`
}

type nameAttr struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type link struct {
	Type  string `xml:"type,attr"`
	ID    string `xml:"id,attr"`
	Value string `xml:"value,attr"`
}

type statistics struct {
	Ratings ratings `xml:"ratings"`
}

type ratings struct {
	Average       valueAttr `xml:"average"`
	AverageWeight valueAttr `xml:"averageweight"`
}
