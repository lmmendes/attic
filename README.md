# Attic

> Self-hosted home inventory for everything you own.

Attic is an open-source home inventory application for individuals and households. Catalog everything you own—from appliances and tools to electronics, books, games, and furniture—and keep its location, condition, purchase information, warranty, photos, receipts, and manuals together.

Attic mirrors the way a home is organized with nested locations such as rooms, shelves, cupboards, and boxes. Categories and custom fields capture the right details for each kind of belonging, while collections group related items without changing where they are stored. It works for a focused collection or an entire household inventory.

## Key Features

**Home Inventory**
- Full CRUD with custom attributes per category (strings, numbers, booleans, dates, dropdowns)
- Hierarchical categories and locations; child categories inherit their ancestors' custom fields
- Condition tracking (new, used, damaged, or custom states)
- Warranty expiration monitoring on the dashboard
- File attachments for invoices, manuals, and photos
- Shared household collections with names, descriptions, and icons (e.g. PS5 games, books, furniture). Assign assets to any number of collections from the asset form and filter inventory by collection. Deleting a collection preserves its assets.
- Purchase dates, prices, and notes with dashboard value summaries

**Search & Discovery**
- Full-text search across asset names and descriptions
- Filter by category, location, and condition
- Category filters include assets assigned to descendant categories
- Search attribute values with case-insensitive substrings, including dropdown option labels
- Build nested Match all (AND) / Match any (OR) filters with typed attribute comparisons and collection membership
- Save named filters privately for reuse, and pin up to five as one-click desktop sidebar shortcuts; even workspace administrators cannot access another user's saved filters

**Smart Integrations**
- Automated imports from Google Books, TMDB (movies and TV), BoardGameGeek, and IGDB (video games)
- Metadata and cover images populated automatically
- Plugin system for adding new import sources

Google Books works without credentials, but Google may apply a low shared quota to unauthenticated
requests. Set `ATTIC_GOOGLE_BOOKS_API_KEY` to a Google Books API key for reliable imports in a
self-hosted or shared deployment. The key is sent only to Google Books API requests.

IGDB imports require `ATTIC_IGDB_CLIENT_ID` and `ATTIC_IGDB_CLIENT_SECRET` from a
[Twitch application](https://dev.twitch.tv/console/apps). Attic exchanges these credentials for a
short-lived access token and keeps the token in memory only.

**Self-Hosted & Secure**
- Docker-based deployment with complete data ownership
- Local password authentication and OIDC/SSO (Keycloak compatible)
- REST API with Swagger documentation
- Local or S3-compatible storage for attachments
- Dark mode with mobile-responsive UI

## Asset filters

Use **Search attribute values** on Assets for one search term, or open **Advanced filter**
to combine rules and nested groups. Text supports equals/contains; numbers and dates
support comparisons and inclusive ranges; booleans support true/false; dropdowns
support any selected option, and multiple selections and collections also support all.
Every attribute supports empty/not-empty checks. Missing values, null, empty strings,
and empty arrays count as empty; zero and false do not.

Quick search matches values and dropdown labels, not attribute names or internal option
identifiers. `%`, `_`, and backslashes are literal characters. Only active, visible
attributes participate. Quick search, existing page filters, and advanced rules combine
with AND. Apply a draft immediately, save it as a new named filter, or explicitly update
an existing saved filter. Saved criteria exclude pagination. Deleted references or
incompatible attribute changes require repairing the affected rules before applying;
renames preserve references through stable IDs.

The API exposes `GET /api/assets?attribute_q=Commodore`, structured searches through
`POST /api/assets/search`, and personal CRUD under `/api/saved-filters`. A structured
search body is:

```json
{
  "criteria": {
    "version": 1,
    "expression": {
      "kind": "group",
      "match": "all",
      "children": [
        { "kind": "rule", "field": "attribute_q", "operator": "contains", "value": "Commodore" },
        { "kind": "rule", "field": "q", "operator": "search", "value": "computer" }
      ]
    }
  },
  "limit": 20,
  "offset": 0
}
```

Create a saved filter with `{"name":"Retro computers","criteria":{...}}`; update with
`PUT /api/saved-filters/{id}` using the same shape. Omitting `criteria` on update only
renames it, including when its rules need repair. GET/list responses include `issues`
with rule paths for invalid saved definitions. Expressions allow five group levels,
50 total rules, and 100 values per membership rule; JSON request bodies are limited to
64 KiB. See the bundled OpenAPI document at `/api/docs` for the complete wire format.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.27+
- Bun 1.1+
- Make

### Development Setup

1. **Clone the repository and start infrastructure:**
   ```bash
   git clone git@github.com:lmmendes/attic.git
   cd attic
   docker compose up -d
   ```

2. **Run database migrations:**
   ```bash
   make migrate-up
   ```

3. **Start the backend:**
   ```bash
   cd backend
   go run ./cmd/server
   ```

4. **Start the frontend (new terminal):**
   ```bash
   cd frontend
   bun install
   bun run dev
   ```

5. **Open the app:**

   | Service     | URL                            |
   |-------------|--------------------------------|
   | Frontend    | http://localhost:3000          |
   | Backend API | http://localhost:8080          |
   | API Docs    | http://localhost:8080/api/docs |
   | Keycloak    | http://localhost:8180          |

   Default test credentials: `testuser` / `testpassword`

### Browser integration tests

The Playwright suite exercises real browser workflows against a running Attic
instance, including login, category/attribute handoffs, and feature settings.

```bash
cd frontend
bunx playwright install chromium
bun run test:e2e
```

The local suite targets the Nuxt development server at `http://127.0.0.1:3000`.
Set `E2E_BASE_URL` when testing another running instance.

GitHub Actions runs the same suite against the compiled application and a
disposable PostgreSQL database on every pull request.

### Production Deployment

```bash
cp .env.example .env
# Edit .env with your production values

docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml exec backend \
  /app/migrate -path /migrations -database "$DATABASE_URL" up
```

For more details, visit [getattic.dev](https://getattic.dev).

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend   | Go 1.27, Chi router, PostgreSQL |
| Frontend  | Nuxt 4, Nuxt UI 4, Tailwind CSS |
| Auth      | Local passwords or OIDC/SSO |
| Storage   | S3-compatible (AWS S3, MinIO, LocalStack) |


## License

[MIT](LICENSE.md)
