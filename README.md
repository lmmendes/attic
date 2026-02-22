# Attic

> Track everything. Lose nothing.

Attic is an open-source asset management system that lets you track, organize, and manage everything you own — from electronics and books to board games and digital assets.

It mirrors the way your real-world spaces are organized with hierarchical locations and categories, supports rich metadata through custom attributes, and integrates with external sources to auto-fill asset details. Whether you're managing a home library or an office inventory.

---

## Key Features

**Asset Management**
- Full CRUD with custom attributes per category (strings, numbers, booleans, dates, dropdowns)
- Hierarchical categories and locations that mirror real-world spaces
- Condition tracking (new, used, damaged, or custom states)
- Warranty expiration monitoring with alerts
- File attachments for invoices, manuals, and photos
- Collections for grouping related assets (e.g. board game + expansions)

**Search & Discovery**
- Full-text search across names, descriptions, tags, and custom fields
- Filter by category, location, condition, tags, and typed attribute values

**Smart Integrations**
- Automated imports from Google Books, TMDB (movies), and BoardGameGeek
- Metadata and cover images populated automatically
- Plugin system for adding new import sources

**Self-Hosted & Secure**
- Docker-based deployment with complete data ownership
- OIDC/SSO authentication (Keycloak compatible)
- REST API with Swagger documentation
- S3-compatible storage for attachments
- Dark mode with mobile-responsive UI

---

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.24+
- Bun 1.1+
- Make

### Development Setup

1. **Clone the repository and start infrastructure:**
   ```bash
   git clone <repo-url>
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

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend   | Go 1.24, Chi router, PostgreSQL |
| Frontend  | Nuxt 3, NuxtUI 4, TailwindCSS |
| Auth      | Keycloak (OIDC), JWT |
| Storage   | S3-compatible (AWS S3, MinIO, LocalStack) |

---

## License

MIT
