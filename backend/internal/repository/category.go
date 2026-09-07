package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, icon, created_at, updated_at
		FROM categories
		WHERE id = $1 AND deleted_at IS NULL
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description, &c.Icon,
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

	if err := r.loadCategoryAttributes(ctx, map[uuid.UUID]*domain.Category{cat.ID: cat}, `
		ca.category_id = $1
	`, id); err != nil {
		return nil, err
	}

	return cat, nil
}

// GetByIDWithInheritedAttributes returns the effective attribute schema for a
// category. Attributes are ordered from the root category to the selected
// category. If an attribute is assigned more than once in the ancestry, the
// closest assignment supplies its required and sort-order settings.
func (r *CategoryRepository) GetByIDWithInheritedAttributes(ctx context.Context, orgID, id uuid.UUID) (*domain.Category, error) {
	cat, err := r.GetByID(ctx, id)
	if err != nil || cat == nil {
		return cat, err
	}
	if cat.OrganizationID != orgID {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE ancestry AS (
			SELECT id, parent_id, 0 AS depth, ARRAY[id] AS path
			FROM categories
			WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT parent.id, parent.parent_id, ancestry.depth + 1, ancestry.path || parent.id
			FROM categories parent
			JOIN ancestry ON parent.id = ancestry.parent_id
			WHERE parent.organization_id = $2
			  AND parent.deleted_at IS NULL
			  AND NOT parent.id = ANY(ancestry.path)
		)
		SELECT ancestry.depth,
		       ca.id, ca.category_id, ca.attribute_id, ca.required, ca.sort_order, ca.created_at,
		       a.id, a.organization_id, a.plugin_id, a.name, a.key, a.data_type, a.created_at, a.updated_at
		FROM ancestry
		JOIN category_attributes ca ON ca.category_id = ancestry.id
		JOIN attributes a ON a.id = ca.attribute_id AND a.deleted_at IS NULL
		WHERE a.organization_id = $2
		ORDER BY ancestry.depth DESC, ca.sort_order, lower(a.name), a.id
	`, id, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type candidate struct {
		depth     int
		attribute domain.CategoryAttribute
	}
	candidates := make([]candidate, 0)
	closestDepth := make(map[uuid.UUID]int)
	for rows.Next() {
		var depth int
		var categoryAttribute domain.CategoryAttribute
		var attribute domain.Attribute
		if err := rows.Scan(
			&depth,
			&categoryAttribute.ID, &categoryAttribute.CategoryID, &categoryAttribute.AttributeID,
			&categoryAttribute.Required, &categoryAttribute.SortOrder, &categoryAttribute.CreatedAt,
			&attribute.ID, &attribute.OrganizationID, &attribute.PluginID, &attribute.Name,
			&attribute.Key, &attribute.DataType, &attribute.CreatedAt, &attribute.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categoryAttribute.Attribute = &attribute
		categoryAttribute.Inherited = depth > 0
		candidates = append(candidates, candidate{depth: depth, attribute: categoryAttribute})
		if previous, ok := closestDepth[categoryAttribute.AttributeID]; !ok || depth < previous {
			closestDepth[categoryAttribute.AttributeID] = depth
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, item := range candidates {
		if item.depth == closestDepth[item.attribute.AttributeID] {
			cat.Attributes = append(cat.Attributes, item.attribute)
		}
	}
	return cat, nil
}

func (r *CategoryRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, icon, created_at, updated_at
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
			&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description, &c.Icon,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(categories) == 0 {
		return categories, nil
	}

	categoriesByID := make(map[uuid.UUID]*domain.Category, len(categories))
	for i := range categories {
		categoriesByID[categories[i].ID] = &categories[i]
	}

	if err := r.loadCategoryAttributes(ctx, categoriesByID, `
		c.organization_id = $1 AND c.deleted_at IS NULL
	`, orgID); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *CategoryRepository) ListTree(ctx context.Context, orgID uuid.UUID) ([]domain.Category, error) {
	categories, err := r.List(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return buildCategoryTree(categories), nil
}

func buildCategoryTree(categories []domain.Category) []domain.Category {
	byID := make(map[uuid.UUID]*domain.Category, len(categories))
	for i := range categories {
		byID[categories[i].ID] = &categories[i]
	}

	childrenByParent := make(map[uuid.UUID][]*domain.Category)
	roots := make([]*domain.Category, 0)
	for i := range categories {
		cat := &categories[i]
		if cat.ParentID == nil || byID[*cat.ParentID] == nil {
			roots = append(roots, cat)
		} else {
			childrenByParent[*cat.ParentID] = append(childrenByParent[*cat.ParentID], cat)
		}
	}

	var build func(*domain.Category, map[uuid.UUID]bool) domain.Category
	build = func(cat *domain.Category, path map[uuid.UUID]bool) domain.Category {
		result := *cat
		result.Children = nil
		if path[cat.ID] {
			return result
		}
		nextPath := make(map[uuid.UUID]bool, len(path)+1)
		for id := range path {
			nextPath[id] = true
		}
		nextPath[cat.ID] = true
		for _, child := range childrenByParent[cat.ID] {
			result.Children = append(result.Children, build(child, nextPath))
		}
		return result
	}

	result := make([]domain.Category, 0, len(roots))
	for _, root := range roots {
		result = append(result, build(root, nil))
	}
	return result
}

// ValidateParent checks that a proposed parent belongs to the same organization
// and is not the category itself or one of its descendants.
func (r *CategoryRepository) ValidateParent(ctx context.Context, orgID, categoryID, parentID uuid.UUID) (bool, error) {
	var valid bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id, ARRAY[id] AS path
			FROM categories
			WHERE id = $2 AND organization_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT parent.id, parent.parent_id, ancestors.path || parent.id
			FROM categories parent
			JOIN ancestors ON parent.id = ancestors.parent_id
			WHERE parent.organization_id = $1
			  AND parent.deleted_at IS NULL
			  AND NOT parent.id = ANY(ancestors.path)
		)
		SELECT EXISTS (SELECT 1 FROM ancestors)
		   AND NOT EXISTS (SELECT 1 FROM ancestors WHERE id = $3)
	`, orgID, parentID, categoryID).Scan(&valid)
	return valid, err
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO categories (id, organization_id, parent_id, plugin_id, name, description, icon)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, query,
		c.ID, c.OrganizationID, c.ParentID, c.PluginID, c.Name, c.Description, c.Icon,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CategoryRepository) GetByPluginID(ctx context.Context, orgID uuid.UUID, pluginID string) (*domain.Category, error) {
	query := `
		SELECT id, organization_id, parent_id, plugin_id, name, description, icon, created_at, updated_at
		FROM categories
		WHERE organization_id = $1 AND plugin_id = $2 AND deleted_at IS NULL
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, orgID, pluginID).Scan(
		&c.ID, &c.OrganizationID, &c.ParentID, &c.PluginID, &c.Name, &c.Description, &c.Icon,
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
		SET parent_id = $2, name = $3, description = $4, icon = $5
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.pool.QueryRow(ctx, query,
		c.ID, c.ParentID, c.Name, c.Description, c.Icon,
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

func (r *CategoryRepository) GetAssetCounts(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT c.id::text, COUNT(a.id)
		FROM categories c
		LEFT JOIN LATERAL (
			WITH RECURSIVE descendants AS (
				SELECT id FROM categories WHERE id = c.id
				UNION
				SELECT child.id FROM categories child
				JOIN descendants parent ON child.parent_id = parent.id
				WHERE child.organization_id = c.organization_id AND child.deleted_at IS NULL
			)
			SELECT id FROM descendants
		) descendant ON TRUE
		LEFT JOIN assets a ON a.category_id = descendant.id AND a.deleted_at IS NULL
		WHERE c.organization_id = $1 AND c.deleted_at IS NULL
		GROUP BY c.id
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var categoryID string
		var count int
		if err := rows.Scan(&categoryID, &count); err != nil {
			return nil, err
		}
		counts[categoryID] = count
	}
	return counts, rows.Err()
}

func (r *CategoryRepository) loadCategoryAttributes(ctx context.Context, categoriesByID map[uuid.UUID]*domain.Category, condition string, args ...any) error {
	query := `
		SELECT ca.id, ca.category_id, ca.attribute_id, ca.required, ca.sort_order, ca.created_at,
		       a.id, a.organization_id, a.plugin_id, a.name, a.key, a.data_type, a.created_at, a.updated_at
		FROM category_attributes ca
		JOIN categories c ON c.id = ca.category_id
		JOIN attributes a ON a.id = ca.attribute_id AND a.deleted_at IS NULL
		WHERE ` + condition + `
		ORDER BY ca.sort_order, a.name
	`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var categoryAttribute domain.CategoryAttribute
		var attribute domain.Attribute
		if err := rows.Scan(
			&categoryAttribute.ID, &categoryAttribute.CategoryID, &categoryAttribute.AttributeID, &categoryAttribute.Required, &categoryAttribute.SortOrder, &categoryAttribute.CreatedAt,
			&attribute.ID, &attribute.OrganizationID, &attribute.PluginID, &attribute.Name, &attribute.Key, &attribute.DataType, &attribute.CreatedAt, &attribute.UpdatedAt,
		); err != nil {
			return err
		}

		category, ok := categoriesByID[categoryAttribute.CategoryID]
		if !ok {
			continue
		}

		categoryAttribute.Attribute = &attribute
		category.Attributes = append(category.Attributes, categoryAttribute)
	}

	return rows.Err()
}
