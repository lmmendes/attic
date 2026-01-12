package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mendelui/attic/internal/domain"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, created_at, updated_at
		FROM categories
		WHERE id = $1 AND deleted_at IS NULL
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) GetByIDWithAttributes(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	cat, err := r.GetByID(ctx, id)
	if err != nil || cat == nil {
		return cat, err
	}

	attrQuery := `
		SELECT ca.id, ca.category_id, ca.attribute_id, ca.required, ca.sort_order, ca.created_at,
		       a.id, a.organization_id, a.name, a.key, a.data_type, a.created_at, a.updated_at
		FROM category_attributes ca
		JOIN attributes a ON a.id = ca.attribute_id AND a.deleted_at IS NULL
		WHERE ca.category_id = $1
		ORDER BY ca.sort_order, a.name
	`
	rows, err := r.pool.Query(ctx, attrQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ca domain.CategoryAttribute
		var attr domain.Attribute
		if err := rows.Scan(
			&ca.ID, &ca.CategoryID, &ca.AttributeID, &ca.Required, &ca.SortOrder, &ca.CreatedAt,
			&attr.ID, &attr.OrganizationID, &attr.Name, &attr.Key, &attr.DataType, &attr.CreatedAt, &attr.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ca.Attribute = &attr
		cat.Attributes = append(cat.Attributes, ca)
	}
	return cat, rows.Err()
}

func (r *CategoryRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, created_at, updated_at
		FROM categories
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY name
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *CategoryRepository) ListTree(ctx context.Context, orgID uuid.UUID) ([]domain.Category, error) {
	categories, err := r.List(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return buildCategoryTree(categories), nil
}

func buildCategoryTree(categories []domain.Category) []domain.Category {
	byID := make(map[uuid.UUID]*domain.Category)
	for i := range categories {
		byID[categories[i].ID] = &categories[i]
	}

	var roots []domain.Category
	for i := range categories {
		cat := &categories[i]
		if cat.ParentID == nil {
			roots = append(roots, *cat)
		} else if parent, ok := byID[*cat.ParentID]; ok {
			parent.Children = append(parent.Children, *cat)
		}
	}
	return roots
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO categories (id, organization_id, parent_id, plugin_id, name, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		c.ID, c.OrganizationID, c.ParentID, c.PluginID, c.Name, c.Description,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CategoryRepository) GetByPluginID(ctx context.Context, orgID uuid.UUID, pluginID string) (*domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, created_at, updated_at
		FROM categories
		WHERE organization_id = $1 AND plugin_id = $2 AND deleted_at IS NULL
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, orgID, pluginID).Scan(
		&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	query := `
		UPDATE categories
		SET parent_id = $2, name = $3, description = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		c.ID, c.ParentID, c.Name, c.Description,
	).Scan(&c.UpdatedAt)
}

func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE categories SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *CategoryRepository) SetAttributes(ctx context.Context, categoryID uuid.UUID, assignments []domain.CategoryAttributeAssignment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete existing assignments
	_, err = tx.Exec(ctx, `DELETE FROM category_attributes WHERE category_id = $1`, categoryID)
	if err != nil {
		return err
	}

	// Insert new assignments
	if len(assignments) > 0 {
		for _, a := range assignments {
			_, err = tx.Exec(ctx, `
				INSERT INTO category_attributes (category_id, attribute_id, required, sort_order)
				VALUES ($1, $2, $3, $4)
			`, categoryID, a.AttributeID, a.Required, a.SortOrder)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
