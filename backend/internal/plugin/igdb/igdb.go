package igdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lmmendes/attic/internal/domain"
)

const (
	// PluginID is the unique identifier for this plugin.
	PluginID = "igdb_games"

	apiBaseURL   = "https://api.igdb.com/v4"
	imageBaseURL = "https://images.igdb.com/igdb/image/upload"

	defaultLimit        = 10
	maxPlatformsPerGame = 6
	rateLimitWindow     = 250 * time.Millisecond // ~4 requests/sec (IGDB's documented cap)
)

// Plugin implements the IGDB import plugin.
type Plugin struct {
	httpClient *http.Client
	tokens     *tokenSource

	rateMu      sync.Mutex
	lastRequest time.Time
}

// New creates a new IGDB plugin instance.
func New() *Plugin {
	return &Plugin{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		tokens:     newTokenSource(),
	}
}

// Plugin metadata

func (p *Plugin) ID() string          { return PluginID }
func (p *Plugin) Name() string        { return "IGDB" }
func (p *Plugin) Description() string { return "Import video games from the Internet Game Database (IGDB)" }

func (p *Plugin) Enabled() bool          { return IsEnabled() }
func (p *Plugin) DisabledReason() string { return GetDisabledReason() }

func (p *Plugin) CategoryName() string        { return "Video Games" }
func (p *Plugin) CategoryDescription() string { return "Video games imported from IGDB" }

// Attributes returns the namespaced attributes managed by this plugin.
func (p *Plugin) Attributes() []domain.PluginAttribute {
	return []domain.PluginAttribute{
		{Key: "games.platform", Name: "Platform", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.release_date", Name: "Release Date", DataType: domain.AttributeTypeDate, Required: false},
		{Key: "games.year_released", Name: "Year Released", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "games.genres", Name: "Genres", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.themes", Name: "Themes", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.developers", Name: "Developers", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.publishers", Name: "Publishers", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.game_modes", Name: "Game Modes", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.player_perspectives", Name: "Player Perspective", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.franchise", Name: "Franchise", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.engine", Name: "Game Engine", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.age_rating", Name: "Age Rating", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.category", Name: "Edition Type", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.status", Name: "Release Status", DataType: domain.AttributeTypeString, Required: false},
		{Key: "games.rating", Name: "IGDB Rating", DataType: domain.AttributeTypeNumber, Required: false},
		{Key: "games.url", Name: "IGDB URL", DataType: domain.AttributeTypeString, Required: false},
	}
}

// SearchFields returns the searchable fields exposed by this plugin.
func (p *Plugin) SearchFields() []domain.SearchField {
	return []domain.SearchField{
		{Key: "name", Label: "Name"},
	}
}

// Search returns one row per (game, platform) combination so users can pick
// the platform that matches their physical/digital copy when importing.
//
// External IDs are encoded as "<gameID>:<platformID>" (or just "<gameID>" when
// the game has no platform metadata). Fetch parses this composite back apart.
func (p *Plugin) Search(ctx context.Context, _, query string, limit int) ([]domain.SearchResult, error) {
	if limit <= 0 || limit > 25 {
		limit = defaultLimit
	}

	// Escape any embedded double-quotes in the user's query.
	escapedQuery := strings.ReplaceAll(query, `"`, `\"`)
	body := fmt.Sprintf(
		`search "%s"; fields name, first_release_date, cover.image_id, platforms.id, platforms.name, release_dates.platform, release_dates.date; limit %d;`,
		escapedQuery, limit,
	)

	var games []searchGame
	if err := p.post(ctx, "/games", body, &games); err != nil {
		return nil, fmt.Errorf("searching games: %w", err)
	}

	results := make([]domain.SearchResult, 0, len(games))
	for _, g := range games {
		results = append(results, p.expandSearchResult(g)...)
	}
	return results, nil
}

// Fetch retrieves full data for a (game, platform) combination.
func (p *Plugin) Fetch(ctx context.Context, externalID string) (*domain.ImportData, error) {
	gameID, platformID, err := parseExternalID(externalID)
	if err != nil {
		return nil, err
	}

	body := fmt.Sprintf(
		`fields name, summary, storyline, first_release_date, cover.image_id, platforms.name, genres.name, themes.name, game_modes.name, player_perspectives.name, involved_companies.developer, involved_companies.publisher, involved_companies.company.name, rating, age_ratings.category, age_ratings.rating, franchises.name, game_engines.name, status, category, url, release_dates.platform, release_dates.date; where id = %d; limit 1;`,
		gameID,
	)

	var games []fetchGame
	if err := p.post(ctx, "/games", body, &games); err != nil {
		return nil, fmt.Errorf("fetching game: %w", err)
	}
	if len(games) == 0 {
		return nil, fmt.Errorf("game not found")
	}
	game := games[0]

	data := &domain.ImportData{
		Name:       game.Name,
		ExternalID: externalID,
		Attributes: make(map[string]any),
	}

	if desc := pickDescription(game.Summary, game.Storyline); desc != "" {
		data.Description = &desc
	}
	if game.Cover.ImageID != "" {
		img := coverURL(game.Cover.ImageID, "t_cover_big")
		data.ImageURL = &img
	}

	releaseDate := selectReleaseDate(game, platformID)
	if releaseDate != "" {
		data.Attributes["games.release_date"] = releaseDate
		if year, err := strconv.Atoi(releaseDate[:4]); err == nil {
			data.Attributes["games.year_released"] = year
		}
	}

	if platformName := selectPlatformName(game.Platforms, platformID); platformName != "" {
		data.Attributes["games.platform"] = platformName
	}

	if v := joinNamed(game.Genres); v != "" {
		data.Attributes["games.genres"] = v
	}
	if v := joinNamed(game.Themes); v != "" {
		data.Attributes["games.themes"] = v
	}
	if v := joinNamed(game.GameModes); v != "" {
		data.Attributes["games.game_modes"] = v
	}
	if v := joinNamed(game.PlayerPerspectives); v != "" {
		data.Attributes["games.player_perspectives"] = v
	}
	if v := joinNamed(game.Franchises); v != "" {
		data.Attributes["games.franchise"] = v
	}
	if v := joinNamed(game.GameEngines); v != "" {
		data.Attributes["games.engine"] = v
	}

	developers, publishers := splitCompanies(game.InvolvedCompanies)
	if developers != "" {
		data.Attributes["games.developers"] = developers
	}
	if publishers != "" {
		data.Attributes["games.publishers"] = publishers
	}

	if rating := formatAgeRatings(game.AgeRatings); rating != "" {
		data.Attributes["games.age_rating"] = rating
	}

	if label := categoryLabel(game.Category); label != "" {
		data.Attributes["games.category"] = label
	}
	if label := statusLabel(game.Status); label != "" {
		data.Attributes["games.status"] = label
	}

	if game.Rating > 0 {
		// IGDB returns rating as a 0-100 float; round to one decimal for readability.
		data.Attributes["games.rating"] = roundTo(game.Rating, 1)
	}
	if game.URL != "" {
		data.Attributes["games.url"] = game.URL
	}

	return data, nil
}

// Private helpers

// post executes an Apicalypse query against IGDB and decodes the JSON result.
// It transparently refreshes the auth token once on a 401.
func (p *Plugin) post(ctx context.Context, endpoint, body string, result any) error {
	resp, err := p.doRequest(ctx, endpoint, body, false)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Token may have been revoked or rotated; force-refresh and retry once.
		resp.Body.Close()
		p.tokens.Invalidate()
		resp, err = p.doRequest(ctx, endpoint, body, true)
		if err != nil {
			return err
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("IGDB API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(preview)))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func (p *Plugin) doRequest(ctx context.Context, endpoint, body string, isRetry bool) (*http.Response, error) {
	token, err := p.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}

	clientID, _ := credentials()

	p.waitForRateLimit()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+endpoint, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if isRetry {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

func (p *Plugin) waitForRateLimit() {
	p.rateMu.Lock()
	defer p.rateMu.Unlock()

	elapsed := time.Since(p.lastRequest)
	if elapsed < rateLimitWindow {
		time.Sleep(rateLimitWindow - elapsed)
	}
	p.lastRequest = time.Now()
}

// expandSearchResult turns one game into one row per relevant platform, capped
// at maxPlatformsPerGame and sorted so the most recent releases come first.
func (p *Plugin) expandSearchResult(g searchGame) []domain.SearchResult {
	year := extractYear(g.FirstReleaseDate)

	base := domain.SearchResult{
		Title: g.Name,
	}
	if g.Cover.ImageID != "" {
		img := coverURL(g.Cover.ImageID, "t_cover_small")
		base.ImageURL = &img
	}

	if len(g.Platforms) == 0 {
		// No platform info – fall back to a single row keyed by game ID only.
		base.ExternalID = strconv.FormatInt(g.ID, 10)
		base.Subtitle = formatSubtitle("", year)
		return []domain.SearchResult{base}
	}

	platforms := selectTopPlatforms(g)

	results := make([]domain.SearchResult, 0, len(platforms))
	for _, plat := range platforms {
		row := base
		row.ExternalID = fmt.Sprintf("%d:%d", g.ID, plat.ID)
		platformYear := year
		if plat.ReleaseUnix > 0 {
			platformYear = extractYear(plat.ReleaseUnix)
		}
		row.Subtitle = formatSubtitle(plat.Name, platformYear)
		results = append(results, row)
	}
	return results
}

// selectTopPlatforms picks up to maxPlatformsPerGame platforms, preferring those
// with the most recent release date for this game.
func selectTopPlatforms(g searchGame) []platformWithRelease {
	releaseByPlatform := make(map[int64]int64, len(g.ReleaseDates))
	for _, rd := range g.ReleaseDates {
		if rd.Date > releaseByPlatform[rd.Platform] {
			releaseByPlatform[rd.Platform] = rd.Date
		}
	}

	enriched := make([]platformWithRelease, 0, len(g.Platforms))
	for _, plat := range g.Platforms {
		enriched = append(enriched, platformWithRelease{
			ID:          plat.ID,
			Name:        plat.Name,
			ReleaseUnix: releaseByPlatform[plat.ID],
		})
	}

	sort.SliceStable(enriched, func(i, j int) bool {
		if enriched[i].ReleaseUnix != enriched[j].ReleaseUnix {
			return enriched[i].ReleaseUnix > enriched[j].ReleaseUnix
		}
		return enriched[i].Name < enriched[j].Name
	})

	if len(enriched) > maxPlatformsPerGame {
		enriched = enriched[:maxPlatformsPerGame]
	}
	return enriched
}

func parseExternalID(externalID string) (int64, int64, error) {
	parts := strings.SplitN(externalID, ":", 2)
	gameID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || gameID <= 0 {
		return 0, 0, fmt.Errorf("invalid external_id: %q", externalID)
	}
	if len(parts) == 1 || parts[1] == "" {
		return gameID, 0, nil
	}
	platformID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || platformID <= 0 {
		return 0, 0, fmt.Errorf("invalid platform id in external_id: %q", externalID)
	}
	return gameID, platformID, nil
}

func selectReleaseDate(game fetchGame, platformID int64) string {
	if platformID > 0 {
		for _, rd := range game.ReleaseDates {
			if rd.Platform == platformID && rd.Date > 0 {
				return formatDate(rd.Date)
			}
		}
	}
	if game.FirstReleaseDate > 0 {
		return formatDate(game.FirstReleaseDate)
	}
	return ""
}

func selectPlatformName(platforms []namedRef, platformID int64) string {
	if platformID == 0 {
		return ""
	}
	for _, p := range platforms {
		if p.ID == platformID {
			return p.Name
		}
	}
	return ""
}

func joinNamed[T namedItem](items []T) string {
	if len(items) == 0 {
		return ""
	}
	names := make([]string, 0, len(items))
	for _, it := range items {
		if n := it.GetName(); n != "" {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

func splitCompanies(items []involvedCompany) (string, string) {
	var devs, pubs []string
	for _, c := range items {
		name := c.Company.Name
		if name == "" {
			continue
		}
		if c.Developer {
			devs = append(devs, name)
		}
		if c.Publisher {
			pubs = append(pubs, name)
		}
	}
	return strings.Join(devs, ", "), strings.Join(pubs, ", ")
}

func formatAgeRatings(ratings []ageRating) string {
	if len(ratings) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ratings))
	for _, r := range ratings {
		org := ageRatingOrg(r.Category)
		val := ageRatingValue(r.Category, r.Rating)
		if org == "" || val == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", org, val))
	}
	return strings.Join(parts, ", ")
}

func ageRatingOrg(code int) string {
	switch code {
	case 1:
		return "ESRB"
	case 2:
		return "PEGI"
	case 3:
		return "CERO"
	case 4:
		return "USK"
	case 5:
		return "GRAC"
	case 6:
		return "CLASS_IND"
	case 7:
		return "ACB"
	}
	return ""
}

func ageRatingValue(orgCode, rating int) string {
	// IGDB encodes ratings as a single shared enum; this maps the common ones.
	// See https://api-docs.igdb.com/#age-rating-enums.
	switch rating {
	case 1:
		return "3"
	case 2:
		return "7"
	case 3:
		return "12"
	case 4:
		return "16"
	case 5:
		return "18"
	case 6:
		return "RP"
	case 7:
		return "EC"
	case 8:
		return "E"
	case 9:
		return "E10+"
	case 10:
		return "T"
	case 11:
		return "M"
	case 12:
		return "AO"
	}
	_ = orgCode
	return ""
}

func categoryLabel(code int) string {
	switch code {
	case 0:
		return "Main Game"
	case 1:
		return "DLC / Add-on"
	case 2:
		return "Expansion"
	case 3:
		return "Bundle"
	case 4:
		return "Standalone Expansion"
	case 5:
		return "Mod"
	case 6:
		return "Episode"
	case 7:
		return "Season"
	case 8:
		return "Remake"
	case 9:
		return "Remaster"
	case 10:
		return "Expanded Game"
	case 11:
		return "Port"
	case 12:
		return "Fork"
	case 13:
		return "Pack"
	case 14:
		return "Update"
	}
	return ""
}

func statusLabel(code int) string {
	switch code {
	case 0:
		return "Released"
	case 2:
		return "Alpha"
	case 3:
		return "Beta"
	case 4:
		return "Early Access"
	case 5:
		return "Offline"
	case 6:
		return "Cancelled"
	case 7:
		return "Rumored"
	case 8:
		return "Delisted"
	}
	return ""
}

func pickDescription(summary, storyline string) string {
	if summary != "" {
		return summary
	}
	return storyline
}

func coverURL(imageID, size string) string {
	return fmt.Sprintf("%s/%s/%s.jpg", imageBaseURL, size, imageID)
}

func formatDate(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}

func extractYear(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format("2006")
}

func formatSubtitle(platform, year string) string {
	switch {
	case platform != "" && year != "":
		return fmt.Sprintf("%s (%s)", platform, year)
	case platform != "":
		return platform
	case year != "":
		return fmt.Sprintf("(%s)", year)
	}
	return ""
}

func roundTo(value float64, decimals int) float64 {
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	return float64(int64(value*mult+0.5)) / mult
}

// IGDB response types

type namedItem interface {
	GetName() string
}

type namedRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (n namedRef) GetName() string { return n.Name }

type platformWithRelease struct {
	ID          int64
	Name        string
	ReleaseUnix int64
}

type cover struct {
	ImageID string `json:"image_id"`
}

type releaseDate struct {
	Platform int64 `json:"platform"`
	Date     int64 `json:"date"`
}

type involvedCompany struct {
	Developer bool `json:"developer"`
	Publisher bool `json:"publisher"`
	Company   struct {
		Name string `json:"name"`
	} `json:"company"`
}

type ageRating struct {
	Category int `json:"category"`
	Rating   int `json:"rating"`
}

type searchGame struct {
	ID               int64         `json:"id"`
	Name             string        `json:"name"`
	FirstReleaseDate int64         `json:"first_release_date"`
	Cover            cover         `json:"cover"`
	Platforms        []namedRef    `json:"platforms"`
	ReleaseDates     []releaseDate `json:"release_dates"`
}

type fetchGame struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	Summary            string            `json:"summary"`
	Storyline          string            `json:"storyline"`
	FirstReleaseDate   int64             `json:"first_release_date"`
	Cover              cover             `json:"cover"`
	Platforms          []namedRef        `json:"platforms"`
	Genres             []namedRef        `json:"genres"`
	Themes             []namedRef        `json:"themes"`
	GameModes          []namedRef        `json:"game_modes"`
	PlayerPerspectives []namedRef        `json:"player_perspectives"`
	Franchises         []namedRef        `json:"franchises"`
	GameEngines        []namedRef        `json:"game_engines"`
	InvolvedCompanies  []involvedCompany `json:"involved_companies"`
	AgeRatings         []ageRating       `json:"age_ratings"`
	Rating             float64           `json:"rating"`
	Status             int               `json:"status"`
	Category           int               `json:"category"`
	URL                string            `json:"url"`
	ReleaseDates       []releaseDate     `json:"release_dates"`
}
