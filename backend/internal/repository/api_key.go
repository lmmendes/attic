package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type APIKeyRepository struct{ pool *pgxpool.Pool }

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository { return &APIKeyRepository{pool} }

func (r *APIKeyRepository) Create(ctx context.Context, k *domain.APIKey) error {
	return r.pool.QueryRow(ctx, `INSERT INTO api_keys(id,owner_id,organization_id,description,permissions,secret_hash,prefix,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING created_at`, k.ID, k.OwnerID, k.OrganizationID, k.Description, k.Permissions, k.SecretHash, k.Prefix, k.ExpiresAt).Scan(&k.CreatedAt)
}

func (r *APIKeyRepository) Authenticate(ctx context.Context, secret string) (*domain.APIKey, error) {
	k := new(domain.APIKey)
	err := r.pool.QueryRow(ctx, `UPDATE api_keys SET last_used_at=now() WHERE secret_hash=$1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now()) RETURNING id,owner_id,organization_id,permissions`, domain.HashAPIKey(secret)).Scan(&k.ID, &k.OwnerID, &k.OrganizationID, &k.Permissions)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

func (r *APIKeyRepository) List(ctx context.Context, org uuid.UUID, owner uuid.UUID) ([]domain.APIKey, error) {
	rows, err := r.pool.Query(ctx, `SELECT k.id,k.owner_id,k.organization_id,k.description,k.permissions,k.prefix,k.created_at,k.expires_at,k.revoked_at,k.last_used_at FROM api_keys k WHERE k.organization_id=$1 AND k.owner_id=$2 AND k.revoked_at IS NULL AND (k.expires_at IS NULL OR k.expires_at>now()) ORDER BY k.created_at DESC`, org, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []domain.APIKey{}
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.OwnerID, &k.OrganizationID, &k.Description, &k.Permissions, &k.Prefix, &k.CreatedAt, &k.ExpiresAt, &k.RevokedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
func (r *APIKeyRepository) Rename(ctx context.Context, org, owner, id uuid.UUID, description string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE api_keys SET description=$4 WHERE organization_id=$1 AND owner_id=$2 AND id=$3`, org, owner, id, description)
	return tag.RowsAffected() > 0, err
}
func (r *APIKeyRepository) Revoke(ctx context.Context, org uuid.UUID, owner uuid.UUID, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE api_keys SET revoked_at=COALESCE(revoked_at,now()) WHERE organization_id=$1 AND owner_id=$2 AND id=$3`, org, owner, id)
	return tag.RowsAffected() > 0, err
}
func (r *APIKeyRepository) MCPEnabled(ctx context.Context, org uuid.UUID) (bool, error) {
	var enabled bool
	err := r.pool.QueryRow(ctx, `SELECT mcp_enabled FROM organizations WHERE id=$1`, org).Scan(&enabled)
	return enabled, err
}
func (r *APIKeyRepository) SetMCPEnabled(ctx context.Context, org uuid.UUID, enabled bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE organizations SET mcp_enabled=$2 WHERE id=$1`, org, enabled)
	return err
}
