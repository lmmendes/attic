package domain

import "context"

// ImportPlugin defines the interface for all import plugins
type ImportPlugin interface {
	// Metadata
	ID() string          // Unique identifier, e.g., "google_books"
	Name() string        // Display name, e.g., "Google Books"
	Description() string // Brief description of the plugin

	// Category management
	CategoryName() string        // Category this plugin manages
	CategoryDescription() string // Description for the category
	Attributes() []PluginAttribute

	// Search capabilities
	SearchFields() []SearchField
	Search(ctx context.Context, field, query string, limit int) ([]SearchResult, error)

	// Data fetching
	Fetch(ctx context.Context, externalID string) (*ImportData, error)
}

// PluginAttribute defines an attribute managed by a plugin
type PluginAttribute struct {
	Key      string            `json:"key"`       // Namespaced key, e.g., "books.isbn"
	Name     string            `json:"name"`      // Display name, e.g., "ISBN"
	DataType AttributeDataType `json:"data_type"` // string, number, boolean, date, text
	Required bool              `json:"required"`  // Is this attribute required?
}

// SearchField defines a searchable field
type SearchField struct {
	Key   string `json:"key"`   // Field identifier, e.g., "title", "isbn"
	Label string `json:"label"` // Display label, e.g., "Title", "ISBN"
}

// SearchResult represents a search result from a plugin
type SearchResult struct {
	ExternalID string  `json:"external_id"` // External identifier for fetching full data
	Title      string  `json:"title"`       // Primary display title
	Subtitle   string  `json:"subtitle"`    // Secondary info (e.g., author, year)
	ImageURL   *string `json:"image_url"`   // Thumbnail URL if available
}

// ImportData contains the full data for an import
type ImportData struct {
	Name        string         `json:"name"`                   // Asset name
	Description *string        `json:"description,omitempty"`  // Asset description
	ImageURL    *string        `json:"image_url,omitempty"`    // Primary image URL
	Attributes  map[string]any `json:"attributes"`             // Attribute values (keyed by attribute key)
	ExternalID  string         `json:"external_id"`            // External ID for source tracking
}

// PluginInfo represents plugin metadata for API responses
type PluginInfo struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	CategoryName        string        `json:"category_name"`
	CategoryDescription string        `json:"category_description"`
	SearchFields        []SearchField `json:"search_fields"`
	Attributes          []PluginAttribute `json:"attributes"`
}

// ToInfo converts an ImportPlugin to PluginInfo for API responses
func PluginToInfo(p ImportPlugin) PluginInfo {
	return PluginInfo{
		ID:                  p.ID(),
		Name:                p.Name(),
		Description:         p.Description(),
		CategoryName:        p.CategoryName(),
		CategoryDescription: p.CategoryDescription(),
		SearchFields:        p.SearchFields(),
		Attributes:          p.Attributes(),
	}
}
