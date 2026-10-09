package repository

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
)

type ContainmentError struct {
	Status  int
	Message string
}

func (e *ContainmentError) Error() string { return e.Message }

func (r *AssetRepository) SetParent(ctx context.Context, org, id uuid.UUID, parent *uuid.UUID) (*domain.Asset, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockAttributeReads(ctx, tx, org); err != nil {
		return nil, err
	}
	a := &domain.Asset{ID: id, OrganizationID: org, ParentID: parent, ParentProvided: true, PreserveLocation: true}
	if err = prepareAssetContainment(ctx, tx, a, false); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE assets SET parent_id=$3, location_id=$4 WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL`, id, org, a.ParentID, a.LocationID); err != nil {
		return nil, err
	}
	if err = applyAssetContainment(ctx, tx, a); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByIDFull(ctx, id)
}

func (r *AssetRepository) GetContainmentSummary(ctx context.Context, org, id uuid.UUID) (*domain.ContainmentSummary, error) {
	var summary domain.ContainmentSummary
	err := r.pool.QueryRow(ctx, `WITH RECURSIVE contents AS (
  SELECT id, purchase_price, quantity FROM assets WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL
  UNION SELECT a.id, a.purchase_price, a.quantity FROM assets a JOIN contents c ON a.parent_id=c.id
  WHERE a.organization_id=$2 AND a.deleted_at IS NULL
 ) SELECT COALESCE(SUM(purchase_price*quantity),0), COUNT(*) FILTER (WHERE purchase_price IS NULL) FROM contents`, id, org).Scan(&summary.TotalValue, &summary.UnpricedAssetCount)
	return &summary, err
}

func invalidContainment(message string) error {
	return &ContainmentError{http.StatusBadRequest, message}
}

func lockContainment(ctx context.Context, tx pgx.Tx, org uuid.UUID) error {
	// Attribute locks must always precede this lock, including soft deletion.
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 111))`, org.String())
	return err
}

func prepareAssetContainment(ctx context.Context, tx pgx.Tx, a *domain.Asset, creating bool) error {
	if err := lockContainment(ctx, tx, a.OrganizationID); err != nil {
		return err
	}
	if !creating {
		var parent, location *uuid.UUID
		err := tx.QueryRow(ctx, `SELECT parent_id, location_id FROM assets WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL`, a.ID, a.OrganizationID).Scan(&parent, &location)
		if errors.Is(err, pgx.ErrNoRows) {
			return &ContainmentError{http.StatusNotFound, "asset not found"}
		}
		if err != nil {
			return err
		}
		if !a.ParentProvided {
			a.ParentID = parent
		}
		// A detach keeps the actual location under the lock, not a stale form value.
		if a.PreserveLocation || (parent != nil && a.ParentID == nil && !a.LocationProvided) {
			a.LocationID = location
		}
	}
	if a.ParentID == nil {
		return nil
	}
	if *a.ParentID == a.ID {
		return invalidContainment("an asset cannot contain itself")
	}
	var location *uuid.UUID
	err := tx.QueryRow(ctx, `SELECT location_id FROM assets WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL`, a.ParentID, a.OrganizationID).Scan(&location)
	if errors.Is(err, pgx.ErrNoRows) {
		return invalidContainment("parent asset is unavailable")
	}
	if err != nil {
		return err
	}
	if (a.LocationProvided || (creating && a.LocationID != nil)) && !sameUUID(a.LocationID, location) {
		return invalidContainment("contained assets must use their parent's location")
	}
	a.LocationID = location
	return nil
}

func applyAssetContainment(ctx context.Context, tx pgx.Tx, a *domain.Asset) error {
	additions := map[uuid.UUID]bool{}
	for _, id := range a.AddChildIDs {
		if id == a.ID {
			return invalidContainment("an asset cannot contain itself")
		}
		additions[id] = true
	}
	for _, id := range a.RemoveChildIDs {
		if additions[id] {
			return invalidContainment("an asset cannot be added and removed in the same save")
		}
		result, err := tx.Exec(ctx, `UPDATE assets SET parent_id=NULL WHERE id=$1 AND parent_id=$2 AND organization_id=$3 AND deleted_at IS NULL`, id, a.ID, a.OrganizationID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return &ContainmentError{http.StatusConflict, "contents changed; reload before saving"}
		}
	}
	for id := range additions {
		result, err := tx.Exec(ctx, `UPDATE assets SET parent_id=$2 WHERE id=$1 AND organization_id=$3 AND deleted_at IS NULL`, id, a.ID, a.OrganizationID)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return invalidContainment("content asset is unavailable")
		}
	}
	// Validate the final graph so a single save can detach and reparent together.
	changed := append([]uuid.UUID{a.ID}, a.AddChildIDs...)
	for _, id := range changed {
		var cyclic bool
		err := tx.QueryRow(ctx, `WITH RECURSIVE ancestors AS (
   SELECT p.id, p.parent_id FROM assets child JOIN assets p ON p.id=child.parent_id
   WHERE child.id=$1 AND child.organization_id=$2 AND p.organization_id=$2 AND p.deleted_at IS NULL
   UNION SELECT p.id,p.parent_id FROM assets p JOIN ancestors c ON p.id=c.parent_id
   WHERE p.organization_id=$2 AND p.deleted_at IS NULL
  ) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id=$1)`, id, a.OrganizationID).Scan(&cyclic)
		if err != nil {
			return err
		}
		if cyclic {
			return invalidContainment("asset containment cannot form a cycle")
		}
	}
	_, err := tx.Exec(ctx, `WITH RECURSIVE contents AS (
  SELECT id FROM assets WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL
  UNION SELECT child.id FROM assets child JOIN contents parent ON child.parent_id=parent.id
  WHERE child.organization_id=$2 AND child.deleted_at IS NULL
 ) UPDATE assets SET location_id=$3 WHERE id IN (SELECT id FROM contents) AND location_id IS DISTINCT FROM $3`, a.ID, a.OrganizationID, a.LocationID)
	return err
}

func sameUUID(a, b *uuid.UUID) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func (r *AssetRepository) loadAssetParents(ctx context.Context, assets []*domain.Asset) error {
	if len(assets) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(assets))
	byID := map[uuid.UUID]*domain.Asset{}
	for i, a := range assets {
		ids[i] = a.ID
		byID[a.ID] = a
		a.Parent = nil
	}
	rows, err := r.pool.Query(ctx, `SELECT child.id, parent.id, parent.name FROM assets child JOIN assets parent ON parent.id=child.parent_id
  AND parent.organization_id=child.organization_id AND parent.deleted_at IS NULL WHERE child.id=ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var parent domain.AssetReference
		if err = rows.Scan(&id, &parent.ID, &parent.Name); err != nil {
			return err
		}
		byID[id].Parent = &parent
	}
	return rows.Err()
}
