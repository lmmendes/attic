package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents a single organization (tenant)
type Organization struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"-"`
}

// User represents an OIDC-authenticated user
type User struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	OIDCSubject    string     `json:"oidc_subject"`
	Email          string     `json:"email"`
	DisplayName    *string    `json:"display_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}

// Condition represents an asset condition (e.g., NEW, GOOD, FAIR)
type Condition struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Code           string     `json:"code"`
	Label          string     `json:"label"`
	Description    *string    `json:"description,omitempty"`
	SortOrder      int        `json:"sort_order"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}

// Category represents an asset category (hierarchical)
type Category struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`

	// Populated by queries
	Children   []Category          `json:"children,omitempty"`
	Attributes []CategoryAttribute `json:"attributes,omitempty"`
}

// AttributeDataType represents the data type of an attribute
type AttributeDataType string

const (
	AttributeTypeString  AttributeDataType = "string"
	AttributeTypeNumber  AttributeDataType = "number"
	AttributeTypeBoolean AttributeDataType = "boolean"
	AttributeTypeText    AttributeDataType = "text"
	AttributeTypeDate    AttributeDataType = "date"
)

// Attribute represents a reusable attribute definition (organization-level)
type Attribute struct {
	ID             uuid.UUID         `json:"id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	Name           string            `json:"name"`
	Key            string            `json:"key"`
	DataType       AttributeDataType `json:"data_type"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DeletedAt      *time.Time        `json:"-"`
}

// CategoryAttribute represents the relationship between a category and an attribute
type CategoryAttribute struct {
	ID          uuid.UUID  `json:"id"`
	CategoryID  uuid.UUID  `json:"category_id"`
	AttributeID uuid.UUID  `json:"attribute_id"`
	Required    bool       `json:"required"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`

	// Populated by queries
	Attribute *Attribute `json:"attribute,omitempty"`
}

// Location represents a physical location (hierarchical)
type Location struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`

	// Populated by queries
	Children []Location `json:"children,omitempty"`
}

// Asset represents a tracked item
type Asset struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	CategoryID     uuid.UUID       `json:"category_id"`
	LocationID     *uuid.UUID      `json:"location_id,omitempty"`
	ConditionID    *uuid.UUID      `json:"condition_id,omitempty"`
	CollectionID   *uuid.UUID      `json:"collection_id,omitempty"`
	Name           string          `json:"name"`
	Description    *string         `json:"description,omitempty"`
	Quantity       int             `json:"quantity"`
	Attributes     json.RawMessage `json:"attributes"`
	PurchaseAt     *time.Time      `json:"purchase_at,omitempty"`
	PurchaseNote   *string         `json:"purchase_note,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      *time.Time      `json:"-"`

	// Populated by queries
	Category   *Category   `json:"category,omitempty"`
	Location   *Location   `json:"location,omitempty"`
	Condition  *Condition  `json:"condition,omitempty"`
	Tags       []Tag       `json:"tags,omitempty"`
	Warranty   *Warranty   `json:"warranty,omitempty"`
}

// Tag represents a free-form tag
type Tag struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
}

// Warranty represents warranty information for an asset
type Warranty struct {
	ID        uuid.UUID  `json:"id"`
	AssetID   uuid.UUID  `json:"asset_id"`
	Provider  *string    `json:"provider,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Attachment represents a file attached to an asset
type Attachment struct {
	ID          uuid.UUID  `json:"id"`
	AssetID     uuid.UUID  `json:"asset_id"`
	UploadedBy  *uuid.UUID `json:"uploaded_by,omitempty"`
	FileKey     string     `json:"file_key"`
	FileName    string     `json:"file_name"`
	FileSize    int64      `json:"file_size"`
	ContentType *string    `json:"content_type,omitempty"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
