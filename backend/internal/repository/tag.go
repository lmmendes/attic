package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type TagRepository struct{ pool *pgxpool.Pool }

func NewTagRepository(pool *pgxpool.Pool) *TagRepository { return &TagRepository{pool: pool} }

const tagSelect = `SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at,
    COUNT(a.id) FROM tags t
    LEFT JOIN asset_tags at ON at.tag_id = t.id
    LEFT JOIN assets a ON a.id = at.asset_id AND a.organization_id = t.organization_id AND a.deleted_at IS NULL `

func (r *TagRepository) List(ctx context.Context, orgID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, tagSelect+`WHERE t.organization_id=$1 GROUP BY t.id ORDER BY lower(t.name), t.id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []domain.Tag{}
	for rows.Next() {
		var tag domain.Tag
		if err := scanTag(rows, &tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *TagRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*domain.Tag, error) {
	var tag domain.Tag
	err := scanTag(r.pool.QueryRow(ctx, tagSelect+`WHERE t.organization_id=$1 AND t.id=$2 GROUP BY t.id`, orgID, id), &tag)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &tag, err
}

func (r *TagRepository) GetByName(ctx context.Context, orgID uuid.UUID, name string) (*domain.Tag, error) {
	var tag domain.Tag
	err := scanTag(r.pool.QueryRow(ctx, tagSelect+`WHERE t.organization_id=$1 AND lower(t.name)=lower($2) GROUP BY t.id`, orgID, name), &tag)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &tag, err
}

func (r *TagRepository) ListByAsset(ctx context.Context, assetID uuid.UUID) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `SELECT t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at, 0
        FROM tags t JOIN asset_tags at ON at.tag_id=t.id WHERE at.asset_id=$1 ORDER BY lower(t.name), t.id`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []domain.Tag{}
	for rows.Next() {
		var tag domain.Tag
		if err := scanTag(rows, &tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *TagRepository) Create(ctx context.Context, tag *domain.Tag) error {
	if tag.ID == uuid.Nil {
		tag.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, `INSERT INTO tags(id, organization_id, name, description)
        VALUES($1,$2,$3,$4) RETURNING created_at, updated_at`, tag.ID, tag.OrganizationID, tag.Name, tag.Description).
		Scan(&tag.CreatedAt, &tag.UpdatedAt)
}

func (r *TagRepository) GetOrCreate(ctx context.Context, orgID uuid.UUID, name string) (*domain.Tag, error) {
	if existing, err := r.GetByName(ctx, orgID, name); err != nil || existing != nil {
		return existing, err
	}
	tag := &domain.Tag{OrganizationID: orgID, Name: name}
	if err := r.Create(ctx, tag); err != nil {
		if existing, lookupErr := r.GetByName(ctx, orgID, name); lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return tag, nil
}

func (r *TagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	return r.pool.QueryRow(ctx, `UPDATE tags SET name=$3, description=$4 WHERE organization_id=$1 AND id=$2 RETURNING updated_at`,
		tag.OrganizationID, tag.ID, tag.Name, tag.Description).Scan(&tag.UpdatedAt)
}

func (r *TagRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE organization_id=$1 AND id=$2`, orgID, id)
	if err == nil && result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func scanTag(row pgx.Row, tag *domain.Tag) error {
	return row.Scan(&tag.ID, &tag.OrganizationID, &tag.Name, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt, &tag.AssetCount)
}
