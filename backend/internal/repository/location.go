package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type LocationRepository struct {
	pool *pgxpool.Pool
}

func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{pool: pool}
}

func (r *LocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Location, error) {
	query := `
		SELECT id, organization_id, parent_id, name, description, created_at, updated_at
		FROM locations
		WHERE id = $1 AND deleted_at IS NULL
	`
	var l domain.Location
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.OrganizationID, &l.ParentID, &l.Name, &l.Description,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LocationRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Location, error) {
	query := `
		SELECT id, organization_id, parent_id, name, description, created_at, updated_at
		FROM locations
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []domain.Location
	for rows.Next() {
		var l domain.Location
		if err := rows.Scan(
			&l.ID, &l.OrganizationID, &l.ParentID, &l.Name, &l.Description,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		locations = append(locations, l)
	}
	return locations, rows.Err()
}

func (r *LocationRepository) ListTree(ctx context.Context, orgID uuid.UUID) ([]domain.Location, error) {
	locations, err := r.List(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return buildLocationTree(locations), nil
}

func buildLocationTree(locations []domain.Location) []domain.Location {
	byID := make(map[uuid.UUID]*domain.Location)
	for i := range locations {
		byID[locations[i].ID] = &locations[i]
	}

	var roots []domain.Location
	for i := range locations {
		loc := &locations[i]
		if loc.ParentID == nil {
			roots = append(roots, *loc)
		} else if parent, ok := byID[*loc.ParentID]; ok {
			parent.Children = append(parent.Children, *loc)
		}
	}
	return roots
}

func (r *LocationRepository) Create(ctx context.Context, l *domain.Location) error {
	query := `
		INSERT INTO locations (id, organization_id, parent_id, name, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		l.ID, l.OrganizationID, l.ParentID, l.Name, l.Description,
	).Scan(&l.CreatedAt, &l.UpdatedAt)
}

func (r *LocationRepository) Update(ctx context.Context, l *domain.Location) error {
	query := `
		UPDATE locations
		SET parent_id = $2, name = $3, description = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		l.ID, l.ParentID, l.Name, l.Description,
	).Scan(&l.UpdatedAt)
}

func (r *LocationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE locations SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
