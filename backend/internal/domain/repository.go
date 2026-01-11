package domain

import (
	"context"

	"github.com/google/uuid"
)

// OrganizationRepository handles organization persistence
type OrganizationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetDefault(ctx context.Context) (*Organization, error)
	Create(ctx context.Context, org *Organization) error
	Update(ctx context.Context, org *Organization) error
}

// UserRepository handles user persistence
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByOIDCSubject(ctx context.Context, subject string) (*User, error)
	List(ctx context.Context, orgID uuid.UUID) ([]User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}

// ConditionRepository handles condition persistence
type ConditionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Condition, error)
	List(ctx context.Context, orgID uuid.UUID) ([]Condition, error)
	Create(ctx context.Context, cond *Condition) error
	Update(ctx context.Context, cond *Condition) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CategoryRepository handles category persistence
type CategoryRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Category, error)
	GetByIDWithAttributes(ctx context.Context, id uuid.UUID) (*Category, error)
	List(ctx context.Context, orgID uuid.UUID) ([]Category, error)
	ListTree(ctx context.Context, orgID uuid.UUID) ([]Category, error)
	Create(ctx context.Context, cat *Category) error
	Update(ctx context.Context, cat *Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetAttributes(ctx context.Context, categoryID uuid.UUID, assignments []CategoryAttributeAssignment) error
}

// AttributeRepository handles attribute persistence
type AttributeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Attribute, error)
	List(ctx context.Context, orgID uuid.UUID) ([]Attribute, error)
	Create(ctx context.Context, attr *Attribute) error
	Update(ctx context.Context, attr *Attribute) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CategoryAttributeAssignment represents an attribute assignment to a category
type CategoryAttributeAssignment struct {
	AttributeID uuid.UUID
	Required    bool
	SortOrder   int
}

// LocationRepository handles location persistence
type LocationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Location, error)
	List(ctx context.Context, orgID uuid.UUID) ([]Location, error)
	ListTree(ctx context.Context, orgID uuid.UUID) ([]Location, error)
	Create(ctx context.Context, loc *Location) error
	Update(ctx context.Context, loc *Location) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AssetFilter defines filters for asset queries
type AssetFilter struct {
	CategoryID  *uuid.UUID
	LocationID  *uuid.UUID
	ConditionID *uuid.UUID
	TagIDs      []uuid.UUID
	Query       string // Full-text search query
	Attributes  map[string]any
}

// Pagination defines pagination parameters
type Pagination struct {
	Limit  int
	Offset int
}

// AssetRepository handles asset persistence
type AssetRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Asset, error)
	GetByIDFull(ctx context.Context, id uuid.UUID) (*Asset, error) // With relations
	List(ctx context.Context, orgID uuid.UUID, filter AssetFilter, page Pagination) ([]Asset, int, error)
	Search(ctx context.Context, orgID uuid.UUID, query string, page Pagination) ([]Asset, int, error)
	Create(ctx context.Context, asset *Asset) error
	Update(ctx context.Context, asset *Asset) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error
}

// TagRepository handles tag persistence
type TagRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Tag, error)
	GetByName(ctx context.Context, orgID uuid.UUID, name string) (*Tag, error)
	List(ctx context.Context, orgID uuid.UUID) ([]Tag, error)
	ListByAsset(ctx context.Context, assetID uuid.UUID) ([]Tag, error)
	Create(ctx context.Context, tag *Tag) error
	GetOrCreate(ctx context.Context, orgID uuid.UUID, name string) (*Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// WarrantyRepository handles warranty persistence
type WarrantyRepository interface {
	GetByAssetID(ctx context.Context, assetID uuid.UUID) (*Warranty, error)
	ListExpiring(ctx context.Context, orgID uuid.UUID, days int) ([]Warranty, error)
	Create(ctx context.Context, warranty *Warranty) error
	Update(ctx context.Context, warranty *Warranty) error
	Delete(ctx context.Context, assetID uuid.UUID) error
}

// AttachmentRepository handles attachment persistence
type AttachmentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Attachment, error)
	ListByAsset(ctx context.Context, assetID uuid.UUID) ([]Attachment, error)
	Create(ctx context.Context, attachment *Attachment) error
	Delete(ctx context.Context, id uuid.UUID) error
}
