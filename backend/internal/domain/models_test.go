package domain

import (
	"testing"

	"github.com/google/uuid"
)

func Test_User_IsAdmin_AdminRole_ReturnsTrue(t *testing.T) {
	user := &User{
		ID:   uuid.New(),
		Role: UserRoleAdmin,
	}

	if !user.IsAdmin() {
		t.Error("expected IsAdmin to return true for admin role")
	}
}

func Test_User_IsAdmin_UserRole_ReturnsFalse(t *testing.T) {
	user := &User{
		ID:   uuid.New(),
		Role: UserRoleUser,
	}

	if user.IsAdmin() {
		t.Error("expected IsAdmin to return false for user role")
	}
}

func Test_User_HasPassword_WithPassword_ReturnsTrue(t *testing.T) {
	hash := "$2a$10$somehashedpassword"
	user := &User{
		ID:           uuid.New(),
		PasswordHash: &hash,
	}

	if !user.HasPassword() {
		t.Error("expected HasPassword to return true when password hash is set")
	}
}

func Test_User_HasPassword_NilPassword_ReturnsFalse(t *testing.T) {
	user := &User{
		ID:           uuid.New(),
		PasswordHash: nil,
	}

	if user.HasPassword() {
		t.Error("expected HasPassword to return false when password hash is nil")
	}
}

func Test_User_HasPassword_EmptyPassword_ReturnsFalse(t *testing.T) {
	emptyHash := ""
	user := &User{
		ID:           uuid.New(),
		PasswordHash: &emptyHash,
	}

	if user.HasPassword() {
		t.Error("expected HasPassword to return false when password hash is empty string")
	}
}

func Test_UserRole_Constants_HaveExpectedValues(t *testing.T) {
	if UserRoleUser != "user" {
		t.Errorf("expected UserRoleUser to be 'user', got '%s'", UserRoleUser)
	}
	if UserRoleAdmin != "admin" {
		t.Errorf("expected UserRoleAdmin to be 'admin', got '%s'", UserRoleAdmin)
	}
}

func Test_AttributeDataType_Constants_HaveExpectedValues(t *testing.T) {
	tests := []struct {
		constant AttributeDataType
		expected string
	}{
		{AttributeTypeString, "string"},
		{AttributeTypeNumber, "number"},
		{AttributeTypeBoolean, "boolean"},
		{AttributeTypeText, "text"},
		{AttributeTypeDate, "date"},
	}

	for _, tt := range tests {
		if string(tt.constant) != tt.expected {
			t.Errorf("expected %s, got '%s'", tt.expected, tt.constant)
		}
	}
}
