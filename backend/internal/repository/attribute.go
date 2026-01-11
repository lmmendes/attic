package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mendelui/attic/internal/domain"
)

type AttributeRepository struct {
	pool *pgxpool.Pool
}

func NewAttributeRepository(pool *pgxpool.Pool) *AttributeRepository {
	return &AttributeRepository{pool: pool}
}

func (r *AttributeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attribute, error) {
	query := `
		SELECT id, organization_id, name, key, data_type, created_at, updated_at
		FROM attributes
		WHERE id = $1 AND deleted_at IS NULL
	`
	var a domain.Attribute
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.OrganizationID, &a.Name, &a.Key, &a.DataType,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttributeRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Attribute, error) {
	query := `
		SELECT id, organization_id, name, key, data_type, created_at, updated_at
		FROM attributes
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attributes []domain.Attribute
	for rows.Next() {
		var a domain.Attribute
		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.Name, &a.Key, &a.DataType,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		attributes = append(attributes, a)
	}
	return attributes, rows.Err()
}

func (r *AttributeRepository) Create(ctx context.Context, a *domain.Attribute) error {
	query := `
		INSERT INTO attributes (id, organization_id, name, key, data_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		a.ID, a.OrganizationID, a.Name, a.Key, a.DataType,
	).Scan(&a.CreatedAt, &a.UpdatedAt)
}

func (r *AttributeRepository) Update(ctx context.Context, a *domain.Attribute) error {
	query := `
		UPDATE attributes
		SET name = $2, data_type = $3
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		a.ID, a.Name, a.DataType,
	).Scan(&a.UpdatedAt)
}

func (r *AttributeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE attributes SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
