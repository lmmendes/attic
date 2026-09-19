package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type SavedFilterRepository struct{ pool *pgxpool.Pool }

func NewSavedFilterRepository(pool *pgxpool.Pool) *SavedFilterRepository {
	return &SavedFilterRepository{pool: pool}
}

const savedFilterSelect = `SELECT id, organization_id, user_id, name, pinned, criteria, created_at, updated_at FROM saved_filters `

func (r *SavedFilterRepository) List(ctx context.Context, org, user uuid.UUID) ([]domain.SavedFilter, error) {
	rows, err := r.pool.Query(ctx, savedFilterSelect+`WHERE organization_id = $1 AND user_id = $2 ORDER BY lower(name), id`, org, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	filters := []domain.SavedFilter{}
	for rows.Next() {
		var filter domain.SavedFilter
		if err := scanSavedFilter(rows, &filter); err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Err()
}

func (r *SavedFilterRepository) GetByID(ctx context.Context, org, user, id uuid.UUID) (*domain.SavedFilter, error) {
	var filter domain.SavedFilter
	err := scanSavedFilter(r.pool.QueryRow(ctx, savedFilterSelect+`WHERE organization_id = $1 AND user_id = $2 AND id = $3`, org, user, id), &filter)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &filter, nil
}

func (r *SavedFilterRepository) Create(ctx context.Context, filter *domain.SavedFilter) error {
	criteria, err := json.Marshal(filter.Criteria)
	if err != nil {
		return err
	}
	if filter.ID == uuid.Nil {
		filter.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, `INSERT INTO saved_filters (id, organization_id, user_id, name, pinned, criteria)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at, updated_at`,
		filter.ID, filter.OrganizationID, filter.UserID, filter.Name, filter.Pinned, criteria).Scan(&filter.CreatedAt, &filter.UpdatedAt)
}

func (r *SavedFilterRepository) Update(ctx context.Context, filter *domain.SavedFilter) error {
	criteria, err := json.Marshal(filter.Criteria)
	if err != nil {
		return err
	}
	return r.pool.QueryRow(ctx, `UPDATE saved_filters SET name = $4, pinned = $5, criteria = $6
		WHERE organization_id = $1 AND user_id = $2 AND id = $3 RETURNING updated_at`,
		filter.OrganizationID, filter.UserID, filter.ID, filter.Name, filter.Pinned, criteria).Scan(&filter.UpdatedAt)
}

func (r *SavedFilterRepository) CountPinned(ctx context.Context, org, user uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM saved_filters WHERE organization_id = $1 AND user_id = $2 AND pinned`, org, user).Scan(&count)
	return count, err
}

func (r *SavedFilterRepository) Delete(ctx context.Context, org, user, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM saved_filters WHERE organization_id = $1 AND user_id = $2 AND id = $3`, org, user, id)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func scanSavedFilter(row pgx.Row, filter *domain.SavedFilter) error {
	var criteria []byte
	if err := row.Scan(&filter.ID, &filter.OrganizationID, &filter.UserID, &filter.Name, &filter.Pinned, &criteria, &filter.CreatedAt, &filter.UpdatedAt); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(criteria))
	decoder.UseNumber()
	return decoder.Decode(&filter.Criteria)
}
