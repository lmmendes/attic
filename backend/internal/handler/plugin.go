package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/plugin"
)

// PluginHandler handles plugin-related HTTP requests
type PluginHandler struct {
	registry *plugin.Registry
	repos    *Repositories
	storage  FileStorage
	orgID    uuid.UUID
}

// NewPluginHandler creates a new PluginHandler
func NewPluginHandler(registry *plugin.Registry, repos *Repositories, storage FileStorage) *PluginHandler {
	return &PluginHandler{
		registry: registry,
		repos:    repos,
		storage:  storage,
		orgID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
}

// PluginListResponse represents the response for listing plugins
type PluginListResponse struct {
	Plugins []PluginResponse `json:"plugins"`
}

// PluginResponse represents a plugin in API responses
type PluginResponse struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	Enabled             bool                     `json:"enabled"`
	DisabledReason      string                   `json:"disabled_reason,omitempty"`
	CategoryName        string                   `json:"category_name"`
	CategoryDescription string                   `json:"category_description"`
	SearchFields        []domain.SearchField     `json:"search_fields"`
	Attributes          []domain.PluginAttribute `json:"attributes"`
	CategoryID          *uuid.UUID               `json:"category_id,omitempty"`
}

// SearchResponse represents the response for plugin search
type SearchResponse struct {
	Results []domain.SearchResult `json:"results"`
}

// ImportRequest represents the request body for importing
type ImportRequest struct {
	ExternalID string `json:"external_id"`
}

// ImportResponse represents the response for importing
type ImportResponse struct {
	Asset *domain.Asset `json:"asset"`
}

// ListPlugins returns all available plugins
func (h *PluginHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := h.registry.List()

	response := PluginListResponse{
		Plugins: make([]PluginResponse, 0, len(plugins)),
	}

	for _, p := range plugins {
		pr := PluginResponse{
			ID:                  p.ID(),
			Name:                p.Name(),
			Description:         p.Description(),
			Enabled:             p.Enabled(),
			DisabledReason:      p.DisabledReason(),
			CategoryName:        p.CategoryName(),
			CategoryDescription: p.CategoryDescription(),
			SearchFields:        p.SearchFields(),
			Attributes:          p.Attributes(),
		}

		// Check if category exists for this plugin
		cat, _ := h.repos.Categories.GetByPluginID(r.Context(), h.orgID, p.ID())
		if cat != nil {
			pr.CategoryID = &cat.ID
		}

		response.Plugins = append(response.Plugins, pr)
	}

	writeJSON(w, http.StatusOK, response)
}

// GetPlugin returns a specific plugin's info
func (h *PluginHandler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, "plugin not found")
		return
	}

	pr := PluginResponse{
		ID:                  p.ID(),
		Name:                p.Name(),
		Description:         p.Description(),
		Enabled:             p.Enabled(),
		DisabledReason:      p.DisabledReason(),
		CategoryName:        p.CategoryName(),
		CategoryDescription: p.CategoryDescription(),
		SearchFields:        p.SearchFields(),
		Attributes:          p.Attributes(),
	}

	// Check if category exists for this plugin
	cat, _ := h.repos.Categories.GetByPluginID(r.Context(), h.orgID, p.ID())
	if cat != nil {
		pr.CategoryID = &cat.ID
	}

	writeJSON(w, http.StatusOK, pr)
}

// Search performs a search using a plugin
func (h *PluginHandler) Search(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("plugin '%s' not found", pluginID))
		return
	}

	if !p.Enabled() {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("plugin '%s' is disabled: %s", pluginID, p.DisabledReason()))
		return
	}

	q := r.URL.Query()
	field := q.Get("field")
	query := q.Get("q")
	limit, _ := strconv.Atoi(q.Get("limit"))

	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	if len(query) < 2 {
		writeError(w, http.StatusBadRequest, "search query must be at least 2 characters")
		return
	}

	// Validate search field
	searchFields := p.SearchFields()
	if field == "" {
		// Default to first search field
		if len(searchFields) > 0 {
			field = searchFields[0].Key
		}
	} else {
		// Validate field exists
		validField := false
		for _, f := range searchFields {
			if f.Key == field {
				validField = true
				break
			}
		}
		if !validField {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid search field '%s'", field))
			return
		}
	}

	if limit <= 0 || limit > 20 {
		limit = 10
	}

	results, err := p.Search(r.Context(), field, query, limit)
	if err != nil {
		slog.Error("plugin search failed",
			"plugin_id", pluginID,
			"field", field,
			"query", query,
			"error", err)

		// Check for context cancellation (user navigated away)
		if r.Context().Err() != nil {
			return
		}

		writeError(w, http.StatusBadGateway, "search service temporarily unavailable")
		return
	}

	if results == nil {
		results = []domain.SearchResult{}
	}

	writeJSON(w, http.StatusOK, SearchResponse{Results: results})
}

// Import fetches data from a plugin and creates an asset
func (h *PluginHandler) Import(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginId")

	p, exists := h.registry.Get(pluginID)
	if !exists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("plugin '%s' not found", pluginID))
		return
	}

	if !p.Enabled() {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("plugin '%s' is disabled: %s", pluginID, p.DisabledReason()))
		return
	}

	var req ImportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: expected JSON with 'external_id' field")
		return
	}

	if req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "external_id is required")
		return
	}

	slog.Info("importing item from plugin",
		"plugin_id", pluginID,
		"external_id", req.ExternalID)

	// Fetch data from plugin
	importData, err := p.Fetch(r.Context(), req.ExternalID)
	if err != nil {
		slog.Error("failed to fetch import data",
			"plugin_id", pluginID,
			"external_id", req.ExternalID,
			"error", err)

		// Check for context cancellation
		if r.Context().Err() != nil {
			return
		}

		// Check for "not found" type errors
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "item not found in external source")
			return
		}

		writeError(w, http.StatusBadGateway, "failed to fetch data from external source")
		return
	}

	// Validate import data
	if importData.Name == "" {
		writeError(w, http.StatusBadGateway, "external source returned invalid data (missing name)")
		return
	}

	// Ensure category exists for this plugin
	cat, err := h.ensurePluginCategory(r, p)
	if err != nil {
		slog.Error("failed to ensure plugin category",
			"plugin_id", pluginID,
			"error", err)
		writeError(w, http.StatusInternalServerError, "failed to initialize plugin category")
		return
	}

	// Convert attributes to JSON
	attrsJSON, err := json.Marshal(importData.Attributes)
	if err != nil {
		slog.Error("failed to marshal import attributes",
			"plugin_id", pluginID,
			"error", err)
		writeError(w, http.StatusInternalServerError, "failed to process import data")
		return
	}

	// Create the asset
	asset := &domain.Asset{
		OrganizationID:   h.orgID,
		CategoryID:       cat.ID,
		Name:             importData.Name,
		Description:      importData.Description,
		Quantity:         1,
		Attributes:       attrsJSON,
		ImportPluginID:   &pluginID,
		ImportExternalID: &importData.ExternalID,
	}

	if err := h.repos.Assets.Create(r.Context(), asset); err != nil {
		slog.Error("failed to create imported asset",
			"plugin_id", pluginID,
			"external_id", req.ExternalID,
			"error", err)
		writeError(w, http.StatusInternalServerError, "failed to save imported item")
		return
	}

	slog.Info("successfully imported item",
		"plugin_id", pluginID,
		"external_id", req.ExternalID,
		"asset_id", asset.ID)

	// Download and store image if available
	if importData.ImageURL != nil && *importData.ImageURL != "" && h.storage != nil {
		if err := h.downloadAndStoreImage(r.Context(), asset.ID, *importData.ImageURL); err != nil {
			// Log error but don't fail the import - image is optional
			slog.Warn("failed to download image for imported asset",
				"asset_id", asset.ID,
				"image_url", *importData.ImageURL,
				"error", err)
		}
	}

	// Load full asset with category
	asset.Category = cat

	writeJSON(w, http.StatusCreated, ImportResponse{Asset: asset})
}

// ensurePluginCategory ensures the plugin's category and attributes exist
func (h *PluginHandler) ensurePluginCategory(r *http.Request, p domain.ImportPlugin) (*domain.Category, error) {
	ctx := r.Context()
	pluginID := p.ID()

	// Check if category already exists
	cat, err := h.repos.Categories.GetByPluginID(ctx, h.orgID, pluginID)
	if err != nil {
		return nil, err
	}

	if cat != nil {
		return cat, nil
	}

	// Create category
	cat = &domain.Category{
		OrganizationID: h.orgID,
		PluginID:       &pluginID,
		Name:           p.CategoryName(),
		Description:    strPtr(p.CategoryDescription()),
	}

	if err := h.repos.Categories.Create(ctx, cat); err != nil {
		return nil, err
	}

	// Create plugin attributes
	pluginAttrs := p.Attributes()
	assignments := make([]domain.CategoryAttributeAssignment, 0, len(pluginAttrs))

	for i, pa := range pluginAttrs {
		// Check if attribute already exists
		attr, err := h.repos.Attributes.GetByKey(ctx, h.orgID, pa.Key)
		if err != nil {
			return nil, err
		}

		if attr == nil {
			// Create the attribute
			attr = &domain.Attribute{
				OrganizationID: h.orgID,
				PluginID:       &pluginID,
				Name:           pa.Name,
				Key:            pa.Key,
				DataType:       pa.DataType,
			}
			if err := h.repos.Attributes.Create(ctx, attr); err != nil {
				return nil, err
			}
		}

		assignments = append(assignments, domain.CategoryAttributeAssignment{
			AttributeID: attr.ID,
			Required:    pa.Required,
			SortOrder:   i,
		})
	}

	// Assign attributes to category
	if err := h.repos.Categories.SetAttributes(ctx, cat.ID, assignments); err != nil {
		return nil, err
	}

	return cat, nil
}

func strPtr(s string) *string {
	return &s
}

// downloadAndStoreImage downloads an image from URL and stores it as an attachment
func (h *PluginHandler) downloadAndStoreImage(ctx context.Context, assetID uuid.UUID, imageURL string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Download the image
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	// Limit download size to 10MB
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
	imageData, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("reading image data: %w", err)
	}

	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = http.DetectContentType(imageData)
	}

	// Only accept image content types
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("invalid content type: %s", contentType)
	}

	// Generate filename from URL or use default
	filename := "cover"
	if urlPath := path.Base(imageURL); urlPath != "" && urlPath != "/" {
		// Remove query string
		if idx := strings.Index(urlPath, "?"); idx > 0 {
			urlPath = urlPath[:idx]
		}
		if urlPath != "" {
			filename = urlPath
		}
	}

	// Add extension based on content type if missing
	ext := ""
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	if ext != "" && !strings.Contains(filename, ".") {
		filename += ext
	}

	// Upload to storage
	key, err := h.storage.Upload(ctx, filename, contentType, bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("uploading to storage: %w", err)
	}

	// Create attachment record
	description := "Imported cover image"
	attachment := &domain.Attachment{
		AssetID:     assetID,
		FileKey:     key,
		FileName:    filename,
		FileSize:    int64(len(imageData)),
		ContentType: &contentType,
		Description: &description,
	}

	if err := h.repos.Attachments.Create(ctx, attachment); err != nil {
		// Try to clean up uploaded file
		h.storage.Delete(ctx, key)
		return fmt.Errorf("creating attachment record: %w", err)
	}

	slog.Info("downloaded and stored import image",
		"asset_id", assetID,
		"attachment_id", attachment.ID,
		"filename", filename,
		"size", len(imageData))

	return nil
}
