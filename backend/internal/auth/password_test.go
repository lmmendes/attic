package auth

import (
	"strings"
	"testing"
)

func Test_HashPassword_ReturnsValidHash(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPassword(password)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash should not equal original password")
	}
	// bcrypt hashes start with $2a$ or $2b$
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt hash prefix, got: %s", hash[:10])
	}
}

func Test_HashPassword_DifferentHashesForSamePassword(t *testing.T) {
	password := "mySecurePassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error for hash1, got: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error for hash2, got: %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("hashes should be different due to salt")
	}
}

func Test_CheckPassword_CorrectPassword_ReturnsTrue(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	result := CheckPassword(password, hash)

	if !result {
		t.Fatal("expected CheckPassword to return true for correct password")
	}
}

func Test_CheckPassword_WrongPassword_ReturnsFalse(t *testing.T) {
	password := "mySecurePassword123"
	wrongPassword := "wrongPassword456"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	result := CheckPassword(wrongPassword, hash)

	if result {
		t.Fatal("expected CheckPassword to return false for wrong password")
	}
}

func Test_CheckPassword_EmptyPassword_ReturnsFalse(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	result := CheckPassword("", hash)

	if result {
		t.Fatal("expected CheckPassword to return false for empty password")
	}
}

func Test_CheckPassword_InvalidHash_ReturnsFalse(t *testing.T) {
	result := CheckPassword("somePassword", "invalidhash")

	if result {
		t.Fatal("expected CheckPassword to return false for invalid hash")
	}
}

func Test_ValidatePassword_ValidLength_ReturnsNil(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		minLength int
	}{
		{
			name:      "exact minimum length",
			password:  "12345678",
			minLength: 8,
		},
		{
			name:      "above minimum length",
			password:  "1234567890",
			minLength: 8,
		},
		{
			name:      "minimum length of 1",
			password:  "a",
			minLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.minLength)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func Test_ValidatePassword_TooShort_ReturnsError(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		minLength int
	}{
		{
			name:      "one character short",
			password:  "1234567",
			minLength: 8,
		},
		{
			name:      "empty password",
			password:  "",
			minLength: 8,
		},
		{
			name:      "empty with minLength 1",
			password:  "",
			minLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.minLength)
			if err == nil {
				t.Error("expected error for password shorter than minimum length")
			}
		})
	}
}

func Test_ValidatePassword_ErrorMessage_ContainsMinLength(t *testing.T) {
	err := ValidatePassword("short", 10)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "10") {
		t.Errorf("error message should contain minimum length, got: %s", err.Error())
	}
}
