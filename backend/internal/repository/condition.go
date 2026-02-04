package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type ConditionRepository struct {
	pool *pgxpool.Pool
}

func NewConditionRepository(pool *pgxpool.Pool) *ConditionRepository {
	return &ConditionRepository{pool: pool}
}

func (r *ConditionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Condition, error) {
	query := `
		SELECT id, organization_id, code, label, description, sort_order, created_at, updated_at
		FROM conditions
		WHERE id = $1 AND deleted_at IS NULL
	`
	var c domain.Condition
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.OrganizationID, &c.Code, &c.Label, &c.Description,
		&c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConditionRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Condition, error) {
	query := `
		SELECT id, organization_id, code, label, description, sort_order, created_at, updated_at
		FROM conditions
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order, label
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conditions []domain.Condition
	for rows.Next() {
		var c domain.Condition
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.Code, &c.Label, &c.Description,
			&c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		conditions = append(conditions, c)
	}
	return conditions, rows.Err()
}

func (r *ConditionRepository) Create(ctx context.Context, c *domain.Condition) error {
	query := `
		INSERT INTO conditions (id, organization_id, code, label, description, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		c.ID, c.OrganizationID, c.Code, c.Label, c.Description, c.SortOrder,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ConditionRepository) Update(ctx context.Context, c *domain.Condition) error {
	query := `
		UPDATE conditions
		SET code = $2, label = $3, description = $4, sort_order = $5
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		c.ID, c.Code, c.Label, c.Description, c.SortOrder,
	).Scan(&c.UpdatedAt)
}

func (r *ConditionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE conditions SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
