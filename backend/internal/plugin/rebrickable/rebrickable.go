// Package rebrickable imports individual LEGO catalog entries from Rebrickable.
package rebrickable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lmmendes/attic/internal/domain"
)

const (
	apiKeyEnv     = "ATTIC_REBRICKABLE_API_KEY"
	maxRetryAfter = 5 * time.Minute
)

var (
	errNotFound = errors.New("Rebrickable item not found")
	catalogID   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

// Client shares the request limit and throttle cooldown across both catalogs.
// The gate covers the response so a 429 updates the cooldown before another
// request can start. Waiting for either the gate or cooldown is cancellable.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	gate        chan struct{}
	nextRequest time.Time // guarded by gate
	interval    time.Duration
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		baseURL:    "https://rebrickable.com/api/v3/lego",
		gate:       make(chan struct{}, 1),
		interval:   time.Second,
	}
}

type Plugin struct {
	client   *Client
	catalog  string
	category string
}

func NewSets(client *Client) *Plugin {
	return &Plugin{client: client, catalog: "sets", category: "LEGO Sets"}
}
func NewMinifigs(client *Client) *Plugin {
	return &Plugin{client: client, catalog: "minifigs", category: "LEGO Minifigures"}
}
func (p *Plugin) ID() string { return "rebrickable_" + p.catalog }
func (p *Plugin) Name() string {
	if p.catalog == "minifigs" {
		return "Rebrickable Minifigures"
	}
	return "Rebrickable LEGO Sets"
}
func (p *Plugin) Description() string {
	return "Import " + p.category + " from the Rebrickable catalog"
}
func (p *Plugin) Enabled() bool { return strings.TrimSpace(os.Getenv(apiKeyEnv)) != "" }
func (p *Plugin) DisabledReason() string {
	if p.Enabled() {
		return ""
	}
	return "Missing API key: " + apiKeyEnv + ". Create a key at rebrickable.com/api/ and restart Attic."
}
func (p *Plugin) CategoryName() string        { return p.category }
func (p *Plugin) CategoryDescription() string { return p.category + " imported from Rebrickable" }
func (p *Plugin) Attributes() []domain.PluginAttribute {
	attrs := []domain.PluginAttribute{
		{Key: "catalog_number", Name: "Catalog Number", DataType: domain.AttributeTypeString},
		{Key: "piece_count", Name: "Piece Count", DataType: domain.AttributeTypeNumber},
		{Key: "url", Name: "Rebrickable URL", DataType: domain.AttributeTypeURL},
	}
	if p.catalog == "sets" {
		attrs = append(attrs, domain.PluginAttribute{Key: "release_year", Name: "Release Year", DataType: domain.AttributeTypeNumber})
	}
	return attrs
}
func (p *Plugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{{Key: "name", Label: "Name"}, {Key: "catalog_number", Label: "Catalog Number"}}
}

// Minifigures use the same set_num, set_img_url and set_url fields as sets.
type item struct {
	Number   string  `json:"set_num"`
	Name     string  `json:"name"`
	Year     *int    `json:"year"`
	Parts    *int    `json:"num_parts"`
	ImageURL *string `json:"set_img_url"`
	URL      string  `json:"set_url"`
}

func (i item) validate() error {
	if !catalogID.MatchString(i.Number) || strings.TrimSpace(i.Name) == "" {
		return errors.New("invalid Rebrickable catalog response")
	}
	return nil
}
func (i item) image() *string {
	if i.ImageURL == nil || strings.TrimSpace(*i.ImageURL) == "" {
		return nil
	}
	return i.ImageURL
}
func (i item) result() domain.SearchResult {
	subtitle := i.Number
	if i.Year != nil {
		subtitle += fmt.Sprintf(" · %d", *i.Year)
	}
	if i.Parts != nil {
		subtitle += fmt.Sprintf(" · %d pieces", *i.Parts)
	}
	return domain.SearchResult{ExternalID: i.Number, Title: i.Name, Subtitle: subtitle, ImageURL: i.image()}
}

func (p *Plugin) Search(ctx context.Context, field, query string, limit int) ([]domain.SearchResult, error) {
	query = strings.TrimSpace(query)
	results := []domain.SearchResult{}
	if field == "catalog_number" {
		i, err := p.lookup(ctx, query)
		if errors.Is(err, errNotFound) {
			return results, nil
		}
		if err != nil {
			return nil, err
		}
		return append(results, i.result()), nil
	}
	if field != "name" {
		return nil, errors.New("invalid Rebrickable search field")
	}
	if query == "" {
		return results, nil
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	var response struct {
		Results *[]item `json:"results"`
	}
	params := url.Values{"search": {query}, "page_size": {strconv.Itoa(limit)}}
	if err := p.client.get(ctx, "/"+p.catalog+"/?"+params.Encode(), &response); err != nil {
		return nil, err
	}
	if response.Results == nil {
		return nil, errors.New("invalid Rebrickable search response")
	}
	for _, i := range *response.Results {
		if err := i.validate(); err != nil {
			return nil, err
		}
		results = append(results, i.result())
		if len(results) == limit {
			break
		}
	}
	return results, nil
}

func (p *Plugin) lookup(ctx context.Context, id string) (*item, error) {
	if !catalogID.MatchString(id) {
		return nil, errors.New("invalid Rebrickable catalog number")
	}
	var i item
	if err := p.client.get(ctx, "/"+p.catalog+"/"+id+"/", &i); err != nil {
		return nil, err
	}
	if err := i.validate(); err != nil {
		return nil, err
	}
	if i.Number != id {
		return nil, errors.New("Rebrickable returned a different catalog number")
	}
	return &i, nil
}

func (p *Plugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	i, err := p.lookup(ctx, externalID)
	if err != nil {
		return nil, err
	}
	attrs := map[string]any{"catalog_number": i.Number}
	if i.Parts != nil {
		attrs["piece_count"] = *i.Parts
	}
	if i.URL != "" {
		value, err := domain.NormalizeURLAttribute(map[string]any{"url": i.URL}, false)
		if err != nil {
			slog.WarnContext(ctx, "omitting invalid plugin URL", "plugin_id", p.ID(), "external_id", externalID, "attribute_key", "url", "reason", err.Error())
		} else if value != nil {
			attrs["url"] = value
		}
	}
	if p.catalog == "sets" && i.Year != nil {
		attrs["release_year"] = *i.Year
	}
	return &domain.ImportData{Name: i.Name, ExternalID: i.Number, ImageURL: i.image(), Attributes: attrs}, nil
}

func (c *Client) get(ctx context.Context, path string, result any) error {
	key := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if key == "" {
		return errors.New("missing API key: " + apiKeyEnv)
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-c.gate }()
	if delay := time.Until(c.nextRequest); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating Rebrickable request: %w", err)
	}
	req.Header.Set("Authorization", "key "+key)
	req.Header.Set("Accept", "application/json")
	c.nextRequest = time.Now().Add(c.interval)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting Rebrickable catalog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAt := retryAfter(resp.Header.Get("Retry-After"), time.Now()); retryAt.After(c.nextRequest) {
			c.nextRequest = retryAt
		}
		return errors.New("Rebrickable request throttled; try again later")
	}
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("Rebrickable rejected API credentials")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Rebrickable API returned status %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 2<<20))
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decoding Rebrickable response: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("invalid trailing data in Rebrickable response")
	}
	return nil
}

func retryAfter(value string, now time.Time) time.Time {
	wait := time.Duration(0)
	if seconds, err := strconv.ParseUint(value, 10, 32); err == nil {
		wait = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(value); err == nil {
		wait = date.Sub(now)
	}
	return now.Add(min(max(wait, 0), maxRetryAfter))
}
