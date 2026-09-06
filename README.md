# Attic

> Self-hosted home inventory for everything you own.

Attic is an open-source home inventory application for individuals and households. Catalog everything you own—from appliances and tools to electronics, books, games, and furniture—and keep its location, condition, purchase information, warranty, photos, receipts, and manuals together.

Attic mirrors the way a home is organized with nested locations such as rooms, shelves, cupboards, and boxes. Categories and custom fields capture the right details for each kind of belonging, while collections group related items without changing where they are stored. It works for a focused collection or an entire household inventory.

## Key Features

**Home Inventory**
- Full CRUD with custom attributes per category (strings, numbers, booleans, dates, dropdowns)
- Hierarchical categories and locations that mirror real-world spaces
- Condition tracking (new, used, damaged, or custom states)
- Warranty expiration monitoring on the dashboard
- File attachments for invoices, manuals, and photos
- Shared household collections with names, descriptions, and icons (e.g. PS5 games, books, furniture). Assign assets to any number of collections from the asset form and filter inventory by collection. Deleting a collection preserves its assets.
- Purchase dates, prices, and notes with dashboard value summaries

**Search & Discovery**
- Full-text search across asset names and descriptions
- Filter by category, location, and condition

**Smart Integrations**
- Automated imports from Google Books, TMDB (movies), and BoardGameGeek
- Metadata and cover images populated automatically
- Plugin system for adding new import sources

**Self-Hosted & Secure**
- Docker-based deployment with complete data ownership
- Local password authentication and OIDC/SSO (Keycloak compatible)
- REST API with Swagger documentation
- Local or S3-compatible storage for attachments
- Dark mode with mobile-responsive UI

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24+
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
| Backend   | Go 1.24, Chi router, PostgreSQL |
| Frontend  | Nuxt 4, Nuxt UI 4, Tailwind CSS |
| Auth      | Local passwords or OIDC/SSO |
| Storage   | S3-compatible (AWS S3, MinIO, LocalStack) |


## License

[MIT](LICENSE.md)
