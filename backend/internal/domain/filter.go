package domain

import (
	"time"

	"github.com/google/uuid"
)

// FilterCriteria is the versioned, reusable definition of an asset search.
type FilterCriteria struct {
	Version        int         `json:"version"`
	Query          string      `json:"q,omitempty"`
	AttributeQuery string      `json:"attribute_q,omitempty"`
	CategoryID     string      `json:"category_id,omitempty"`
	LocationID     string      `json:"location_id,omitempty"`
	ConditionID    string      `json:"condition_id,omitempty"`
	CollectionID   string      `json:"collection_id,omitempty"`
	Expression     *FilterNode `json:"expression,omitempty"`
}

// FilterNode is either a group or a typed rule. References use stable IDs.
type FilterNode struct {
	Kind          string            `json:"kind"`
	Match         string            `json:"match,omitempty"`
	Children      []FilterNode      `json:"children,omitempty"`
	Field         string            `json:"field,omitempty"`
	AttributeID   string            `json:"attribute_id,omitempty"`
	DataType      AttributeDataType `json:"data_type,omitempty"`
	SelectionMode string            `json:"selection_mode,omitempty"`
	Operator      string            `json:"operator,omitempty"`
	Value         any               `json:"value,omitempty"`
	Values        []string          `json:"values,omitempty"`
	Upper         any               `json:"upper,omitempty"`
}

type FilterIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type SavedFilter struct {
	ID             uuid.UUID      `json:"id"`
	OrganizationID uuid.UUID      `json:"-"`
	UserID         uuid.UUID      `json:"-"`
	Name           string         `json:"name"`
	Criteria       FilterCriteria `json:"criteria"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Issues         []FilterIssue  `json:"issues,omitempty"`
}
