package storage

import (
	"context"
	"io"
	"time"
)

// FileStorage defines the interface for file storage backends
type FileStorage interface {
	// Upload uploads a file and returns the storage key
	Upload(ctx context.Context, filename string, contentType string, body io.Reader) (string, error)

	// GetPresignedURL returns a URL for downloading the file
	// For S3, this is a presigned URL. For local storage, this is a direct path.
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, key string) error
}
