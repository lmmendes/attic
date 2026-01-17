package storage

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testS3Client *S3Client
var testBucket = "test-bucket"

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start LocalStack container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "localstack/localstack:latest",
			ExposedPorts: []string{"4566/tcp"},
			Env: map[string]string{
				"SERVICES":       "s3",
				"DEFAULT_REGION": "us-east-1",
			},
			WaitingFor: wait.ForLog("Ready.").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		panic("failed to start localstack: " + err.Error())
	}
	defer container.Terminate(ctx)

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		panic("failed to get endpoint: " + err.Error())
	}

	// Create S3 client
	cfg := S3Config{
		Endpoint:  "http://" + endpoint,
		Region:    "us-east-1",
		Bucket:    testBucket,
		AccessKey: "test",
		SecretKey: "test",
	}

	testS3Client, err = NewS3Client(ctx, cfg)
	if err != nil {
		panic("failed to create S3 client: " + err.Error())
	}

	// Create test bucket
	_, err = testS3Client.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		panic("failed to create test bucket: " + err.Error())
	}

	m.Run()
}

func Test_S3Client_Upload_Success(t *testing.T) {
	ctx := context.Background()

	content := "Hello, World!"
	body := strings.NewReader(content)

	key, err := testS3Client.Upload(ctx, "test.txt", "text/plain", body)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	if key == "" {
		t.Error("expected non-empty key")
	}
	if !strings.HasSuffix(key, "/test.txt") {
		t.Errorf("expected key to end with '/test.txt', got '%s'", key)
	}

	// Verify file exists by getting it
	output, err := testS3Client.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("failed to get uploaded file: %v", err)
	}
	defer output.Body.Close()

	data, _ := io.ReadAll(output.Body)
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}
}

func Test_S3Client_Upload_WithContentType(t *testing.T) {
	ctx := context.Background()

	content := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	body := bytes.NewReader(content)

	key, err := testS3Client.Upload(ctx, "image.png", "image/png", body)
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify content type
	output, err := testS3Client.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("failed to head object: %v", err)
	}

	if output.ContentType == nil || *output.ContentType != "image/png" {
		t.Errorf("expected content type 'image/png', got '%v'", output.ContentType)
	}
}

func Test_S3Client_Upload_GeneratesUniqueKeys(t *testing.T) {
	ctx := context.Background()

	key1, err := testS3Client.Upload(ctx, "file.txt", "text/plain", strings.NewReader("content1"))
	if err != nil {
		t.Fatalf("failed to upload first file: %v", err)
	}

	key2, err := testS3Client.Upload(ctx, "file.txt", "text/plain", strings.NewReader("content2"))
	if err != nil {
		t.Fatalf("failed to upload second file: %v", err)
	}

	if key1 == key2 {
		t.Error("expected unique keys for each upload")
	}
}

func Test_S3Client_GetPresignedURL_Success(t *testing.T) {
	ctx := context.Background()

	// Upload a file first
	key, err := testS3Client.Upload(ctx, "presign-test.txt", "text/plain", strings.NewReader("test content"))
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Get presigned URL
	url, err := testS3Client.GetPresignedURL(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to get presigned URL: %v", err)
	}

	if url == "" {
		t.Error("expected non-empty URL")
	}
	if !strings.Contains(url, key) {
		t.Error("expected URL to contain the object key")
	}
	if !strings.Contains(url, "X-Amz-Signature") {
		t.Error("expected URL to contain signature")
	}
}

func Test_S3Client_GetPresignedURL_WithExpiry(t *testing.T) {
	ctx := context.Background()

	// Upload a file
	key, _ := testS3Client.Upload(ctx, "expiry-test.txt", "text/plain", strings.NewReader("test"))

	// Get presigned URLs with different expiries
	url1, err := testS3Client.GetPresignedURL(ctx, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to get presigned URL: %v", err)
	}

	url2, err := testS3Client.GetPresignedURL(ctx, key, 60*time.Minute)
	if err != nil {
		t.Fatalf("failed to get presigned URL: %v", err)
	}

	// Both should be valid URLs
	if url1 == "" || url2 == "" {
		t.Error("expected non-empty URLs")
	}

	// URLs should be different due to different expiry times
	if url1 == url2 {
		t.Error("expected different URLs for different expiry times")
	}
}

func Test_S3Client_Delete_Success(t *testing.T) {
	ctx := context.Background()

	// Upload a file
	key, err := testS3Client.Upload(ctx, "delete-test.txt", "text/plain", strings.NewReader("to be deleted"))
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify it exists
	_, err = testS3Client.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("file should exist before delete: %v", err)
	}

	// Delete the file
	err = testS3Client.Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify it no longer exists
	_, err = testS3Client.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(key),
	})
	if err == nil {
		t.Error("expected file to be deleted")
	}
}

func Test_S3Client_Delete_NonExistentKey_NoError(t *testing.T) {
	ctx := context.Background()

	// Delete a non-existent key - S3 doesn't error on this
	err := testS3Client.Delete(ctx, "non-existent-key-12345")
	if err != nil {
		t.Errorf("delete of non-existent key should not error: %v", err)
	}
}

func Test_NewS3Client_Success(t *testing.T) {
	// This test verifies the client was created successfully in TestMain
	if testS3Client == nil {
		t.Fatal("expected testS3Client to be initialized")
	}
	if testS3Client.bucket != testBucket {
		t.Errorf("expected bucket '%s', got '%s'", testBucket, testS3Client.bucket)
	}
	if testS3Client.client == nil {
		t.Error("expected S3 client to be initialized")
	}
}

func Test_S3Client_Upload_LargeFile(t *testing.T) {
	ctx := context.Background()

	// Create a 1MB file
	size := 1024 * 1024
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	key, err := testS3Client.Upload(ctx, "large-file.bin", "application/octet-stream", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to upload large file: %v", err)
	}

	// Verify the file size
	output, err := testS3Client.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("failed to head object: %v", err)
	}

	if output.ContentLength == nil || *output.ContentLength != int64(size) {
		t.Errorf("expected size %d, got %v", size, output.ContentLength)
	}
}
