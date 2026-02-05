package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
)

// Migrate applies all pending migrations to the database using golang-migrate
func (db *DB) Migrate(ctx context.Context, migrationsFS fs.FS) error {
	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	// Get a *sql.DB from the pgx pool for golang-migrate
	sqlDB := stdlib.OpenDBFromPool(db.Pool)

	// Create postgres driver
	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("creating database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("migrations up to date")
			return nil
		}
		return fmt.Errorf("running migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("getting migration version: %w", err)
	}

	slog.Info("migrations applied", "version", version, "dirty", dirty)
	return nil
}
