# RFC 002: Google Books Plugin

> **Status**: Implemented
> **Created**: 2026-01-12
> **Author**: @lmmendes

---

## Summary

The Google Books plugin enables users to import books from the Google Books API. It automatically creates and manages a "Books" category with namespaced attributes for book metadata.

---

## Plugin Details

| Property | Value |
|----------|-------|
| **Plugin ID** | `google_books` |
| **Display Name** | Google Books |
| **Category** | Books |
| **API** | Google Books API v1 |
| **Base URL** | `https://www.googleapis.com/books/v1/volumes` |
| **Authentication** | None required (free public API) |

---

## Attributes

The plugin defines the following namespaced attributes for the Books category:

| Key | Display Name | Data Type | Required |
|-----|--------------|-----------|----------|
| `books.isbn` | ISBN | string | No |
| `books.author` | Author | string | No |
| `books.publisher` | Publisher | string | No |
| `books.published_date` | Published Date | string | No |
| `books.page_count` | Page Count | number | No |
| `books.language` | Language | string | No |
| `books.categories` | Categories | string | No |

---

## Search Fields

Users can search for books using the following fields:

| Field Key | Label | API Mapping |
|-----------|-------|-------------|
| `title` | Title | `intitle:{query}` |
| `isbn` | ISBN | `isbn:{query}` |
| `author` | Author | `inauthor:{query}` |

---

## API Integration

### Search Request

```
GET https://www.googleapis.com/books/v1/volumes?q={search_query}&maxResults={limit}&printType=books
```

**Query Format:**
- Title search: `intitle:pragmatic programmer`
- ISBN search: `isbn:9780135957059`
- Author search: `inauthor:david thomas`

### Search Response Mapping

| Google Books Field | Maps To |
|-------------------|---------|
| `id` | `ExternalID` |
| `volumeInfo.title` | `Title` |
| `volumeInfo.authors[]` + `volumeInfo.publishedDate` | `Subtitle` |
| `volumeInfo.imageLinks.thumbnail` | `ImageURL` |

### Fetch Request

```
GET https://www.googleapis.com/books/v1/volumes/{externalID}
```

### Fetch Response Mapping

| Google Books Field | Maps To |
|-------------------|---------|
| `volumeInfo.title` | Asset Name |
| `volumeInfo.description` | Asset Description |
| `volumeInfo.imageLinks.large/medium/thumbnail` | Asset Image |
| `volumeInfo.industryIdentifiers[].identifier` | `books.isbn` |
| `volumeInfo.authors[]` | `books.author` |
| `volumeInfo.publisher` | `books.publisher` |
| `volumeInfo.publishedDate` | `books.published_date` |
| `volumeInfo.pageCount` | `books.page_count` |
| `volumeInfo.language` | `books.language` |
| `volumeInfo.categories[]` | `books.categories` |

---

## Implementation Details

### ISBN Preference

When multiple industry identifiers are available, the plugin prefers:
1. ISBN-13 (type: `ISBN_13`)
2. ISBN-10 (type: `ISBN_10`) as fallback

### Image URL Handling

- All image URLs are converted from HTTP to HTTPS
- Image preference order: large > medium > thumbnail
- Thumbnails use `w185` size for search results

### Author Formatting

Multiple authors are joined with commas: `"David Thomas, Andrew Hunt"`

### Category Formatting

Multiple categories are joined with commas: `"Programming, Software Development"`

---

## Configuration

No configuration is required. The Google Books API is free and does not require authentication for basic search operations.

---

## Rate Limits

The Google Books API has the following limits:
- 1,000 requests per day (without API key)
- 100 requests per 100 seconds per user

For higher limits, an API key can be configured via environment variable (future enhancement).

---

## Example Usage

### Search by Title

```
GET /api/plugins/google_books/search?field=title&q=pragmatic%20programmer&limit=10
```

**Response:**
```json
{
  "results": [
    {
      "external_id": "LYoQDgAAQBAJ",
      "title": "The Pragmatic Programmer",
      "subtitle": "David Thomas, Andrew Hunt (2019)",
      "image_url": "https://books.google.com/books/..."
    }
  ]
}
```

### Import Book

```
POST /api/plugins/google_books/import
{
  "external_id": "LYoQDgAAQBAJ"
}
```

**Response:**
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "The Pragmatic Programmer",
    "category_id": "uuid-of-books-category",
    "attributes": {
      "books.isbn": "978-0135957059",
      "books.author": "David Thomas, Andrew Hunt",
      "books.publisher": "Addison-Wesley Professional",
      "books.published_date": "2019-09-23",
      "books.page_count": 352,
      "books.language": "en",
      "books.categories": "Computers"
    },
    "import_plugin_id": "google_books",
    "import_external_id": "LYoQDgAAQBAJ"
  }
}
```

---

## Files

| File | Description |
|------|-------------|
| `backend/internal/plugin/googlebooks/googlebooks.go` | Plugin implementation |

---

## Related RFCs

- [RFC 001: Import Plugins](./001-import-plugins.md) - Plugin system architecture
