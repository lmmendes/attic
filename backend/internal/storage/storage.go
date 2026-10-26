package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage backends
type FileStorage interface {
	// Upload stores a file under an optional single-directory prefix and returns its full key.
	Upload(ctx context.Context, prefix string, filename string, contentType string, body io.Reader) (string, error)

	// GetPresignedURL returns a URL for downloading the file
	// For S3, this is a presigned URL. For local storage, this is a direct path.
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, key string) error
}

// newUploadKey keeps local paths and S3 object keys consistent. Prefixes are
// supplied by the server, separately from uploaded filenames.
func newUploadKey(prefix, filename string) (string, error) {
	if prefix == "." || prefix == ".." || strings.ContainsAny(prefix, "/\\\x00") {
		return "", fmt.Errorf("upload prefix must be a single directory name")
	}
	key := uuid.NewString() + "/" + filename
	if prefix != "" {
		key = prefix + "/" + key
	}
	return key, nil
}
