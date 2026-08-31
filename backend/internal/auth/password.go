package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10

var dummyPasswordHash = func() string {
	hash, _ := bcrypt.GenerateFromPassword([]byte("attic-dummy-password"), bcryptCost)
	return string(hash)
}()

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CheckPasswordHash performs a bcrypt comparison even when an account has no
// password hash, reducing account-enumeration timing differences at login.
func CheckPasswordHash(password string, hash *string) bool {
	candidate := dummyPasswordHash
	hasPassword := hash != nil && *hash != ""
	if hasPassword {
		candidate = *hash
	}
	return CheckPassword(password, candidate) && hasPassword
}

// ValidatePassword checks if password meets minimum requirements
func ValidatePassword(password string, minLength int) error {
	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
	}
	return nil
}
