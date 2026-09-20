package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

var ErrInvalidTags = errors.New("one or more tags do not exist in this workspace")

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

func (r *AssetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Asset, error) {
	query := `
		SELECT id, organization_id, category_id, location_id, condition_id, collection_id, main_attachment_id,
		       name, description, quantity, attributes, purchase_at, purchase_price, purchase_note, notes,
		       import_plugin_id, import_external_id, created_at, updated_at
		FROM assets
		WHERE id = $1 AND deleted_at IS NULL
	`
	var a domain.Asset
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.OrganizationID, &a.CategoryID, &a.LocationID, &a.ConditionID, &a.CollectionID, &a.MainAttachmentID,
		&a.Name, &a.Description, &a.Quantity, &a.Attributes, &a.PurchaseAt, &a.PurchasePrice, &a.PurchaseNote, &a.Notes,
		&a.ImportPluginID, &a.ImportExternalID, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadAssetCollections(ctx, []*domain.Asset{&a}); err != nil {
		return nil, err
	}
	if err := r.loadAssetTags(ctx, []*domain.Asset{&a}); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AssetRepository) GetByIDFull(ctx context.Context, id uuid.UUID) (*domain.Asset, error) {
	asset, err := r.GetByID(ctx, id)
	if err != nil || asset == nil {
		return asset, err
	}

	// Load category
	if asset.CategoryID != nil {
		catQuery := `SELECT id, organization_id, parent_id, plugin_id, name, description, icon, created_at, updated_at FROM categories WHERE id = $1`
		var cat domain.Category
		if err := r.pool.QueryRow(ctx, catQuery, *asset.CategoryID).Scan(
			&cat.ID, &cat.OrganizationID, &cat.ParentID, &cat.PluginID, &cat.Name, &cat.Description, &cat.Icon, &cat.CreatedAt, &cat.UpdatedAt,
		); err == nil {
			asset.Category = &cat
		}
	}

	// Load location if set
	if asset.LocationID != nil {
		locQuery := `SELECT id, organization_id, parent_id, name, description, created_at, updated_at FROM locations WHERE id = $1`
		var loc domain.Location
		if err := r.pool.QueryRow(ctx, locQuery, asset.LocationID).Scan(
			&loc.ID, &loc.OrganizationID, &loc.ParentID, &loc.Name, &loc.Description, &loc.CreatedAt, &loc.UpdatedAt,
		); err == nil {
			asset.Location = &loc
		}
	}

	// Load condition if set
	if asset.ConditionID != nil {
		condQuery := `SELECT id, organization_id, code, label, description, sort_order, created_at, updated_at FROM conditions WHERE id = $1`
		var cond domain.Condition
		if err := r.pool.QueryRow(ctx, condQuery, asset.ConditionID).Scan(
			&cond.ID, &cond.OrganizationID, &cond.Code, &cond.Label, &cond.Description, &cond.SortOrder, &cond.CreatedAt, &cond.UpdatedAt,
		); err == nil {
			asset.Condition = &cond
		}
	}

	// Load warranty
	warQuery := `SELECT id, asset_id, provider, start_date, end_date, notes, created_at, updated_at FROM warranties WHERE asset_id = $1`
	var w domain.Warranty
	if err := r.pool.QueryRow(ctx, warQuery, id).Scan(
		&w.ID, &w.AssetID, &w.Provider, &w.StartDate, &w.EndDate, &w.Notes, &w.CreatedAt, &w.UpdatedAt,
	); err == nil {
		asset.Warranty = &w
	}

	// Load main attachment if set
	if asset.MainAttachmentID != nil {
		attQuery := `
			SELECT id, asset_id, uploaded_by, file_key, file_name, file_size, content_type, description, created_at
			FROM attachments WHERE id = $1
		`
		var att domain.Attachment
		if err := r.pool.QueryRow(ctx, attQuery, asset.MainAttachmentID).Scan(
			&att.ID, &att.AssetID, &att.UploadedBy, &att.FileKey, &att.FileName,
			&att.FileSize, &att.ContentType, &att.Description, &att.CreatedAt,
		); err == nil {
			asset.MainAttachment = &att
		}
	}

	return asset, nil
}

func (r *AssetRepository) List(ctx context.Context, orgID uuid.UUID, filter domain.AssetFilter, page domain.Pagination) ([]domain.Asset, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("a.organization_id = $%d", argNum))
	args = append(args, orgID)
	argNum++

	conditions = append(conditions, "a.deleted_at IS NULL")

	if filter.Uncategorized {
		conditions = append(conditions, "a.category_id IS NULL")
	} else if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			WITH RECURSIVE descendants AS (
				SELECT id FROM categories
				WHERE id = $%d AND organization_id = a.organization_id AND deleted_at IS NULL
				UNION
				SELECT child.id FROM categories child
				JOIN descendants parent ON child.parent_id = parent.id
				WHERE child.organization_id = a.organization_id AND child.deleted_at IS NULL
			)
			SELECT 1 FROM descendants WHERE descendants.id = a.category_id
		)`, argNum))
		args = append(args, *filter.CategoryID)
		argNum++
	}
	if filter.LocationID != nil {
		conditions = append(conditions, fmt.Sprintf("a.location_id = $%d", argNum))
		args = append(args, *filter.LocationID)
		argNum++
	}
	if filter.ConditionID != nil {
		conditions = append(conditions, fmt.Sprintf("a.condition_id = $%d", argNum))
		args = append(args, *filter.ConditionID)
		argNum++
	}
	if filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf(`(a.search_vector @@ attic_prefix_tsquery($%d)
            OR EXISTS (SELECT 1 FROM asset_tags at JOIN tags t ON t.id=at.tag_id
                WHERE at.asset_id=a.id AND t.organization_id=a.organization_id
                AND to_tsvector('english', t.name) @@ attic_prefix_tsquery($%d)))`, argNum, argNum))
		args = append(args, filter.Query)
		argNum++
	}
	if len(filter.TagIDs) > 0 {
		operator := "> 0"
		if filter.TagMatch == "all" {
			operator = fmt.Sprintf("= %d", len(filter.TagIDs))
		}
		conditions = append(conditions, fmt.Sprintf(`(SELECT COUNT(DISTINCT at.tag_id) FROM asset_tags at
            JOIN tags t ON t.id=at.tag_id AND t.organization_id=a.organization_id
            WHERE at.asset_id=a.id AND at.tag_id=ANY($%d)) %s`, argNum, operator))
		args = append(args, filter.TagIDs)
		argNum++
	}
	if filter.CollectionID != nil {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (SELECT 1 FROM asset_collections ac
            JOIN collections cl ON cl.id = ac.collection_id AND cl.organization_id = a.organization_id
            WHERE ac.asset_id = a.id AND ac.collection_id = $%d)`, argNum))
		args = append(args, *filter.CollectionID)
		argNum++
	}

	if filter.Criteria != nil {
		if filter.Features == nil {
			return nil, 0, errors.New("filter feature settings are required")
		}
		predicate, filterArgs, err := r.CompileCriteria(ctx, orgID, *filter.Criteria, *filter.Features, argNum)
		if err != nil {
			return nil, 0, err
		}
		conditions = append(conditions, predicate)
		args = append(args, filterArgs...)
		argNum += len(filterArgs)
	}
	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM assets a WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get assets with related data
	query := fmt.Sprintf(`
		SELECT a.id, a.organization_id, a.category_id, a.location_id, a.condition_id, a.collection_id, a.main_attachment_id,
		       a.name, a.description, a.quantity, a.attributes, a.purchase_at, a.purchase_price, a.purchase_note, a.notes, a.created_at, a.updated_at,
		       c.id, c.name, c.plugin_id,
		       l.id, l.name,
		       cond.id, cond.code, cond.label,
		       att.id, att.file_key, att.file_name, att.content_type
		FROM assets a
		LEFT JOIN categories c ON c.id = a.category_id AND c.deleted_at IS NULL
		LEFT JOIN locations l ON l.id = a.location_id AND l.deleted_at IS NULL
		LEFT JOIN conditions cond ON cond.id = a.condition_id AND cond.deleted_at IS NULL
		LEFT JOIN attachments att ON att.id = a.main_attachment_id
		WHERE %s
		ORDER BY a.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, page.Limit, page.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []domain.Asset
	for rows.Next() {
		var a domain.Asset
		var catID, catName, catPluginID *string
		var locID, locName *string
		var condID, condCode, condLabel *string
		var attID, attFileKey, attFileName, attContentType *string

		if err := rows.Scan(
			&a.ID, &a.OrganizationID, &a.CategoryID, &a.LocationID, &a.ConditionID, &a.CollectionID, &a.MainAttachmentID,
			&a.Name, &a.Description, &a.Quantity, &a.Attributes, &a.PurchaseAt, &a.PurchasePrice, &a.PurchaseNote, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
			&catID, &catName, &catPluginID,
			&locID, &locName,
			&condID, &condCode, &condLabel,
			&attID, &attFileKey, &attFileName, &attContentType,
		); err != nil {
			return nil, 0, err
		}

		// Populate category
		if catID != nil && catName != nil {
			a.Category = &domain.Category{
				ID:       uuid.MustParse(*catID),
				Name:     *catName,
				PluginID: catPluginID,
			}
		}

		// Populate location
		if locID != nil && locName != nil {
			a.Location = &domain.Location{
				ID:   uuid.MustParse(*locID),
				Name: *locName,
			}
		}

		// Populate condition
		if condID != nil && condCode != nil && condLabel != nil {
			a.Condition = &domain.Condition{
				ID:    uuid.MustParse(*condID),
				Code:  *condCode,
				Label: *condLabel,
			}
		}

		// Populate main attachment
		if attID != nil && attFileKey != nil && attFileName != nil {
			a.MainAttachment = &domain.Attachment{
				ID:          uuid.MustParse(*attID),
				FileKey:     *attFileKey,
				FileName:    *attFileName,
				ContentType: attContentType,
			}
		}

		assets = append(assets, a)
	}

	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	assetPointers := make([]*domain.Asset, len(assets))
	for i := range assets {
		assetPointers[i] = &assets[i]
	}
	if err := r.loadAssetCollections(ctx, assetPointers); err != nil {
		return nil, 0, err
	}
	if err := r.loadAssetTags(ctx, assetPointers); err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

func (r *AssetRepository) Search(ctx context.Context, orgID uuid.UUID, query string, page domain.Pagination) ([]domain.Asset, int, error) {
	filter := domain.AssetFilter{Query: query}
	return r.List(ctx, orgID, filter, page)
}

func (r *AssetRepository) Create(ctx context.Context, a *domain.Asset) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = validateSelectAsset(ctx, tx, a); err != nil {
		return err
	}
	query := `
		INSERT INTO assets (id, organization_id, category_id, location_id, condition_id, collection_id,
		                    name, description, quantity, attributes, purchase_at, purchase_price, purchase_note, notes,
		                    import_plugin_id, import_external_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING created_at, updated_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Attributes == nil {
		a.Attributes = []byte("{}")
	}
	if err := tx.QueryRow(ctx, query,
		a.ID, a.OrganizationID, a.CategoryID, a.LocationID, a.ConditionID, a.CollectionID,
		a.Name, a.Description, a.Quantity, a.Attributes, a.PurchaseAt, a.PurchasePrice, a.PurchaseNote, a.Notes,
		a.ImportPluginID, a.ImportExternalID,
	).Scan(&a.CreatedAt, &a.UpdatedAt); err != nil {
		return err
	}
	if a.CollectionIDs == nil {
		a.CollectionIDs = []uuid.UUID{}
	}
	if err := replaceAssetCollections(ctx, tx, a); err != nil {
		return err
	}
	if err := replaceAssetTags(ctx, tx, a); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AssetRepository) Update(ctx context.Context, a *domain.Asset) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = validateSelectAsset(ctx, tx, a); err != nil {
		return err
	}
	query := `
		UPDATE assets
		SET category_id = $2, location_id = $3, condition_id = $4, collection_id = $5,
		    name = $6, description = $7, quantity = $8, attributes = $9, purchase_at = $10, purchase_price = $11, purchase_note = $12, notes = $13
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	if a.Attributes == nil {
		a.Attributes = []byte("{}")
	}
	if err := tx.QueryRow(ctx, query,
		a.ID, a.CategoryID, a.LocationID, a.ConditionID, a.CollectionID,
		a.Name, a.Description, a.Quantity, a.Attributes, a.PurchaseAt, a.PurchasePrice, a.PurchaseNote, a.Notes,
	).Scan(&a.UpdatedAt); err != nil {
		return err
	}
	if err := replaceAssetCollections(ctx, tx, a); err != nil {
		return err
	}
	if err := replaceAssetTags(ctx, tx, a); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if err := r.loadAssetCollections(ctx, []*domain.Asset{a}); err != nil {
		return err
	}
	return r.loadAssetTags(ctx, []*domain.Asset{a})
}

func (r *AssetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var org uuid.UUID
	if err = tx.QueryRow(ctx, "SELECT organization_id FROM assets WHERE id=$1", id).Scan(&org); err == pgx.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	if err = lockAttributeWrites(ctx, tx, org); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE assets SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AssetRepository) SetTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	asset := &domain.Asset{ID: assetID, TagIDs: tagIDs}
	if err := tx.QueryRow(ctx, `SELECT organization_id FROM assets WHERE id=$1 AND deleted_at IS NULL`, assetID).Scan(&asset.OrganizationID); err != nil {
		return err
	}
	if err := replaceAssetTags(ctx, tx, asset); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func replaceAssetTags(ctx context.Context, tx pgx.Tx, asset *domain.Asset) error {
	if asset.TagIDs == nil && asset.NewTagNames == nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(asset.TagIDs))
	seenIDs := map[uuid.UUID]bool{}
	for _, id := range asset.TagIDs {
		if !seenIDs[id] {
			ids = append(ids, id)
			seenIDs[id] = true
		}
	}
	tags := []domain.Tag{}
	if len(ids) > 0 {
		rows, err := tx.Query(ctx, `SELECT id, organization_id, name, description, created_at, updated_at, 0
            FROM tags WHERE organization_id=$1 AND id=ANY($2) ORDER BY lower(name), id FOR SHARE`, asset.OrganizationID, ids)
		if err != nil {
			return err
		}
		for rows.Next() {
			var tag domain.Tag
			if err := scanTag(rows, &tag); err != nil {
				rows.Close()
				return err
			}
			tags = append(tags, tag)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(tags) != len(ids) {
			return ErrInvalidTags
		}
	}
	seenNames := map[string]bool{}
	for _, tag := range tags {
		seenNames[strings.ToLower(tag.Name)] = true
	}
	for _, name := range asset.NewTagNames {
		key := strings.ToLower(name)
		if seenNames[key] {
			continue
		}
		id := uuid.New()
		_, err := tx.Exec(ctx, `INSERT INTO tags(id, organization_id, name) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, id, asset.OrganizationID, name)
		if err != nil {
			return err
		}
		var tag domain.Tag
		if err := tx.QueryRow(ctx, `SELECT id, organization_id, name, description, created_at, updated_at, 0
            FROM tags WHERE organization_id=$1 AND lower(name)=lower($2)`, asset.OrganizationID, name).Scan(
			&tag.ID, &tag.OrganizationID, &tag.Name, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt, &tag.AssetCount); err != nil {
			return err
		}
		tags = append(tags, tag)
		ids = append(ids, tag.ID)
		seenIDs[tag.ID] = true
		seenNames[key] = true
	}
	if len(ids) > 50 {
		return errors.New("assign at most 50 tags")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM asset_tags WHERE asset_id=$1`, asset.ID); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `INSERT INTO asset_tags(asset_id, tag_id) VALUES($1,$2)`, asset.ID, id); err != nil {
			return err
		}
	}
	asset.TagIDs, asset.Tags = ids, tags
	return nil
}

func (r *AssetRepository) loadAssetTags(ctx context.Context, assets []*domain.Asset) error {
	if len(assets) == 0 {
		return nil
	}
	byID := map[uuid.UUID]*domain.Asset{}
	ids := make([]uuid.UUID, 0, len(assets))
	for _, asset := range assets {
		asset.TagIDs = []uuid.UUID{}
		asset.Tags = []domain.Tag{}
		byID[asset.ID] = asset
		ids = append(ids, asset.ID)
	}
	rows, err := r.pool.Query(ctx, `SELECT at.asset_id, t.id, t.organization_id, t.name, t.description, t.created_at, t.updated_at
        FROM asset_tags at JOIN tags t ON t.id=at.tag_id WHERE at.asset_id=ANY($1) ORDER BY lower(t.name), t.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var assetID uuid.UUID
		var tag domain.Tag
		if err := rows.Scan(&assetID, &tag.ID, &tag.OrganizationID, &tag.Name, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return err
		}
		if asset := byID[assetID]; asset != nil && asset.OrganizationID == tag.OrganizationID {
			asset.TagIDs = append(asset.TagIDs, tag.ID)
			asset.Tags = append(asset.Tags, tag)
		}
	}
	return rows.Err()
}

func (r *AssetRepository) GetTotalValue(ctx context.Context, orgID uuid.UUID, filter domain.AssetFilter) (float64, error) {
	query := `
		SELECT COALESCE(SUM(purchase_price * quantity), 0)
		FROM assets
		WHERE organization_id = $1 AND deleted_at IS NULL
	`
	args := []any{orgID}
	if filter.LocationID != nil {
		query += " AND location_id = $2"
		args = append(args, *filter.LocationID)
	}
	var total float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	return total, err
}

func (r *AssetRepository) SetMainAttachment(ctx context.Context, assetID uuid.UUID, attachmentID *uuid.UUID) error {
	query := `
		UPDATE assets
		SET main_attachment_id = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, assetID, attachmentID)
	return err
}

func (r *AssetRepository) GetMainAttachmentID(ctx context.Context, assetID uuid.UUID) (*uuid.UUID, error) {
	query := `SELECT main_attachment_id FROM assets WHERE id = $1 AND deleted_at IS NULL`
	var mainID *uuid.UUID
	err := r.pool.QueryRow(ctx, query, assetID).Scan(&mainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mainID, nil
}
