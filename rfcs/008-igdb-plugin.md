# RFC 008: IGDB Plugin

> **Status**: Implemented
> **Created**: 2026-06-27
> **Author**: @lmmendes

---

## Summary

The IGDB plugin enables users to import video games from the [Internet Game Database (IGDB)](https://www.igdb.com/api). It automatically creates and manages a "Video Games" category with namespaced attributes for game metadata, and lets users pick the specific platform release (PS5, Xbox, PC, etc.) that matches their physical or digital copy.

---

## Plugin Details

| Property | Value |
|----------|-------|
| **Plugin ID** | `igdb_games` |
| **Display Name** | IGDB |
| **Category** | Video Games |
| **API** | IGDB REST API v4 |
| **Base URL** | `https://api.igdb.com/v4` |
| **Authentication** | Twitch OAuth (Client-ID + Bearer token) |

---

## Authentication

Unlike the other plugins (which use a single bearer token), IGDB is owned by Twitch and requires Twitch app credentials. The plugin needs **two** values:

| Environment variable | Description |
|---|---|
| `ATTIC_IGDB_CLIENT_ID` | Twitch application Client-ID |
| `ATTIC_IGDB_CLIENT_SECRET` | Twitch application Client Secret |

Build-time injection is also supported:
```
-ldflags="-X github.com/lmmendes/attic/internal/plugin/igdb.ClientID=... \
          -X github.com/lmmendes/attic/internal/plugin/igdb.ClientSecret=..."
```

### Token lifecycle

The plugin exchanges the credentials for a Bearer token via:
```
POST https://id.twitch.tv/oauth2/token?client_id=...&client_secret=...&grant_type=client_credentials
```

Tokens are cached in memory (per-process) and refreshed automatically:
- Before any request that would land inside a 5-minute leeway of expiry.
- After receiving a `401 Unauthorized` from the IGDB API (token revoked / rotated). The plugin invalidates the cache and retries the request **once** with a fresh token.

### Obtaining credentials

1. Register an application at https://dev.twitch.tv/console/apps.
2. Choose any OAuth Redirect URL (it isn't used; client-credentials flow doesn't require redirect).
3. Copy the generated Client-ID + Client Secret into the environment variables above.

---

## Category & Attributes

The plugin manages a single category called **Video Games**, with all attributes namespaced under `games.*`:

| Key | Display Name | Data Type | Required |
|-----|--------------|-----------|----------|
| `games.platform` | Platform | string | No |
| `games.release_date` | Release Date | date | No |
| `games.year_released` | Year Released | number | No |
| `games.genres` | Genres | string | No |
| `games.themes` | Themes | string | No |
| `games.developers` | Developers | string | No |
| `games.publishers` | Publishers | string | No |
| `games.game_modes` | Game Modes | string | No |
| `games.player_perspectives` | Player Perspective | string | No |
| `games.franchise` | Franchise | string | No |
| `games.engine` | Game Engine | string | No |
| `games.age_rating` | Age Rating | string | No |
| `games.category` | Edition Type | string | No |
| `games.status` | Release Status | string | No |
| `games.rating` | IGDB Rating | number | No |
| `games.url` | IGDB URL | string | No |

`games.platform` is **singular** — each imported asset represents a single platform release.

---

## Search Fields

| Field Key | Label | API Mapping |
|-----------|-------|-------------|
| `name` | Name | Apicalypse `search "<query>";` |

---

## Platform Selection via Composite External IDs

A game like *Cyberpunk 2077* exists on PS5, Xbox Series X, PC, etc., and each platform release is effectively a different asset in a collection. To let the user pick the right one **without** changing the `ImportPlugin` interface (which only accepts `external_id`), the plugin uses a composite external ID:

```
<game_id>:<platform_id>
```

| Step | Behavior |
|---|---|
| **Search** | One row per `(game, platform)` combination. Capped at 6 platforms per game, sorted by release date descending. Subtitle is rendered as `Platform Name (Year)`. |
| **Fetch** | The composite ID is parsed back. The plugin selects the platform-specific release date when available, and writes the platform name into `games.platform`. |

Games with no platform metadata fall back to a single search row with just `<game_id>` as the external ID.

### Why cap at 6 platforms?

Some games (e.g. Tetris) have 50+ platform releases. Without a cap, search results become unusable. We surface the 6 most recent releases — typically the platforms users actually own physical/digital copies for.

---

## API Integration

### Authentication query

All API requests include:
```
Client-ID: <client_id>
Authorization: Bearer <access_token>
Accept: application/json
Content-Type: text/plain
```

### Search

```
POST /games
search "<query>";
fields name, first_release_date, cover.image_id,
       platforms.id, platforms.name,
       release_dates.platform, release_dates.date;
limit 10;
```

### Fetch

```
POST /games
fields name, summary, storyline, first_release_date, cover.image_id,
       platforms.name, genres.name, themes.name,
       game_modes.name, player_perspectives.name,
       involved_companies.developer, involved_companies.publisher, involved_companies.company.name,
       rating, age_ratings.category, age_ratings.rating,
       franchises.name, game_engines.name,
       status, category, url,
       release_dates.platform, release_dates.date;
where id = <game_id>;
limit 1;
```

---

## Implementation Details

### Apicalypse Query Format

IGDB uses its custom **Apicalypse** query language (sent as the POST body, content-type `text/plain`). The plugin builds these queries as plain strings rather than using a generated query builder — the surface area is small enough that this is simpler and easier to read.

### Image URLs

IGDB cover URLs follow the pattern:
```
https://images.igdb.com/igdb/image/upload/<size>/<image_id>.jpg
```

The plugin uses:
- `t_cover_small` for search thumbnails
- `t_cover_big` for the imported asset image (downloaded and stored by the existing import handler)

### Date Handling

IGDB returns dates as Unix timestamps. The plugin converts them to UTC `YYYY-MM-DD` strings (compatible with the `date` attribute type) and additionally extracts the year as a numeric `games.year_released` attribute for easy filtering.

### Description Selection

`summary` is preferred over `storyline` because the latter often contains plot spoilers.

### Enum Mapping

IGDB returns enum codes for `category`, `status`, and `age_ratings`. The plugin maps the common codes to human-readable labels (Main Game, DLC, Released, Cancelled, ESRB, PEGI, M, T, etc.). Unknown codes produce no attribute value, so future IGDB additions degrade gracefully.

### Rate Limiting

IGDB allows ~4 requests/sec. The plugin enforces a 250 ms minimum between requests via a mutex-guarded timestamp.

---

## Example Usage

### Search for Games

```
GET /api/plugins/igdb_games/search?field=name&q=zelda+breath&limit=10
```

**Response:**
```json
{
  "results": [
    {
      "external_id": "7346:130",
      "title": "The Legend of Zelda: Breath of the Wild",
      "subtitle": "Nintendo Switch (2017)",
      "image_url": "https://images.igdb.com/igdb/image/upload/t_cover_small/co3p2d.jpg"
    },
    {
      "external_id": "7346:41",
      "title": "The Legend of Zelda: Breath of the Wild",
      "subtitle": "Wii U (2017)",
      "image_url": "https://images.igdb.com/igdb/image/upload/t_cover_small/co3p2d.jpg"
    }
  ]
}
```

### Import a Game

```
POST /api/plugins/igdb_games/import
{
  "external_id": "7346:130"
}
```

**Response:**
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "The Legend of Zelda: Breath of the Wild",
    "category_id": "uuid-of-video-games-category",
    "attributes": {
      "games.platform": "Nintendo Switch",
      "games.release_date": "2017-03-03",
      "games.year_released": 2017,
      "games.genres": "Role-playing (RPG), Adventure",
      "games.themes": "Action, Fantasy, Open world",
      "games.developers": "Nintendo EPD",
      "games.publishers": "Nintendo",
      "games.game_modes": "Single player",
      "games.player_perspectives": "Third person",
      "games.franchise": "The Legend of Zelda",
      "games.engine": "Havok",
      "games.age_rating": "ESRB: E10+, PEGI: 12",
      "games.category": "Main Game",
      "games.status": "Released",
      "games.rating": 91.2,
      "games.url": "https://www.igdb.com/games/the-legend-of-zelda-breath-of-the-wild"
    },
    "import_plugin_id": "igdb_games",
    "import_external_id": "7346:130"
  }
}
```

---

## Rate Limits

IGDB documents the following limits:
- ~4 requests/sec per IP
- Excessive requests return HTTP 429

The plugin enforces this from the client side with a 250 ms minimum interval between requests.

---

## Security Considerations

1. **Credentials**: Both Client-ID and Client Secret are read from environment variables (preferred) or build-time injection. The Client Secret is the more sensitive value — treat it like an API key.
2. **Token Caching**: Tokens are kept in memory only. They are never persisted to disk or logs.
3. **Input Validation**: External IDs are parsed defensively (positive integer game ID, optional positive integer platform ID).
4. **Image Downloads**: Cover images are downloaded and stored by the existing import handler, not hot-linked.

---

## Files

| File | Description |
|------|-------------|
| `backend/internal/plugin/igdb/auth.go` | Twitch OAuth client-credentials token management |
| `backend/internal/plugin/igdb/igdb.go` | Plugin implementation (search, fetch, mapping) |
| `backend/internal/plugin/igdb/igdb_test.go` | Unit tests for pure helpers |

---

## Future Work

- **Multiple language support**: IGDB supports localised game info via `language_supports` and `alternative_names` — could be a future opt-in field.
- **Screenshots & artwork**: Currently only the cover is imported. A future enhancement could attach additional screenshots as asset attachments.
- **Refresh / re-sync**: Re-fetch existing imported assets to pick up updated ratings or status changes.

---

## Related RFCs

- [RFC 001: Import Plugins](./001-import-plugins.md) – Plugin system architecture
- [RFC 002: Google Books Plugin](./002-google-books-plugin.md)
- [RFC 003: TMDB Plugin](./003-tmdb-plugin.md)
- [RFC 007: BGG Plugin](./007-bgg-plugin.md)
