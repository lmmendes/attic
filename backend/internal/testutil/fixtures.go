package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmmendes/attic/internal/domain"
)

// Fixtures provides methods for creating test data
type Fixtures struct {
	pool *pgxpool.Pool
}

// NewFixtures creates a new Fixtures instance
func NewFixtures(pool *pgxpool.Pool) *Fixtures {
	return &Fixtures{pool: pool}
}

// CreateOrganization creates a test organization
func (f *Fixtures) CreateOrganization(ctx context.Context, name string) (*domain.Organization, error) {
	org := &domain.Organization{
		ID:   uuid.New(),
		Name: name,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, org.ID, org.Name)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// CreateUser creates a test user
func (f *Fixtures) CreateUser(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error) {
	user := &domain.User{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Email:          email,
		Role:           domain.UserRoleUser,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO users (id, organization_id, email, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, user.OrganizationID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateCategory creates a test category
func (f *Fixtures) CreateCategory(ctx context.Context, orgID uuid.UUID, name string, parentID *uuid.UUID) (*domain.Category, error) {
	cat := &domain.Category{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ParentID:       parentID,
		Name:           name,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO categories (id, organization_id, parent_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, cat.ID, cat.OrganizationID, cat.ParentID, cat.Name)
	if err != nil {
		return nil, err
	}

	return cat, nil
}

// CreateLocation creates a test location
func (f *Fixtures) CreateLocation(ctx context.Context, orgID uuid.UUID, name string, parentID *uuid.UUID) (*domain.Location, error) {
	loc := &domain.Location{
		ID:             uuid.New(),
		OrganizationID: orgID,
		ParentID:       parentID,
		Name:           name,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO locations (id, organization_id, parent_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, loc.ID, loc.OrganizationID, loc.ParentID, loc.Name)
	if err != nil {
		return nil, err
	}

	return loc, nil
}

// CreateCondition creates a test condition
func (f *Fixtures) CreateCondition(ctx context.Context, orgID uuid.UUID, code, label string, sortOrder int) (*domain.Condition, error) {
	cond := &domain.Condition{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Code:           code,
		Label:          label,
		SortOrder:      sortOrder,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO conditions (id, organization_id, code, label, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, cond.ID, cond.OrganizationID, cond.Code, cond.Label, cond.SortOrder)
	if err != nil {
		return nil, err
	}

	return cond, nil
}

// CreateAsset creates a test asset
func (f *Fixtures) CreateAsset(ctx context.Context, orgID, categoryID uuid.UUID, name string) (*domain.Asset, error) {
	asset := &domain.Asset{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CategoryID:     categoryID,
		Name:           name,
		Quantity:       1,
		Attributes:     []byte("{}"),
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO assets (id, organization_id, category_id, name, quantity, attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, asset.ID, asset.OrganizationID, asset.CategoryID, asset.Name, asset.Quantity, asset.Attributes)
	if err != nil {
		return nil, err
	}

	return asset, nil
}

// CreateAssetFull creates a test asset with all optional fields
func (f *Fixtures) CreateAssetFull(ctx context.Context, asset *domain.Asset) error {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	if asset.Attributes == nil {
		asset.Attributes = []byte("{}")
	}
	if asset.Quantity == 0 {
		asset.Quantity = 1
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO assets (id, organization_id, category_id, location_id, condition_id, collection_id,
		                    name, description, quantity, attributes, purchase_at, purchase_price, purchase_note,
		                    import_plugin_id, import_external_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
	`, asset.ID, asset.OrganizationID, asset.CategoryID, asset.LocationID, asset.ConditionID, asset.CollectionID,
		asset.Name, asset.Description, asset.Quantity, asset.Attributes, asset.PurchaseAt, asset.PurchasePrice, asset.PurchaseNote,
		asset.ImportPluginID, asset.ImportExternalID)
	return err
}

// CreateWarranty creates a test warranty
func (f *Fixtures) CreateWarranty(ctx context.Context, assetID uuid.UUID, provider string, endDate time.Time) (*domain.Warranty, error) {
	warranty := &domain.Warranty{
		ID:        uuid.New(),
		AssetID:   assetID,
		Provider:  &provider,
		EndDate:   &endDate,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO warranties (id, asset_id, provider, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, warranty.ID, warranty.AssetID, warranty.Provider, warranty.EndDate, warranty.CreatedAt, warranty.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return warranty, nil
}

// CreateAttachment creates a test attachment
func (f *Fixtures) CreateAttachment(ctx context.Context, assetID uuid.UUID, fileName, fileKey string) (*domain.Attachment, error) {
	attachment := &domain.Attachment{
		ID:        uuid.New(),
		AssetID:   assetID,
		FileKey:   fileKey,
		FileName:  fileName,
		FileSize:  1024,
		CreatedAt: time.Now().UTC(),
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO attachments (id, asset_id, file_key, file_name, file_size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, attachment.ID, attachment.AssetID, attachment.FileKey, attachment.FileName, attachment.FileSize, attachment.CreatedAt)
	if err != nil {
		return nil, err
	}

	return attachment, nil
}

// CreateAttribute creates a test attribute
func (f *Fixtures) CreateAttribute(ctx context.Context, orgID uuid.UUID, name, key string, dataType domain.AttributeDataType) (*domain.Attribute, error) {
	attr := &domain.Attribute{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           name,
		Key:            key,
		DataType:       dataType,
	}

	_, err := f.pool.Exec(ctx, `
		INSERT INTO attributes (id, organization_id, name, key, data_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, attr.ID, attr.OrganizationID, attr.Name, attr.Key, attr.DataType)
	if err != nil {
		return nil, err
	}

	return attr, nil
}

// CreateTag creates a test tag
func (f *Fixtures) CreateTag(ctx context.Context, orgID uuid.UUID, name string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := f.pool.Exec(ctx, `
		INSERT INTO tags (id, organization_id, name, created_at)
		VALUES ($1, $2, $3, NOW())
	`, id, orgID, name)
	return id, err
}

// AddTagToAsset adds a tag to an asset
func (f *Fixtures) AddTagToAsset(ctx context.Context, assetID, tagID uuid.UUID) error {
	_, err := f.pool.Exec(ctx, `
		INSERT INTO asset_tags (asset_id, tag_id)
		VALUES ($1, $2)
	`, assetID, tagID)
	return err
}
