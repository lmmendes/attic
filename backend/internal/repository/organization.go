package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mendelui/attic/internal/domain"
)

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`
	var o domain.Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.Name, &o.Description, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizationRepository) GetDefault(ctx context.Context) (*domain.Organization, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations
		WHERE deleted_at IS NULL
		ORDER BY created_at
		LIMIT 1
	`
	var o domain.Organization
	err := r.pool.QueryRow(ctx, query).Scan(
		&o.ID, &o.Name, &o.Description, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizationRepository) Create(ctx context.Context, o *domain.Organization) error {
	query := `
		INSERT INTO organizations (id, name, description)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query, o.ID, o.Name, o.Description).Scan(&o.CreatedAt, &o.UpdatedAt)
}

func (r *OrganizationRepository) Update(ctx context.Context, o *domain.Organization) error {
	query := `
		UPDATE organizations
		SET name = $2, description = $3
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query, o.ID, o.Name, o.Description).Scan(&o.UpdatedAt)
}
