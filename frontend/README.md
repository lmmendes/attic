# Attic frontend

The Nuxt frontend for Attic, an open-source, self-hosted home inventory application. It helps individuals and households catalog their belongings, record where items are stored, and keep details such as condition, purchase information, warranties, receipts, manuals, and photos together.

## Requirements

- Bun 1.1+
- The Attic backend, available at `http://localhost:8080` by default

## Development

Install dependencies and start the frontend:

```bash
bun install --frozen-lockfile
bun run dev
```

The frontend is available at `http://localhost:3000`. When the backend runs separately, create `frontend/.env` with:

```env
NUXT_PUBLIC_API_BASE=http://localhost:8080
```

The backend must allow the frontend origin, typically with `ATTIC_CORS_ORIGINS=http://localhost:3000`.

## Validation

```bash
bun run test:run
bun run lint
bun run typecheck
```

## Production build

```bash
bun run build
```

Nuxt writes the generated static frontend to `backend/cmd/server/dist` so the Go server can embed and serve it.
