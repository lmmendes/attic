package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type AssetEventRepository struct{ pool *pgxpool.Pool }

func NewAssetEventRepository(pool *pgxpool.Pool) *AssetEventRepository {
	return &AssetEventRepository{pool: pool}
}

func (r *AssetEventRepository) GetByID(ctx context.Context, orgID, assetID, eventID uuid.UUID) (*domain.AssetEvent, error) {
	var event domain.AssetEvent
	err := scanAssetEvent(r.pool.QueryRow(ctx, `
		SELECT e.id, e.asset_id, e.title, e.description, e.icon, e.event_date, e.created_at, e.updated_at
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

func (r *AssetEventRepository) ListByAsset(ctx context.Context, orgID, assetID uuid.UUID) ([]domain.AssetEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.asset_id, e.title, e.description, e.icon, e.event_date, e.created_at, e.updated_at
		FROM asset_events e
		JOIN assets a ON a.id = e.asset_id
		WHERE e.asset_id = $1 AND a.organization_id = $2 AND a.deleted_at IS NULL
		ORDER BY e.event_date DESC, e.created_at DESC, e.id DESC
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

func (r *AssetEventRepository) Create(ctx context.Context, orgID uuid.UUID, event *domain.AssetEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO asset_events (id, asset_id, title, description, icon, event_date)
		SELECT $1, $2, $3, $4, $5, $6
		WHERE EXISTS (
			SELECT 1 FROM assets WHERE id = $2 AND organization_id = $7 AND deleted_at IS NULL
		)
		RETURNING created_at, updated_at
	`, event.ID, event.AssetID, event.Title, event.Description, event.Icon, event.EventDate, orgID).
		Scan(&event.CreatedAt, &event.UpdatedAt)
}

func (r *AssetEventRepository) Update(ctx context.Context, orgID uuid.UUID, event *domain.AssetEvent) error {
	return r.pool.QueryRow(ctx, `
		UPDATE asset_events e
		SET title = $4, description = $5, icon = $6, event_date = $7
		FROM assets a
		WHERE e.id = $1 AND e.asset_id = $2 AND a.id = e.asset_id
			AND a.organization_id = $3 AND a.deleted_at IS NULL
		RETURNING e.updated_at
	`, event.ID, event.AssetID, orgID, event.Title, event.Description, event.Icon, event.EventDate).
		Scan(&event.UpdatedAt)
}

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

func scanAssetEvent(row pgx.Row, event *domain.AssetEvent) error {
	return row.Scan(
		&event.ID, &event.AssetID, &event.Title, &event.Description,
		&event.Icon, &event.EventDate, &event.CreatedAt, &event.UpdatedAt,
	)
}
