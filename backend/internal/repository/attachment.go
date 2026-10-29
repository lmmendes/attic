package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

type AttachmentRepository struct {
	pool *pgxpool.Pool
}

func NewAttachmentRepository(pool *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{pool: pool}
}

func (r *AttachmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	query := `
		SELECT id, asset_id, uploaded_by, file_key, file_name, file_size, content_type, description, created_at
		FROM attachments
		WHERE id = $1
	`
	var a domain.Attachment
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.AssetID, &a.UploadedBy, &a.FileKey, &a.FileName,
		&a.FileSize, &a.ContentType, &a.Description, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepository) ListByAsset(ctx context.Context, assetID uuid.UUID) ([]domain.Attachment, error) {
	query := `
		SELECT id, asset_id, uploaded_by, file_key, file_name, file_size, content_type, description, created_at
		FROM attachments
		WHERE asset_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []domain.Attachment
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(
			&a.ID, &a.AssetID, &a.UploadedBy, &a.FileKey, &a.FileName,
			&a.FileSize, &a.ContentType, &a.Description, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

func (r *AttachmentRepository) Create(ctx context.Context, a *domain.Attachment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Serialize uploads for the same asset so concurrent images cannot both
	// claim the default, or overwrite a user's selection.
	var assetID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM assets WHERE id = $1 FOR UPDATE`, a.AssetID).Scan(&assetID); err != nil {
		return err
	}

	query := `
		INSERT INTO attachments (id, asset_id, uploaded_by, file_key, file_name, file_size, content_type, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if err := tx.QueryRow(ctx, query,
		a.ID, a.AssetID, a.UploadedBy, a.FileKey, a.FileName, a.FileSize, a.ContentType, a.Description,
	).Scan(&a.CreatedAt); err != nil {
		return err
	}

	if a.ContentType != nil && strings.HasPrefix(*a.ContentType, "image/") {
		// Existing images also preserve an explicitly cleared default.
		_, err := tx.Exec(ctx, `
			UPDATE assets SET main_attachment_id = $2
			WHERE id = $1 AND deleted_at IS NULL AND main_attachment_id IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM attachments
				WHERE asset_id = $1 AND id <> $2 AND content_type LIKE 'image/%'
			)
		`, a.AssetID, a.ID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *AttachmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM attachments WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
