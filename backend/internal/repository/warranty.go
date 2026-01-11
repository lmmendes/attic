package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mendelui/attic/internal/domain"
)

type WarrantyRepository struct {
	pool *pgxpool.Pool
}

func NewWarrantyRepository(pool *pgxpool.Pool) *WarrantyRepository {
	return &WarrantyRepository{pool: pool}
}

func (r *WarrantyRepository) GetByAssetID(ctx context.Context, assetID uuid.UUID) (*domain.Warranty, error) {
	query := `
		SELECT id, asset_id, provider, start_date, end_date, notes, created_at, updated_at
		FROM warranties
		WHERE asset_id = $1
	`
	var w domain.Warranty
	err := r.pool.QueryRow(ctx, query, assetID).Scan(
		&w.ID, &w.AssetID, &w.Provider, &w.StartDate, &w.EndDate,
		&w.Notes, &w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WarrantyRepository) ListExpiring(ctx context.Context, orgID uuid.UUID, days int) ([]domain.Warranty, error) {
	query := `
		SELECT w.id, w.asset_id, w.provider, w.start_date, w.end_date, w.notes, w.created_at, w.updated_at
		FROM warranties w
		JOIN assets a ON a.id = w.asset_id
		WHERE a.organization_id = $1
		  AND a.deleted_at IS NULL
		  AND w.end_date IS NOT NULL
		  AND w.end_date <= $2
		ORDER BY w.end_date ASC
	`
	expiryDate := time.Now().AddDate(0, 0, days)
	rows, err := r.pool.Query(ctx, query, orgID, expiryDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warranties []domain.Warranty
	for rows.Next() {
		var w domain.Warranty
		if err := rows.Scan(
			&w.ID, &w.AssetID, &w.Provider, &w.StartDate, &w.EndDate,
			&w.Notes, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		warranties = append(warranties, w)
	}
	return warranties, rows.Err()
}

func (r *WarrantyRepository) Create(ctx context.Context, w *domain.Warranty) error {
	query := `
		INSERT INTO warranties (id, asset_id, provider, start_date, end_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		w.ID, w.AssetID, w.Provider, w.StartDate, w.EndDate, w.Notes,
	).Scan(&w.CreatedAt, &w.UpdatedAt)
}

func (r *WarrantyRepository) Update(ctx context.Context, w *domain.Warranty) error {
	query := `
		UPDATE warranties
		SET provider = $2, start_date = $3, end_date = $4, notes = $5
		WHERE id = $1
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		w.ID, w.Provider, w.StartDate, w.EndDate, w.Notes,
	).Scan(&w.UpdatedAt)
}

func (r *WarrantyRepository) Delete(ctx context.Context, assetID uuid.UUID) error {
	query := `DELETE FROM warranties WHERE asset_id = $1`
	_, err := r.pool.Exec(ctx, query, assetID)
	return err
}
