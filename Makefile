.PHONY: help dev dev-up dev-down backend-run backend-build backend-test migrate-up migrate-down migrate-create frontend-dev frontend-build frontend-test build clean

help:
	@echo "Available commands:"
	@echo "  build         - Build complete application (frontend + backend)"
	@echo "  clean         - Remove build artifacts"
	@echo "  dev           - Start all development services"
	@echo "  dev-up        - Start Docker Compose services"
	@echo "  dev-down      - Stop Docker Compose services"
	@echo "  backend-run   - Run backend server"
	@echo "  backend-build - Build backend binary"
	@echo "  backend-test  - Run backend tests"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback last migration"
	@echo "  migrate-create - Create new migration (NAME=xxx)"
	@echo "  frontend-dev  - Run frontend dev server"
	@echo "  frontend-build - Build frontend for production"
	@echo "  frontend-test - Run frontend tests"

# Combined build (frontend embedded in backend)
build: frontend-build backend-build
	@echo "Build complete: backend/bin/attic"

clean:
	rm -rf backend/bin
	rm -rf backend/cmd/server/dist/*
	rm -rf backend/cmd/server/.output
	touch backend/cmd/server/dist/.gitkeep
	rm -rf frontend/.nuxt frontend/.output

# Development
dev: dev-up backend-run

dev-up:
	docker compose up -d

dev-down:
	docker compose down

# Backend
backend-run:
	cd backend && go run ./cmd/server

LDFLAGS := -w -s
ifdef ATTIC_TMDB_API_KEY
	LDFLAGS += -X github.com/lmmendes/attic/internal/plugin/tmdb.APIKey=$(ATTIC_TMDB_API_KEY)
endif

backend-build:
	cd backend && go build -ldflags="$(LDFLAGS)" -o bin/attic ./cmd/server

backend-test:
	cd backend && go test -v ./...

# Migrations
DATABASE_URL ?= postgres://attic:attic@localhost:5432/attic?sslmode=disable

migrate-up:
	migrate -path backend/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path backend/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir backend/migrations -seq $(NAME)

# Frontend
frontend-dev:
	cd frontend && bun run dev

frontend-build:
	cd frontend && bun run build

frontend-test:
	cd frontend && bun run test
