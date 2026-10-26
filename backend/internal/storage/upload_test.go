package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUploadPrefixStorageLifecycle(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer server.Close()
	local, err := NewLocalStorage(LocalConfig{BasePath: root, BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	for name, store := range map[string]FileStorage{"local": local, "s3": testS3Client} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			keys := make(map[string]bool)
			for _, prefix := range []string{"locations", "locations", ""} {
				key, err := store.Upload(ctx, prefix, "location-photo.png", "image/png", strings.NewReader("photo content"))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { store.Delete(ctx, key) })
				if keys[key] {
					t.Fatalf("uploads reused key %q", key)
				}
				keys[key] = true
				parts := strings.Split(key, "/")
				if prefix == "" {
					if len(parts) != 2 {
						t.Fatalf("unprefixed upload changed layout: %q", key)
					}
				} else {
					if len(parts) != 3 || parts[0] != prefix {
						t.Fatalf("incorrect prefixed key: %q", key)
					}
					parts = parts[1:]
				}
				if _, err := uuid.Parse(parts[0]); err != nil || parts[1] != "location-photo.png" {
					t.Fatalf("incorrect upload UUID or filename: %q", key)
				}
				url, err := store.GetPresignedURL(ctx, key, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				response, err := http.Get(url)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil || response.StatusCode != http.StatusOK || string(body) != "photo content" {
					t.Fatalf("download failed: status=%d body=%q error=%v", response.StatusCode, body, err)
				}
				if err := store.Delete(ctx, key); err != nil {
					t.Fatal(err)
				}
				response, err = http.Get(url)
				if err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				if response.StatusCode != http.StatusNotFound {
					t.Fatalf("deleted file is still available: status=%d", response.StatusCode)
				}
			}
		})
	}
}

func TestUploadRejectsInvalidPrefix(t *testing.T) {
	local, err := NewLocalStorage(LocalConfig{BasePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for name, store := range map[string]FileStorage{"local": local, "s3": testS3Client} {
		t.Run(name, func(t *testing.T) {
			for _, prefix := range []string{".", "..", "/locations", "locations/", "../locations", "locations/sub", `locations\sub`, `C:\locations`, "locations\x00"} {
				key, err := store.Upload(context.Background(), prefix, "photo.png", "image/png", strings.NewReader("content"))
				if err == nil || key != "" {
					t.Fatalf("accepted invalid prefix %q: key=%q error=%v", prefix, key, err)
				}
			}
		})
	}
}
