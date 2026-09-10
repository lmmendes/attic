package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

// AssetEventRepository persists custom asset history events.
type AssetEventRepository struct{ pool *pgxpool.Pool }

// NewAssetEventRepository creates an asset event repository backed by PostgreSQL.
func NewAssetEventRepository(pool *pgxpool.Pool) *AssetEventRepository {
	return &AssetEventRepository{pool: pool}
}

// GetByID returns an event scoped to its asset and organization.
func (r *AssetEventRepository) GetByID(ctx context.Context, orgID, assetID, eventID uuid.UUID) (*domain.AssetEvent, error) {
	var event domain.AssetEvent
	err := scanAssetEvent(r.pool.QueryRow(ctx, `
		SELECT e.id, e.asset_id, e.title, e.category, e.description, e.icon, e.occurred_at, e.created_at, e.updated_at
		FROM asset_events e
		JOIN assets a ON a.id = e.asset_id
		WHERE e.id = $1 AND e.asset_id = $2 AND a.organization_id = $3 AND a.deleted_at IS NULL
	`, eventID, assetID, orgID), &event)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// ListByAsset returns an asset's events in reverse occurrence order.
func (r *AssetEventRepository) ListByAsset(ctx context.Context, orgID, assetID uuid.UUID) ([]domain.AssetEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.asset_id, e.title, e.category, e.description, e.icon, e.occurred_at, e.created_at, e.updated_at
		FROM asset_events e
		JOIN assets a ON a.id = e.asset_id
		WHERE e.asset_id = $1 AND a.organization_id = $2 AND a.deleted_at IS NULL
		ORDER BY e.occurred_at DESC, e.created_at DESC, e.id DESC
	`, assetID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []domain.AssetEvent{}
	for rows.Next() {
		var event domain.AssetEvent
		if err := scanAssetEvent(rows, &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// Create persists an event when its parent asset belongs to the organization.
func (r *AssetEventRepository) Create(ctx context.Context, orgID uuid.UUID, event *domain.AssetEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO asset_events (id, asset_id, title, category, description, icon, occurred_at)
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE EXISTS (
			SELECT 1 FROM assets WHERE id = $2 AND organization_id = $8 AND deleted_at IS NULL
		)
		RETURNING created_at, updated_at
	`, event.ID, event.AssetID, event.Title, event.Category, event.Description, event.Icon, event.OccurredAt, orgID).
		Scan(&event.CreatedAt, &event.UpdatedAt)
}

// Update replaces editable event fields within the organization scope.
func (r *AssetEventRepository) Update(ctx context.Context, orgID uuid.UUID, event *domain.AssetEvent) error {
	return r.pool.QueryRow(ctx, `
		UPDATE asset_events e
		SET title = $4, category = $5, description = $6, icon = $7, occurred_at = $8
		FROM assets a
		WHERE e.id = $1 AND e.asset_id = $2 AND a.id = e.asset_id
			AND a.organization_id = $3 AND a.deleted_at IS NULL
		RETURNING e.updated_at
	`, event.ID, event.AssetID, orgID, event.Title, event.Category, event.Description, event.Icon, event.OccurredAt).
		Scan(&event.UpdatedAt)
}

// Delete removes an event scoped to its asset and organization.
func (r *AssetEventRepository) Delete(ctx context.Context, orgID, assetID, eventID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM asset_events e
		USING assets a
		WHERE e.id = $1 AND e.asset_id = $2 AND a.id = e.asset_id
			AND a.organization_id = $3 AND a.deleted_at IS NULL
	`, eventID, assetID, orgID)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

// scanAssetEvent reads an event from a PostgreSQL row.
func scanAssetEvent(row pgx.Row, event *domain.AssetEvent) error {
	return row.Scan(
		&event.ID, &event.AssetID, &event.Title, &event.Category, &event.Description,
		&event.Icon, &event.OccurredAt, &event.CreatedAt, &event.UpdatedAt,
	)
}
