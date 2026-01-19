package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func Test_LocalStorage_Upload_Success(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	content := "Hello, World!"
	body := strings.NewReader(content)

	key, err := storage.Upload(ctx, "test.txt", "text/plain", body)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}
	if !strings.HasSuffix(key, "/test.txt") {
		t.Errorf("expected key to end with '/test.txt', got '%s'", key)
	}

	// Verify file exists and has correct content
	fullPath := filepath.Join(tmpDir, key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}
}

func Test_LocalStorage_Upload_GeneratesUniqueKeys(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	key1, err := storage.Upload(ctx, "file.txt", "text/plain", strings.NewReader("content1"))
	if err != nil {
		t.Fatalf("failed to upload first file: %v", err)
	}

	key2, err := storage.Upload(ctx, "file.txt", "text/plain", strings.NewReader("content2"))
	if err != nil {
		t.Fatalf("failed to upload second file: %v", err)
	}

	if key1 == key2 {
		t.Error("expected unique keys for each upload")
	}
}

func Test_LocalStorage_GetPresignedURL_Success(t *testing.T) {
	tmpDir := t.TempDir()
	baseURL := "http://localhost:8080/files"
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  baseURL,
	})

	ctx := context.Background()

	// Upload a file first
	key, err := storage.Upload(ctx, "presign-test.txt", "text/plain", strings.NewReader("test content"))
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Get URL
	url, err := storage.GetPresignedURL(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to get URL: %v", err)
	}

	if url == "" {
		t.Error("expected non-empty URL")
	}

	expectedURL := baseURL + "/" + key
	if url != expectedURL {
		t.Errorf("expected URL '%s', got '%s'", expectedURL, url)
	}
}

func Test_LocalStorage_GetPresignedURL_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	_, err := storage.GetPresignedURL(ctx, "non-existent-key/file.txt", 15*time.Minute)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func Test_LocalStorage_Delete_Success(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	// Upload a file
	key, err := storage.Upload(ctx, "delete-test.txt", "text/plain", strings.NewReader("to be deleted"))
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify it exists
	fullPath := filepath.Join(tmpDir, key)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("file should exist before delete")
	}

	// Delete the file
	err = storage.Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify it no longer exists
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func Test_LocalStorage_Delete_NonExistentKey_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	// Delete a non-existent key - should not error
	err := storage.Delete(ctx, "non-existent-key/file.txt")
	if err != nil {
		t.Errorf("delete of non-existent key should not error: %v", err)
	}
}

func Test_NewLocalStorage_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "nested", "storage", "dir")

	storage, err := NewLocalStorage(LocalConfig{
		BasePath: storagePath,
		BaseURL:  "http://localhost:8080/files",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(storagePath)
	if err != nil {
		t.Fatalf("storage directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected storage path to be a directory")
	}

	if storage.BasePath() != storagePath {
		t.Errorf("expected BasePath '%s', got '%s'", storagePath, storage.BasePath())
	}
}

func Test_LocalStorage_Upload_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	// Create a 1MB file
	size := 1024 * 1024
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	key, err := storage.Upload(ctx, "large-file.bin", "application/octet-stream", strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("failed to upload large file: %v", err)
	}

	// Verify the file size
	fullPath := filepath.Join(tmpDir, key)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if info.Size() != int64(size) {
		t.Errorf("expected size %d, got %d", size, info.Size())
	}
}

func Test_LocalStorage_Upload_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	// Binary content with various bytes
	content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x89, 0x50, 0x4E, 0x47}

	key, err := storage.Upload(ctx, "binary.bin", "application/octet-stream", strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Read back and verify
	fullPath := filepath.Join(tmpDir, key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(data) != len(content) {
		t.Errorf("expected length %d, got %d", len(content), len(data))
	}
	for i := range content {
		if data[i] != content[i] {
			t.Errorf("byte mismatch at position %d: expected %x, got %x", i, content[i], data[i])
		}
	}
}

func Test_LocalStorage_ImplementsFileStorage(t *testing.T) {
	// Compile-time check that LocalStorage implements FileStorage
	var _ FileStorage = (*LocalStorage)(nil)
}

func Test_S3Client_ImplementsFileStorage(t *testing.T) {
	// Compile-time check that S3Client implements FileStorage
	var _ FileStorage = (*S3Client)(nil)
}

func Test_LocalStorage_Upload_ReaderError(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})

	ctx := context.Background()

	// Create a reader that fails after some bytes
	errReader := &errorReader{
		data: []byte("some data"),
		pos:  0,
	}

	_, err := storage.Upload(ctx, "error-test.txt", "text/plain", errReader)
	if err == nil {
		t.Error("expected error from failing reader")
	}
}

type errorReader struct {
	data []byte
	pos  int
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, io.ErrUnexpectedEOF
	}
	return n, nil
}

func Test_LocalStorage_WithPUIDPGID_StoresConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Use current user's UID/GID so chown succeeds without root
	puid := os.Getuid()
	pgid := os.Getgid()

	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
		PUID:     &puid,
		PGID:     &pgid,
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	if storage.puid == nil || *storage.puid != puid {
		t.Errorf("expected puid to be %d, got %v", puid, storage.puid)
	}
	if storage.pgid == nil || *storage.pgid != pgid {
		t.Errorf("expected pgid to be %d, got %v", pgid, storage.pgid)
	}
}

func Test_LocalStorage_WithoutPUIDPGID_NilValues(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	if storage.puid != nil {
		t.Errorf("expected puid to be nil, got %v", storage.puid)
	}
	if storage.pgid != nil {
		t.Errorf("expected pgid to be nil, got %v", storage.pgid)
	}
}

func Test_LocalStorage_chown_NilPUIDPGID_NoError(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// chown should return nil when PUID/PGID are not set
	err = storage.chown(tmpDir)
	if err != nil {
		t.Errorf("expected no error when PUID/PGID are nil, got: %v", err)
	}
}

func Test_LocalStorage_chown_OnlyPUID_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	puid := 1000

	storage := &LocalStorage{
		basePath: tmpDir,
		baseURL:  "http://localhost:8080/files",
		puid:     &puid,
		pgid:     nil, // Only PUID set, not PGID
	}

	// chown should return nil when only one of PUID/PGID is set
	err := storage.chown(tmpDir)
	if err != nil {
		t.Errorf("expected no error when only PUID is set, got: %v", err)
	}
}

func Test_LocalStorage_Upload_WithoutPUIDPGID_Success(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	content := "Hello, World!"
	body := strings.NewReader(content)

	key, err := storage.Upload(ctx, "test.txt", "text/plain", body)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}

	// Verify file exists and has correct content
	fullPath := filepath.Join(tmpDir, key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}
}

func Test_LocalStorage_Upload_WithPUIDPGID_Success(t *testing.T) {
	tmpDir := t.TempDir()
	// Use current user's UID/GID so chown succeeds without root
	puid := os.Getuid()
	pgid := os.Getgid()

	storage, err := NewLocalStorage(LocalConfig{
		BasePath: tmpDir,
		BaseURL:  "http://localhost:8080/files",
		PUID:     &puid,
		PGID:     &pgid,
	})
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	content := "Hello, World with chown!"
	body := strings.NewReader(content)

	key, err := storage.Upload(ctx, "test-chown.txt", "text/plain", body)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}

	// Verify file exists and has correct content
	fullPath := filepath.Join(tmpDir, key)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}
}
