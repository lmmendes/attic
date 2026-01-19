package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// LocalStorage implements file storage on the local filesystem
type LocalStorage struct {
	basePath string
	baseURL  string
	puid     *int
	pgid     *int
}

// LocalConfig holds local storage configuration
type LocalConfig struct {
	// BasePath is the directory where files will be stored
	BasePath string
	// BaseURL is the base URL for serving files (e.g., "http://localhost:8080/files")
	BaseURL string
	// PUID is the user ID for file ownership (nil = don't change ownership)
	PUID *int
	// PGID is the group ID for file ownership (nil = don't change ownership)
	PGID *int
}

// NewLocalStorage creates a new local file storage client
func NewLocalStorage(cfg LocalConfig) (*LocalStorage, error) {
	// Ensure the base path exists
	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	s := &LocalStorage{
		basePath: cfg.BasePath,
		baseURL:  cfg.BaseURL,
		puid:     cfg.PUID,
		pgid:     cfg.PGID,
	}

	// Chown the base directory if PUID/PGID are configured
	if err := s.chown(cfg.BasePath); err != nil {
		return nil, fmt.Errorf("setting ownership of storage directory: %w", err)
	}

	return s, nil
}

// Upload saves a file to local storage and returns the storage key
func (s *LocalStorage) Upload(ctx context.Context, filename string, contentType string, body io.Reader) (string, error) {
	// Create a unique key like S3 does: uuid/filename
	key := fmt.Sprintf("%s/%s", uuid.New().String(), filename)

	// Create the full path
	fullPath := filepath.Join(s.basePath, key)

	// Create parent directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	// Chown the directory if PUID/PGID are configured
	if err := s.chown(dir); err != nil {
		return "", fmt.Errorf("setting directory ownership: %w", err)
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	// Copy the content
	if _, err := io.Copy(file, body); err != nil {
		// Clean up on failure
		os.Remove(fullPath)
		return "", fmt.Errorf("writing file: %w", err)
	}

	// Chown the file if PUID/PGID are configured
	if err := s.chown(fullPath); err != nil {
		// Clean up on failure
		os.Remove(fullPath)
		return "", fmt.Errorf("setting file ownership: %w", err)
	}

	return key, nil
}

// GetPresignedURL returns a URL for accessing the file
// For local storage, this returns a direct URL path (no expiry is enforced)
func (s *LocalStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// Verify the file exists
	fullPath := filepath.Join(s.basePath, key)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", key)
	}

	// Return the URL to access this file
	return fmt.Sprintf("%s/%s", s.baseURL, key), nil
}

// Delete removes a file from local storage
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("deleting file: %w", err)
	}

	// Try to remove the parent directory if empty
	dir := filepath.Dir(fullPath)
	os.Remove(dir) // Ignore error - directory might not be empty

	return nil
}

// BasePath returns the base storage path (useful for serving files)
func (s *LocalStorage) BasePath() string {
	return s.basePath
}

// chown changes the ownership of a file or directory if PUID/PGID are configured
func (s *LocalStorage) chown(path string) error {
	if s.puid == nil || s.pgid == nil {
		return nil
	}
	return os.Chown(path, *s.puid, *s.pgid)
}
