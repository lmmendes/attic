package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

var ErrInvalidCollections = errors.New("one or more collections do not exist in this workspace")

type CollectionRepository struct{ pool *pgxpool.Pool }

func NewCollectionRepository(pool *pgxpool.Pool) *CollectionRepository {
	return &CollectionRepository{pool: pool}
}

const collectionSelect = `SELECT c.id, c.organization_id, c.name, c.description, c.icon,
    c.created_at, c.updated_at, COUNT(a.id)
    FROM collections c
    LEFT JOIN asset_collections ac ON ac.collection_id = c.id
    LEFT JOIN assets a ON a.id = ac.asset_id AND a.organization_id = c.organization_id AND a.deleted_at IS NULL `

func (r *CollectionRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Collection, error) {
	rows, err := r.pool.Query(ctx, collectionSelect+`WHERE c.organization_id = $1 GROUP BY c.id ORDER BY lower(c.name), c.id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	collections := []domain.Collection{}
	for rows.Next() {
		var c domain.Collection
		if err := scanCollection(rows, &c); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, rows.Err()
}

func (r *CollectionRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Collection, error) {
	var c domain.Collection
	err := scanCollection(r.pool.QueryRow(ctx, collectionSelect+`WHERE c.organization_id = $1 AND c.id = $2 GROUP BY c.id`, orgID, id), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) Create(ctx context.Context, c *domain.Collection) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, `INSERT INTO collections (id, organization_id, name, description, icon)
        VALUES ($1, $2, $3, $4, $5) RETURNING created_at, updated_at`,
		c.ID, c.OrganizationID, c.Name, c.Description, c.Icon).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CollectionRepository) Update(ctx context.Context, c *domain.Collection) error {
	return r.pool.QueryRow(ctx, `UPDATE collections SET name = $3, description = $4, icon = $5
        WHERE organization_id = $1 AND id = $2 RETURNING updated_at`,
		c.OrganizationID, c.ID, c.Name, c.Description, c.Icon).Scan(&c.UpdatedAt)
}

func (r *CollectionRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM collections WHERE organization_id = $1 AND id = $2`, orgID, id)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func scanCollection(row pgx.Row, c *domain.Collection) error {
	return row.Scan(&c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.Icon, &c.CreatedAt, &c.UpdatedAt, &c.AssetCount)
}

// replaceAssetCollections runs in the asset transaction, so a bad membership
// cannot partially save an asset or clear its existing memberships.
func replaceAssetCollections(ctx context.Context, tx pgx.Tx, a *domain.Asset) error {
	if a.CollectionIDs == nil {
		return nil
	} // Omission preserves memberships; an empty array clears them.
	ids := make([]uuid.UUID, 0, len(a.CollectionIDs))
	seen := make(map[uuid.UUID]bool)
	for _, id := range a.CollectionIDs {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, organization_id, name, description, icon, created_at, updated_at
        FROM collections WHERE organization_id = $1 AND id = ANY($2) ORDER BY id FOR SHARE`, a.OrganizationID, ids)
	if err != nil {
		return err
	}
	collections := []domain.Collection{}
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(&c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.Icon, &c.CreatedAt, &c.UpdatedAt); err != nil {
			rows.Close()
			return err
		}
		collections = append(collections, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(collections) != len(ids) {
		return ErrInvalidCollections
	}
	if _, err := tx.Exec(ctx, `DELETE FROM asset_collections WHERE asset_id = $1`, a.ID); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `INSERT INTO asset_collections (asset_id, collection_id) VALUES ($1, $2)`, a.ID, id); err != nil {
			return err
		}
	}
	a.CollectionIDs = ids
	a.Collections = collections
	return nil
}

// loadAssetCollections uses one query for the whole page, avoiding a query per asset.
func (r *AssetRepository) loadAssetCollections(ctx context.Context, assets []*domain.Asset) error {
	if len(assets) == 0 {
		return nil
	}
	byID := make(map[uuid.UUID]*domain.Asset, len(assets))
	ids := make([]uuid.UUID, 0, len(assets))
	for _, a := range assets {
		a.CollectionIDs = []uuid.UUID{}
		a.Collections = []domain.Collection{}
		byID[a.ID] = a
		ids = append(ids, a.ID)
	}
	rows, err := r.pool.Query(ctx, `SELECT ac.asset_id, c.id, c.organization_id, c.name, c.description, c.icon, c.created_at, c.updated_at
        FROM asset_collections ac JOIN collections c ON c.id = ac.collection_id
        WHERE ac.asset_id = ANY($1) ORDER BY lower(c.name), c.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var assetID uuid.UUID
		var c domain.Collection
		if err := rows.Scan(&assetID, &c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.Icon, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return err
		}
		if a := byID[assetID]; a != nil && a.OrganizationID == c.OrganizationID {
			a.CollectionIDs = append(a.CollectionIDs, c.ID)
			a.Collections = append(a.Collections, c)
		}
	}
	return rows.Err()
}
