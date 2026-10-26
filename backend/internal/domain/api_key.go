package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type APIKey struct {
	ID             uuid.UUID         `json:"id"`
	OwnerID        uuid.UUID         `json:"owner_id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	Description    string            `json:"description"`
	Permissions    map[string]string `json:"permissions"`
	SecretHash     []byte            `json:"-"`
	Prefix         string            `json:"prefix"`
	CreatedAt      time.Time         `json:"created_at"`
	ExpiresAt      *time.Time        `json:"expires_at"`
	RevokedAt      *time.Time        `json:"revoked_at"`
	LastUsedAt     *time.Time        `json:"last_used_at"`
}

type PermissionResource struct {
	Resource string   `json:"resource"`
	Label    string   `json:"label"`
	Levels   []string `json:"levels"`
}

func PermissionCatalog(role UserRole) []PermissionResource {
	resources := []PermissionResource{
		{Resource: "assets", Label: "Assets"}, {Resource: "locations", Label: "Locations"},
		{Resource: "collections", Label: "Collections"}, {Resource: "categories", Label: "Categories"},
		{Resource: "attributes", Label: "Attributes"}, {Resource: "tags", Label: "Tags"},
		{Resource: "conditions", Label: "Conditions"}, {Resource: "warranties", Label: "Warranties"},
		{Resource: "attachments", Label: "Attachments"}, {Resource: "asset_events", Label: "Asset events"},
		{Resource: "saved_filters", Label: "Saved filters"}, {Resource: "plugins", Label: "Import plugins"},
		{Resource: "users", Label: "Users"}, {Resource: "organization", Label: "Organization settings"},
	}
	for i := range resources {
		resources[i].Levels = []string{"none", "read"}
		if role == UserRoleAdmin || (resources[i].Resource != "attributes" && resources[i].Resource != "users" && resources[i].Resource != "organization") {
			resources[i].Levels = append(resources[i].Levels, "write")
		}
		if role != UserRoleAdmin && resources[i].Resource == "users" {
			resources[i].Levels = []string{"none"}
		}
	}
	return resources
}

func RoleAllows(role UserRole, resource, action string) bool {
	for _, p := range PermissionCatalog(role) {
		if p.Resource == resource {
			for _, level := range p.Levels {
				if level == action || (level == "write" && action == "read") {
					return true
				}
			}
		}
	}
	return false
}

func (k *APIKey) Allows(user *User, resource, action string) bool {
	if user == nil || k.OwnerID != user.ID || k.OrganizationID != user.OrganizationID || !RoleAllows(user.Role, resource, action) {
		return false
	}
	level := k.Permissions[resource]
	return level == "write" || level == action
}

func ValidateKeyDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if n := utf8.RuneCountInString(description); n < 1 || n > 255 {
		return "", fmt.Errorf("description must contain 1–255 characters")
	}
	return description, nil
}

func ValidateKeyPermissions(permissions map[string]string, role UserRole) error {
	if len(permissions) == 0 {
		return fmt.Errorf("at least one permission is required")
	}
	for resource, level := range permissions {
		if (level != "read" && level != "write") || !RoleAllows(role, resource, level) {
			return fmt.Errorf("invalid permission for %s", resource)
		}
	}
	return nil
}

func NewAPIKeySecret() (string, []byte, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", nil, err
	}
	secret := "attic_" + base64.RawURLEncoding.EncodeToString(raw[:])
	return secret, HashAPIKey(secret), nil
}

func HashAPIKey(secret string) []byte { hash := sha256.Sum256([]byte(secret)); return hash[:] }
