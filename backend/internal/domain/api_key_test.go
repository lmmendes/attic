package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAPIKeySecretAndPermissions(t *testing.T) {
	secret, hash, err := NewAPIKeySecret()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := NewAPIKeySecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) != 49 || !strings.HasPrefix(secret, "attic_") || len(hash) != 32 || !bytes.Equal(hash, HashAPIKey(secret)) || secret == second {
		t.Fatal("invalid secret generation")
	}
	key := APIKey{SecretHash: hash}
	encoded, _ := json.Marshal(key)
	if bytes.Contains(encoded, hash) || bytes.Contains(encoded, []byte("secret")) {
		t.Fatal("hash exposed in JSON")
	}
	for _, role := range []UserRole{UserRoleUser, UserRoleAdmin} {
		user := &User{ID: uuid.New(), OrganizationID: uuid.New(), Role: role}
		for _, resource := range PermissionCatalog(UserRoleAdmin) {
			for _, grant := range []string{"none", "read", "write"} {
				for _, action := range []string{"read", "write"} {
					key := APIKey{OwnerID: user.ID, OrganizationID: user.OrganizationID, Permissions: map[string]string{resource.Resource: grant}}
					want := (grant == "write" || grant == action) && RoleAllows(role, resource.Resource, action)
					if key.Allows(user, resource.Resource, action) != want {
						t.Errorf("role=%s resource=%s grant=%s action=%s", role, resource.Resource, grant, action)
					}
					other := *user
					other.OrganizationID = uuid.New()
					if key.Allows(&other, resource.Resource, action) {
						t.Fatal("cross-organization access")
					}
				}
			}
		}
	}
}
func TestAPIKeyValidation(t *testing.T) {
	for _, raw := range []string{"", "  ", strings.Repeat("x", 256)} {
		if _, err := ValidateKeyDescription(raw); err == nil {
			t.Fatalf("accepted description %q", raw)
		}
	}
	if description, err := ValidateKeyDescription("  Reader  "); err != nil || description != "Reader" {
		t.Fatal(description, err)
	}
	for _, permissions := range []map[string]string{nil, {}, {"assets": "none"}, {"fake": "read"}, {"attributes": "write"}, {"users": "read"}, {"organization": "write"}} {
		if err := ValidateKeyPermissions(permissions, UserRoleUser); err == nil {
			t.Fatal("accepted forbidden permissions", permissions)
		}
	}
	if err := ValidateKeyPermissions(map[string]string{"assets": "write", "organization": "read"}, UserRoleUser); err != nil {
		t.Fatal(err)
	}
}
