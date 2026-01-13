# RFC 001: Import Plugins

> **Status**: Accepted
> **Created**: 2026-01-12
> **Author**: @lmmendes

---

## Summary

Add an "Import" feature that allows users to import assets from external data sources (plugins). Each plugin manages its own category and attributes, enabling users to quickly create assets with rich metadata fetched from external APIs.

---

## Motivation

Manually entering asset details is tedious and error-prone. For common asset types (books, movies, video games), rich metadata already exists in public APIs. This feature allows users to:

1. Search external sources by title, ISBN, or other identifiers
2. Import assets with pre-populated metadata and images
3. Maintain a link to the source for potential future refresh

---

## Design

### Core Concepts

#### Plugin

A plugin is a built-in module that:
- Manages a specific category (e.g., "Books")
- Defines namespaced attributes for that category (e.g., `books.isbn`, `books.author`)
- Provides search and fetch capabilities from an external API

#### Plugin-Owned vs User-Owned Attributes

| Type | Namespaced | Reusable | Editable by User | Example |
|------|------------|----------|------------------|---------|
| Plugin-owned | Yes (`books.isbn`) | No | No (values yes, definition no) | ISBN, Author, Publisher |
| User-owned | No | Yes | Yes | Purchase Price, Shelf Location |

Users can add their own attributes to plugin-managed categories. These user-added attributes follow the existing reusable attribute pattern.

#### Source Tracking

Assets imported via plugins track their origin:
- `import_plugin_id`: Which plugin imported it (e.g., `"google_books"`)
- `import_external_id`: External identifier for refresh capability

---

### Plugin Interface

```go
// ImportPlugin defines the interface for all import plugins
type ImportPlugin interface {
    // Metadata
    ID() string          // Unique identifier, e.g., "google_books"
    Name() string        // Display name, e.g., "Google Books"
    Description() string // Brief description of the plugin

    // Category management
    CategoryName() string              // Category this plugin manages
    CategoryDescription() string       // Description for the category
    Attributes() []PluginAttribute     // Attributes this plugin provides

    // Search capabilities
    SearchFields() []SearchField       // Available search fields
    Search(ctx context.Context, field, query string, limit int) ([]SearchResult, error)

    // Data fetching
    Fetch(ctx context.Context, externalID string) (*ImportData, error)
}

// PluginAttribute defines an attribute managed by a plugin
type PluginAttribute struct {
    Key      string             // Namespaced key, e.g., "books.isbn"
    Name     string             // Display name, e.g., "ISBN"
    DataType AttributeDataType  // string, number, boolean, date, text
    Required bool               // Is this attribute required?
}

// SearchField defines a searchable field
type SearchField struct {
    Key   string // Field identifier, e.g., "title", "isbn"
    Label string // Display label, e.g., "Title", "ISBN"
}

// SearchResult represents a search result from a plugin
type SearchResult struct {
    ExternalID  string  // External identifier for fetching full data
    Title       string  // Primary display title
    Subtitle    string  // Secondary info (e.g., author, year)
    ImageURL    *string // Thumbnail URL if available
}

// ImportData contains the full data for an import
type ImportData struct {
    Name        string            // Asset name
    Description *string           // Asset description
    ImageURL    *string           // Primary image URL
    Attributes  map[string]any    // Attribute values (keyed by attribute key)
    ExternalID  string            // External ID for source tracking
}
```

---

### Data Model Changes

#### Asset Table

Add columns for source tracking:

```sql
ALTER TABLE assets ADD COLUMN import_plugin_id VARCHAR(50);
ALTER TABLE assets ADD COLUMN import_external_id VARCHAR(255);

CREATE INDEX idx_assets_import ON assets(import_plugin_id, import_external_id)
    WHERE import_plugin_id IS NOT NULL;
```

#### Attributes Table

Add column for plugin ownership:

```sql
ALTER TABLE attributes ADD COLUMN plugin_id VARCHAR(50);

-- Plugin attributes are namespaced and not reusable
-- plugin_id = NULL means user-defined (reusable)
-- plugin_id = 'google_books' means owned by that plugin
```

#### Categories Table

Add column to track plugin-managed categories:

```sql
ALTER TABLE categories ADD COLUMN plugin_id VARCHAR(50);

-- plugin_id = NULL means user-created category
-- plugin_id = 'google_books' means managed by that plugin
```

---

### API Endpoints

#### Plugin Discovery

```
GET /api/plugins
```

Returns list of available plugins with their metadata:

```json
{
  "plugins": [
    {
      "id": "google_books",
      "name": "Google Books",
      "description": "Import books from Google Books API",
      "category_name": "Books",
      "search_fields": [
        {"key": "title", "label": "Title"},
        {"key": "isbn", "label": "ISBN"}
      ],
      "enabled": true,
      "category_id": "uuid-of-books-category"
    }
  ]
}
```

#### Plugin Search

```
GET /api/plugins/{plugin_id}/search?field={field}&q={query}&limit={limit}
```

Returns search results from the plugin:

```json
{
  "results": [
    {
      "external_id": "abc123",
      "title": "The Pragmatic Programmer",
      "subtitle": "David Thomas, Andrew Hunt (2019)",
      "image_url": "https://..."
    }
  ]
}
```

#### Plugin Import

```
POST /api/plugins/{plugin_id}/import
```

Request body:
```json
{
  "external_id": "abc123"
}
```

Response:
```json
{
  "asset": {
    "id": "uuid-of-created-asset",
    "name": "The Pragmatic Programmer",
    "category_id": "uuid-of-books-category",
    "attributes": {
      "books.isbn": "978-0135957059",
      "books.author": "David Thomas, Andrew Hunt",
      "books.publisher": "Addison-Wesley",
      "books.page_count": 352
    },
    "import_plugin_id": "google_books",
    "import_external_id": "abc123"
  }
}
```

#### Plugin Enable/Disable

```
POST /api/plugins/{plugin_id}/enable
POST /api/plugins/{plugin_id}/disable
```

- **Enable**: Creates the plugin's category and attributes if they don't exist
- **Disable**: Marks plugin as disabled; keeps category/attributes if assets exist

---

### Import Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  1. User clicks "Import" button                                 │
└─────────────────┬───────────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. GET /api/plugins → Display available plugins                │
│     User selects: [Google Books]                                │
└─────────────────┬───────────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. Display search form with plugin's search fields             │
│     Search by: [Title ▼]  Query: [pragmatic programmer]  🔍     │
└─────────────────┬───────────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. GET /api/plugins/google_books/search?field=title&q=...      │
│     Display results with [Import] buttons                       │
└─────────────────┬───────────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. User clicks [Import] on desired result                      │
│     POST /api/plugins/google_books/import {external_id: "..."}  │
└─────────────────┬───────────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. Asset created → Redirect to /assets/{id}/edit               │
│     User can review and modify imported data                    │
└─────────────────────────────────────────────────────────────────┘
```

---

### Plugin Lifecycle

#### Installation (Enable)

When a plugin is enabled:

1. Check if plugin's category exists (by `plugin_id`)
2. If not, create category with `plugin_id` set
3. Create/update plugin's namespaced attributes
4. Link attributes to category

#### Uninstallation (Disable)

When a plugin is disabled:

1. Check if any assets exist in the plugin's category
2. If assets exist: Keep category and attributes, just mark plugin as disabled
3. If no assets: Optionally delete category and attributes (or keep for re-enable)

---

### Initial Plugin: Google Books

#### Configuration

- **ID**: `google_books`
- **API**: Google Books API (free, no auth required for basic search)
- **Base URL**: `https://www.googleapis.com/books/v1/volumes`

#### Attributes

| Key | Name | Data Type |
|-----|------|-----------|
| `books.isbn` | ISBN | string |
| `books.author` | Author | string |
| `books.publisher` | Publisher | string |
| `books.published_date` | Published Date | string |
| `books.page_count` | Page Count | number |
| `books.language` | Language | string |

#### Search Fields

| Key | Label | API Mapping |
|-----|-------|-------------|
| `title` | Title | `intitle:` |
| `isbn` | ISBN | `isbn:` |
| `author` | Author | `inauthor:` |

---

### Future Plugins (Out of Scope)

Potential plugins for future implementation:

- **TMDB** (The Movie Database) - Movies & TV shows
- **IGDB** - Video games
- **Discogs** - Music/Vinyl records
- **OpenLibrary** - Alternative book source
- **MusicBrainz** - Music metadata

---

## Security Considerations

1. **API Keys**: Some plugins may require API keys. Store securely in environment variables.
2. **Rate Limiting**: Implement rate limiting on search endpoints to prevent abuse.
3. **Image Downloads**: When importing images, download and store in S3 rather than hotlinking.
4. **Input Validation**: Validate all external data before storing.

---

## Alternatives Considered

### 1. Plugin Attached to Existing Categories

**Rejected**: Adds complexity in mapping plugin attributes to user categories. Having plugins own their categories is cleaner.

### 2. Automatic Import on Asset Creation

**Rejected**: Less control for users. Explicit import flow is more predictable and avoids unwanted data population.

### 3. Multiple Plugins per Asset

**Rejected**: Adds complexity in handling conflicts between data sources. Users wanting merged data can use a dedicated merge plugin in the future.

### 4. Preview Before Import

**Rejected**: Adds UI complexity. Direct import with edit-after is simpler and achieves the same goal.
