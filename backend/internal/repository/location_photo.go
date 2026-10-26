package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

// lockPhotoLocation serializes all gallery mutations, including location deletion.
func lockPhotoLocation(ctx context.Context, tx pgx.Tx, org, id uuid.UUID) error {
	var found uuid.UUID
	return tx.QueryRow(ctx, `SELECT id FROM locations WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL FOR UPDATE`, id, org).Scan(&found)
}

func (r *LocationRepository) ListPhotos(ctx context.Context, org, id uuid.UUID) ([]domain.LocationPhoto, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id,p.location_id,p.file_key,p.file_name,p.is_cover,p.created_at FROM location_photos p JOIN locations l ON l.id=p.location_id WHERE l.id=$1 AND l.organization_id=$2 AND l.deleted_at IS NULL ORDER BY p.is_cover DESC,p.created_at,p.id`, id, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photos := []domain.LocationPhoto{}
	for rows.Next() {
		var p domain.LocationPhoto
		if err := rows.Scan(&p.ID, &p.LocationID, &p.FileKey, &p.FileName, &p.IsCover, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

// AddPhoto appends an image without changing the selected cover.
func (r *LocationRepository) AddPhoto(ctx context.Context, org uuid.UUID, p *domain.LocationPhoto) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockPhotoLocation(ctx, tx, org, p.LocationID); err != nil {
		return err
	}
	p.ID = uuid.New()
	err = tx.QueryRow(ctx, `INSERT INTO location_photos(id,location_id,file_key,file_name,is_cover) VALUES($1,$2,$3,$4,false) RETURNING is_cover,created_at`, p.ID, p.LocationID, p.FileKey, p.FileName).Scan(&p.IsCover, &p.CreatedAt)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE locations SET updated_at=now() WHERE id=$1`, p.LocationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *LocationRepository) SetCoverPhoto(ctx context.Context, org, id, photoID uuid.UUID, selected bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockPhotoLocation(ctx, tx, org, id); err != nil {
		return err
	}
	var found uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM location_photos WHERE id=$1 AND location_id=$2`, photoID, id).Scan(&found); err != nil {
		return err
	}
	if selected {
		if _, err = tx.Exec(ctx, `UPDATE location_photos SET is_cover=false WHERE location_id=$1 AND is_cover`, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE location_photos SET is_cover=$2 WHERE id=$1`, photoID, selected); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE locations SET updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeletePhoto removes an image without selecting a replacement cover.
func (r *LocationRepository) DeletePhoto(ctx context.Context, org, id, photoID uuid.UUID) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err = lockPhotoLocation(ctx, tx, org, id); err != nil {
		return "", err
	}
	var key string
	if err = tx.QueryRow(ctx, `DELETE FROM location_photos WHERE id=$1 AND location_id=$2 RETURNING file_key`, photoID, id).Scan(&key); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `UPDATE locations SET updated_at=now() WHERE id=$1`, id); err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

// DeleteWithPhotos removes the location and its gallery atomically, returning
// storage keys for cleanup only after the transaction commits.
func (r *LocationRepository) DeleteWithPhotos(ctx context.Context, org, id uuid.UUID) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockPhotoLocation(ctx, tx, org, id); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `DELETE FROM location_photos WHERE location_id=$1 RETURNING file_key`, id)
	if err != nil {
		return nil, err
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, key)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE locations SET deleted_at=now() WHERE id=$1`, id); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return keys, nil
}
