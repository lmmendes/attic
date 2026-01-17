package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB wraps a PostgreSQL testcontainer with a connection pool
type TestDB struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
}

// NewTestDB creates a new PostgreSQL testcontainer and applies all migrations
func NewTestDB(ctx context.Context) (*TestDB, error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("attic_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	testDB := &TestDB{
		Container: container,
		Pool:      pool,
	}

	if err := testDB.applyMigrations(ctx); err != nil {
		testDB.Close(ctx)
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return testDB, nil
}

// Close terminates the container and closes the pool
func (t *TestDB) Close(ctx context.Context) {
	if t.Pool != nil {
		t.Pool.Close()
	}
	if t.Container != nil {
		t.Container.Terminate(ctx)
	}
}

// TruncateAll truncates all tables to reset state between tests
func (t *TestDB) TruncateAll(ctx context.Context) error {
	tables := []string{
		"attachments",
		"warranties",
		"asset_tags",
		"assets",
		"category_attributes",
		"attributes",
		"tags",
		"locations",
		"categories",
		"conditions",
		"users",
		"organizations",
	}

	for _, table := range tables {
		_, err := t.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}
	return nil
}

func (t *TestDB) applyMigrations(ctx context.Context) error {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("could not find migrations directory")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var upMigrations []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			upMigrations = append(upMigrations, entry.Name())
		}
	}
	sort.Strings(upMigrations)

	for _, migration := range upMigrations {
		path := filepath.Join(migrationsDir, migration)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		_, err = t.Pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration, err)
		}
	}

	return nil
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"../migrations",
		"../../migrations",
		"../../../migrations",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return ""
}
