# RFC 003: TMDB Plugin (Movies & TV Series)

> **Status**: Implemented
> **Created**: 2026-01-12
> **Author**: @lmmendes

---

## Summary

The TMDB plugin provides integration with The Movie Database (TMDB) API, enabling users to import movies and TV series with rich metadata. This plugin is implemented as two separate sub-plugins that share a common client infrastructure.

---

## Plugin Details

### Movies Plugin

| Property | Value |
|----------|-------|
| **Plugin ID** | `tmdb_movies` |
| **Display Name** | TMDB Movies |
| **Category** | Movies |
| **API Endpoint** | `/search/movie`, `/movie/{id}` |

### TV Series Plugin

| Property | Value |
|----------|-------|
| **Plugin ID** | `tmdb_series` |
| **Display Name** | TMDB TV Series |
| **Category** | TV Series |
| **API Endpoint** | `/search/tv`, `/tv/{id}` |

### Common Configuration

| Property | Value |
|----------|-------|
| **API** | TMDB API v3 |
| **Base URL** | `https://api.themoviedb.org/3` |
| **Image Base URL** | `https://image.tmdb.org/t/p` |
| **Authentication** | Bearer token (API key required) |

---

## Authentication

The TMDB API requires an API key for all requests. The key can be configured via:

1. **Environment Variable** (preferred): `TMDB_API_KEY`
2. **Build-time Injection**: `-ldflags="-X github.com/mendelui/attic/internal/plugin/tmdb.APIKey=your-key"`

All requests include the header:
```
Authorization: Bearer {api_key}
```

---

## Movies Attributes

The Movies plugin defines the following namespaced attributes:

| Key | Display Name | Data Type | Required |
|-----|--------------|-----------|----------|
| `movies.release_date` | Release Date | date | No |
| `movies.genres` | Genres | string | No |
| `movies.rating` | Rating | number | No |
| `movies.runtime` | Runtime (minutes) | number | No |
| `movies.language` | Original Language | string | No |
| `movies.status` | Status | string | No |
| `movies.tagline` | Tagline | string | No |
| `movies.budget` | Budget | number | No |
| `movies.revenue` | Revenue | number | No |

### Movies Search Fields

| Field Key | Label |
|-----------|-------|
| `title` | Title |

---

## TV Series Attributes

The TV Series plugin defines the following namespaced attributes:

| Key | Display Name | Data Type | Required |
|-----|--------------|-----------|----------|
| `series.first_air_date` | First Air Date | date | No |
| `series.last_air_date` | Last Air Date | date | No |
| `series.genres` | Genres | string | No |
| `series.rating` | Rating | number | No |
| `series.seasons` | Number of Seasons | number | No |
| `series.episodes` | Number of Episodes | number | No |
| `series.language` | Original Language | string | No |
| `series.status` | Status | string | No |

### TV Series Search Fields

| Field Key | Label |
|-----------|-------|
| `name` | Title |

---

## API Integration

### Movies Search Request

```
GET https://api.themoviedb.org/3/search/movie?query={query}&include_adult=false
```

### Movies Fetch Request

```
GET https://api.themoviedb.org/3/movie/{id}
```

### TV Series Search Request

```
GET https://api.themoviedb.org/3/search/tv?query={query}&include_adult=false
```

### TV Series Fetch Request

```
GET https://api.themoviedb.org/3/tv/{id}
```

---

## Response Mapping

### Movies Search Response

| TMDB Field | Maps To |
|------------|---------|
| `id` | `ExternalID` |
| `title` | `Title` |
| `release_date` (year) + `vote_average` | `Subtitle` |
| `poster_path` | `ImageURL` (w185 size) |

### Movies Fetch Response

| TMDB Field | Maps To |
|------------|---------|
| `title` | Asset Name |
| `overview` | Asset Description |
| `poster_path` | Asset Image (w500 size) |
| `release_date` | `movies.release_date` |
| `genres[].name` | `movies.genres` |
| `vote_average` | `movies.rating` |
| `runtime` | `movies.runtime` |
| `original_language` | `movies.language` |
| `status` | `movies.status` |
| `tagline` | `movies.tagline` |
| `budget` | `movies.budget` |
| `revenue` | `movies.revenue` |

### TV Series Search Response

| TMDB Field | Maps To |
|------------|---------|
| `id` | `ExternalID` |
| `name` | `Title` |
| `first_air_date` (year) + `vote_average` | `Subtitle` |
| `poster_path` | `ImageURL` (w185 size) |

### TV Series Fetch Response

| TMDB Field | Maps To |
|------------|---------|
| `name` | Asset Name |
| `overview` | Asset Description |
| `poster_path` | Asset Image (w500 size) |
| `first_air_date` | `series.first_air_date` |
| `last_air_date` | `series.last_air_date` |
| `genres[].name` | `series.genres` |
| `vote_average` | `series.rating` |
| `number_of_seasons` | `series.seasons` |
| `number_of_episodes` | `series.episodes` |
| `original_language` | `series.language` |
| `status` | `series.status` |

---

## Image Handling

### Poster Sizes

| Size | Usage |
|------|-------|
| `w185` | Search result thumbnails |
| `w500` | Full asset images |

### URL Construction

```
https://image.tmdb.org/t/p/{size}{poster_path}
```

Example:
```
https://image.tmdb.org/t/p/w500/abc123.jpg
```

---

## Genre Formatting

Genres are returned as an array of objects and formatted as a comma-separated string:

**Input:**
```json
[{"id": 28, "name": "Action"}, {"id": 12, "name": "Adventure"}]
```

**Output:**
```
"Action, Adventure"
```

---

## Subtitle Formatting

Search result subtitles combine year and rating:

**Movies:**
```
(2024) ★ 8.5
```

**TV Series:**
```
(2019) ★ 9.2
```

---

## Rate Limits

TMDB API rate limits:
- 50 requests per second per API key
- Results are limited to 20 items per page (plugin limits to 10 by default)

---

## Example Usage

### Search for Movies

```
GET /api/plugins/tmdb_movies/search?field=title&q=inception&limit=10
```

**Response:**
```json
{
  "results": [
    {
      "external_id": "27205",
      "title": "Inception",
      "subtitle": "(2010) ★ 8.4",
      "image_url": "https://image.tmdb.org/t/p/w185/..."
    }
  ]
}
```

### Import Movie

```
POST /api/plugins/tmdb_movies/import
{
  "external_id": "27205"
}
```

**Response:**
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "Inception",
    "category_id": "uuid-of-movies-category",
    "attributes": {
      "movies.release_date": "2010-07-16",
      "movies.genres": "Action, Science Fiction, Adventure",
      "movies.rating": 8.369,
      "movies.runtime": 148,
      "movies.language": "en",
      "movies.status": "Released",
      "movies.tagline": "Your mind is the scene of the crime.",
      "movies.budget": 160000000,
      "movies.revenue": 825532764
    },
    "import_plugin_id": "tmdb_movies",
    "import_external_id": "27205"
  }
}
```

### Search for TV Series

```
GET /api/plugins/tmdb_series/search?field=name&q=breaking%20bad&limit=10
```

**Response:**
```json
{
  "results": [
    {
      "external_id": "1396",
      "title": "Breaking Bad",
      "subtitle": "(2008) ★ 8.9",
      "image_url": "https://image.tmdb.org/t/p/w185/..."
    }
  ]
}
```

### Import TV Series

```
POST /api/plugins/tmdb_series/import
{
  "external_id": "1396"
}
```

**Response:**
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "Breaking Bad",
    "category_id": "uuid-of-series-category",
    "attributes": {
      "series.first_air_date": "2008-01-20",
      "series.last_air_date": "2013-09-29",
      "series.genres": "Drama, Crime",
      "series.rating": 8.9,
      "series.seasons": 5,
      "series.episodes": 62,
      "series.language": "en",
      "series.status": "Ended"
    },
    "import_plugin_id": "tmdb_series",
    "import_external_id": "1396"
  }
}
```

---

## Architecture

### Shared Components

The TMDB plugins share a common client (`Client`) that handles:
- API authentication (Bearer token)
- HTTP request construction
- JSON response parsing
- Error handling

### File Structure

```
backend/internal/plugin/tmdb/
├── common.go   # Shared client and types (Client, Genre, GetPosterURL, etc.)
├── movies.go   # MoviesPlugin implementation
└── series.go   # SeriesPlugin implementation
```

---

## Files

| File | Description |
|------|-------------|
| `backend/internal/plugin/tmdb/common.go` | Shared TMDB client and common types |
| `backend/internal/plugin/tmdb/movies.go` | Movies plugin implementation |
| `backend/internal/plugin/tmdb/series.go` | TV Series plugin implementation |

---

## Related RFCs

- [RFC 001: Import Plugins](./001-import-plugins.md) - Plugin system architecture
- [RFC 002: Google Books Plugin](./002-google-books-plugin.md) - Google Books plugin
