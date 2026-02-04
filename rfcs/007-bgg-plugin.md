# RFC 007: BoardGameGeek (BGG) Plugin

> **Status**: Implemented
> **Created**: 2026-01-19
> **Author**: @lmmendes

---

## Summary

The BGG plugin enables users to import board games from the BoardGameGeek XML API2. It automatically creates and manages a "Board Games" category with namespaced attributes for board game metadata.

---

## Plugin Details

| Property | Value |
|----------|-------|
| **Plugin ID** | `bgg_boardgames` |
| **Display Name** | BoardGameGeek |
| **Category** | Board Games |
| **API** | BGG XML API2 |
| **Base URL** | `https://boardgamegeek.com/xmlapi2` |
| **Authentication** | Bearer token (API key required) |

---

## Authentication

The BGG API requires an application token for all requests. The key can be configured via:

1. **Environment Variable** (preferred): `ATTIC_BGG_API_KEY`
2. **Build-time Injection**: `-ldflags="-X github.com/lmmendes/attic/internal/plugin/bgg.APIKey=your-key"`

All requests include the header:
```
Authorization: Bearer {api_key}
```

To obtain an API key:
1. Register your application at https://boardgamegeek.com/applications
2. Once approved, create a token via the applications dashboard

---

## Attributes

The plugin defines the following namespaced attributes for the Board Games category:

| Key | Display Name | Data Type | Required |
|-----|--------------|-----------|----------|
| `boardgames.year_published` | Year Published | number | No |
| `boardgames.min_players` | Min Players | number | No |
| `boardgames.max_players` | Max Players | number | No |
| `boardgames.playing_time` | Playing Time (min) | number | No |
| `boardgames.min_playtime` | Min Playtime (min) | number | No |
| `boardgames.max_playtime` | Max Playtime (min) | number | No |
| `boardgames.min_age` | Minimum Age | number | No |
| `boardgames.rating` | BGG Rating | number | No |
| `boardgames.weight` | Complexity/Weight | number | No |
| `boardgames.designers` | Designers | string | No |
| `boardgames.publishers` | Publishers | string | No |
| `boardgames.categories` | Categories | string | No |
| `boardgames.mechanics` | Mechanics | string | No |

---

## Search Fields

| Field Key | Label | API Mapping |
|-----------|-------|-------------|
| `name` | Name | `?query={query}&type=boardgame` |

---

## API Integration

### Search Request

```
GET https://boardgamegeek.com/xmlapi2/search?query={query}&type=boardgame
```

**Response XML:**
```xml
<items total="123">
  <item type="boardgame" id="174430">
    <name type="primary" value="Gloomhaven"/>
    <yearpublished value="2017"/>
  </item>
</items>
```

### Search Response Mapping

| BGG Field | Maps To |
|-----------|---------|
| `item[@id]` | `ExternalID` |
| `name[@value]` | `Title` |
| `yearpublished[@value]` | `Subtitle` (formatted as "(year)") |

### Fetch Request

```
GET https://boardgamegeek.com/xmlapi2/thing?id={externalID}&stats=1
```

### Fetch Response Mapping

| BGG Field | Maps To |
|-----------|---------|
| `name[@type='primary'][@value]` | Asset Name |
| `description` | Asset Description |
| `image` or `thumbnail` | Asset Image |
| `yearpublished[@value]` | `boardgames.year_published` |
| `minplayers[@value]` | `boardgames.min_players` |
| `maxplayers[@value]` | `boardgames.max_players` |
| `playingtime[@value]` | `boardgames.playing_time` |
| `minplaytime[@value]` | `boardgames.min_playtime` |
| `maxplaytime[@value]` | `boardgames.max_playtime` |
| `minage[@value]` | `boardgames.min_age` |
| `statistics/ratings/average[@value]` | `boardgames.rating` |
| `statistics/ratings/averageweight[@value]` | `boardgames.weight` |
| `link[@type='boardgamedesigner']` | `boardgames.designers` |
| `link[@type='boardgamepublisher']` | `boardgames.publishers` |
| `link[@type='boardgamecategory']` | `boardgames.categories` |
| `link[@type='boardgamemechanic']` | `boardgames.mechanics` |

---

## Implementation Details

### XML Parsing

Unlike other plugins (Google Books, TMDB) which use JSON, the BGG API returns XML responses. The plugin uses Go's `encoding/xml` package for parsing.

### Rate Limiting

The BGG API recommends a 5-second wait between requests. The plugin implements rate limiting with a mutex-protected timestamp to ensure compliance.

### Image URL Handling

BGG images are served from `https://cf.geekdo-images.com/`. URLs are returned directly in the API response without modification needed.

### Description Cleaning

Game descriptions may contain HTML entities. The plugin replaces common entities (`&#10;`, `&amp;`, etc.) with their text equivalents.

### Link Formatting

Multiple designers, publishers, categories, and mechanics are joined with commas: `"Isaac Childres"` or `"Strategy, Adventure"`.

---

## Rate Limits

BGG API recommendations:
- ~5 seconds between requests
- Exceeding limits returns 500 or 503 status codes

---

## Example Usage

### Search for Board Games

```
GET /api/plugins/bgg_boardgames/search?field=name&q=gloomhaven&limit=10
```

**Response:**
```json
{
  "results": [
    {
      "external_id": "174430",
      "title": "Gloomhaven",
      "subtitle": "(2017)",
      "image_url": null
    }
  ]
}
```

### Import Board Game

```
POST /api/plugins/bgg_boardgames/import
{
  "external_id": "174430"
}
```

**Response:**
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "Gloomhaven",
    "category_id": "uuid-of-boardgames-category",
    "attributes": {
      "boardgames.year_published": 2017,
      "boardgames.min_players": 1,
      "boardgames.max_players": 4,
      "boardgames.playing_time": 120,
      "boardgames.min_playtime": 60,
      "boardgames.max_playtime": 120,
      "boardgames.min_age": 14,
      "boardgames.rating": 8.5,
      "boardgames.weight": 3.86,
      "boardgames.designers": "Isaac Childres",
      "boardgames.publishers": "Cephalofair Games",
      "boardgames.categories": "Adventure, Exploration, Fantasy, Fighting, Miniatures",
      "boardgames.mechanics": "Action Queue, Campaign / Battle Card Driven, ..."
    },
    "import_plugin_id": "bgg_boardgames",
    "import_external_id": "174430"
  }
}
```

---

## Files

| File | Description |
|------|-------------|
| `backend/internal/plugin/bgg/bgg.go` | Plugin implementation |

---

## Related RFCs

- [RFC 001: Import Plugins](./001-import-plugins.md) - Plugin system architecture
- [RFC 002: Google Books Plugin](./002-google-books-plugin.md) - Google Books plugin
- [RFC 003: TMDB Plugin](./003-tmdb-plugin.md) - TMDB Movies & TV Series plugin
