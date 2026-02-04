package googlebooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lmmendes/attic/internal/domain"
)

const (
	PluginID    = "google_books"
	baseURL     = "https://www.googleapis.com/books/v1/volumes"
	defaultLimit = 10
)

// Plugin implements the Google Books import plugin
type Plugin struct {
	client *http.Client
}

// New creates a new Google Books plugin
func New() *Plugin {
	return &Plugin{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the display name
func (p *Plugin) Name() string {
	return "Google Books"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Import books from Google Books API"
}

// Enabled returns true as Google Books API doesn't require authentication
func (p *Plugin) Enabled() bool {
	return true
}

// DisabledReason returns empty string as Google Books is always enabled
func (p *Plugin) DisabledReason() string {
	return ""
}

// CategoryName returns the category this plugin manages
func (p *Plugin) CategoryName() string {
	return "Books"
}

// CategoryDescription returns the category description
func (p *Plugin) CategoryDescription() string {
	return "Books imported from Google Books"
}

// Attributes returns the attributes this plugin provides
func (p *Plugin) Attributes() []domain.PluginAttribute {
	return []domain.PluginAttribute{
		{Key: "books.isbn", Name: "ISBN", DataType: domain.AttributeTypeString, Required: false},
		{Key: "books.author", Name: "Author", DataType: domain.AttributeTypeString, Required: false},
		{Key: "books.publisher", Name: "Publisher", DataType: domain.AttributeTypeString, Required: false},
		{Key: "books.published_date", Name: "Published Date", DataType: domain.AttributeTypeString, Required: false},
		{Key: "books.page_count", Name: "Page Count", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "books.language", Name: "Language", DataType: domain.AttributeTypeString, Required: false},
		{Key: "books.categories", Name: "Categories", DataType: domain.AttributeTypeString, Required: false},
	}
}

// SearchFields returns the available search fields
func (p *Plugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{
		{Key: "title", Label: "Title"},
		{Key: "isbn", Label: "ISBN"},
		{Key: "author", Label: "Author"},
	}
}

// Search searches for books using the Google Books API
func (p *Plugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 40 {
		limit = defaultLimit
	}

	// Build the search query based on field
	var q string
	switch field {
	case "isbn":
		q = fmt.Sprintf("isbn:%s", query)
	case "author":
		q = fmt.Sprintf("inauthor:%s", query)
	case "title":
		fallthrough
	default:
		q = fmt.Sprintf("intitle:%s", query)
	}

	// Build URL
	u, _ := url.Parse(baseURL)
	params := url.Values{}
	params.Set("q", q)
	params.Set("maxResults", fmt.Sprintf("%d", limit))
	params.Set("printType", "books")
	u.RawQuery = params.Encode()

	// Make request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to SearchResult
	results := make([]domain.SearchResult, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		result := domain.SearchResult{
			ExternalID: item.ID,
			Title:      item.VolumeInfo.Title,
		}

		// Build subtitle (authors + year)
		var subtitleParts []string
		if len(item.VolumeInfo.Authors) > 0 {
			subtitleParts = append(subtitleParts, strings.Join(item.VolumeInfo.Authors, ", "))
		}
		if item.VolumeInfo.PublishedDate != "" {
			// Extract year from date
			year := item.VolumeInfo.PublishedDate
			if len(year) >= 4 {
				year = year[:4]
			}
			subtitleParts = append(subtitleParts, fmt.Sprintf("(%s)", year))
		}
		result.Subtitle = strings.Join(subtitleParts, " ")

		// Get thumbnail
		if item.VolumeInfo.ImageLinks.Thumbnail != "" {
			thumbnail := item.VolumeInfo.ImageLinks.Thumbnail
			// Use HTTPS
			thumbnail = strings.Replace(thumbnail, "http://", "https://", 1)
			result.ImageURL = &thumbnail
		}

		results = append(results, result)
	}

	return results, nil
}

// Fetch retrieves full book data by external ID
func (p *Plugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	u := fmt.Sprintf("%s/%s", baseURL, url.PathEscape(externalID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("book not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var item volumeItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Build ImportData
	data := &domain.ImportData{
		Name:       item.VolumeInfo.Title,
		ExternalID: item.ID,
		Attributes: make(map[string]any),
	}

	// Description
	if item.VolumeInfo.Description != "" {
		data.Description = &item.VolumeInfo.Description
	}

	// Image - prefer larger image
	if item.VolumeInfo.ImageLinks.Large != "" {
		img := strings.Replace(item.VolumeInfo.ImageLinks.Large, "http://", "https://", 1)
		data.ImageURL = &img
	} else if item.VolumeInfo.ImageLinks.Medium != "" {
		img := strings.Replace(item.VolumeInfo.ImageLinks.Medium, "http://", "https://", 1)
		data.ImageURL = &img
	} else if item.VolumeInfo.ImageLinks.Thumbnail != "" {
		img := strings.Replace(item.VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
		data.ImageURL = &img
	}

	// Attributes
	if len(item.VolumeInfo.IndustryIdentifiers) > 0 {
		// Prefer ISBN-13, fallback to ISBN-10
		for _, id := range item.VolumeInfo.IndustryIdentifiers {
			if id.Type == "ISBN_13" {
				data.Attributes["books.isbn"] = id.Identifier
				break
			}
			if id.Type == "ISBN_10" {
				data.Attributes["books.isbn"] = id.Identifier
			}
		}
	}

	if len(item.VolumeInfo.Authors) > 0 {
		data.Attributes["books.author"] = strings.Join(item.VolumeInfo.Authors, ", ")
	}

	if item.VolumeInfo.Publisher != "" {
		data.Attributes["books.publisher"] = item.VolumeInfo.Publisher
	}

	if item.VolumeInfo.PublishedDate != "" {
		data.Attributes["books.published_date"] = item.VolumeInfo.PublishedDate
	}

	if item.VolumeInfo.PageCount > 0 {
		data.Attributes["books.page_count"] = item.VolumeInfo.PageCount
	}

	if item.VolumeInfo.Language != "" {
		data.Attributes["books.language"] = item.VolumeInfo.Language
	}

	if len(item.VolumeInfo.Categories) > 0 {
		data.Attributes["books.categories"] = strings.Join(item.VolumeInfo.Categories, ", ")
	}

	return data, nil
}

// Google Books API response types

type searchResponse struct {
	TotalItems int          `json:"totalItems"`
	Items      []volumeItem `json:"items"`
}

type volumeItem struct {
	ID         string     `json:"id"`
	VolumeInfo volumeInfo `json:"volumeInfo"`
}

type volumeInfo struct {
	Title               string               `json:"title"`
	Authors             []string             `json:"authors"`
	Publisher           string               `json:"publisher"`
	PublishedDate       string               `json:"publishedDate"`
	Description         string               `json:"description"`
	IndustryIdentifiers []industryIdentifier `json:"industryIdentifiers"`
	PageCount           int                  `json:"pageCount"`
	Categories          []string             `json:"categories"`
	ImageLinks          imageLinks           `json:"imageLinks"`
	Language            string               `json:"language"`
}

type industryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type imageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
	Small          string `json:"small"`
	Medium         string `json:"medium"`
	Large          string `json:"large"`
	ExtraLarge     string `json:"extraLarge"`
}
