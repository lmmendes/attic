package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func cookieKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func encodeCookie(key []byte, name string, value any) (string, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshaling cookie: %w", err)
	}

	aead, err := newCookieAEAD(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating cookie nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, plaintext, []byte(name))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decodeCookie(key []byte, name, encoded string, value any) error {
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decoding cookie: %w", err)
	}
	aead, err := newCookieAEAD(key)
	if err != nil {
		return err
	}
	if len(sealed) < aead.NonceSize() {
		return fmt.Errorf("invalid cookie")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(name))
	if err != nil {
		return fmt.Errorf("authenticating cookie: %w", err)
	}
	if err := json.Unmarshal(plaintext, value); err != nil {
		return fmt.Errorf("unmarshaling cookie: %w", err)
	}
	return nil
}

func newCookieAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cookie cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating cookie AEAD: %w", err)
	}
	return aead, nil
}
