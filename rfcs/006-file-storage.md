# RFC-006: File Storage and Attachments

| Field       | Value                          |
|-------------|--------------------------------|
| Status      | Implemented                    |
| Created     | 2026-01-17                     |
| Author      | @lmmendes                      |

## Summary

This RFC describes the file storage and attachment system in Attic, which supports two storage backends: Amazon S3 (or S3-compatible services) and local filesystem storage. The system allows users to upload files as attachments to assets, with automatic content-type detection, size limits, and secure download URLs.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Storage Backends](#storage-backends)
4. [Configuration](#configuration)
5. [API Endpoints](#api-endpoints)
6. [Database Schema](#database-schema)
7. [Upload Flow](#upload-flow)
8. [Download Flow](#download-flow)
9. [Delete Flow](#delete-flow)
10. [Security Considerations](#security-considerations)
11. [Error Handling](#error-handling)

## Overview

Attic uses a two-tier storage architecture:
- **File Storage**: Binary file data stored in S3 or local filesystem
- **Metadata Storage**: Attachment records stored in PostgreSQL

This separation allows for:
- Efficient binary file storage optimized for each backend
- Fast metadata queries via the database
- Flexible storage backend switching without data migration complexity

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Client                                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           HTTP Handler                                   │
│                    (internal/handler/attachment.go)                      │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │ Upload      │  │ List         │  │ Get         │  │ Delete       │  │
│  │ Attachment  │  │ Attachments  │  │ Attachment  │  │ Attachment   │  │
│  └─────────────┘  └──────────────┘  └─────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                    │                                      │
                    ▼                                      ▼
┌─────────────────────────────────┐    ┌─────────────────────────────────┐
│         FileStorage             │    │      AttachmentRepository       │
│    (storage.FileStorage)        │    │  (repository/attachment.go)     │
│                                 │    │                                 │
│  ┌───────────┐ ┌─────────────┐  │    │  ┌─────────┐ ┌──────────────┐  │
│  │ S3Client  │ │LocalStorage │  │    │  │ Create  │ │ GetByID      │  │
│  └───────────┘ └─────────────┘  │    │  │ Delete  │ │ ListByAsset  │  │
│        │              │         │    │  └─────────┘ └──────────────┘  │
└────────┼──────────────┼─────────┘    └─────────────────────────────────┘
         │              │                              │
         ▼              ▼                              ▼
┌─────────────┐  ┌─────────────┐              ┌─────────────┐
│  Amazon S3  │  │   Local     │              │ PostgreSQL  │
│  / MinIO    │  │ Filesystem  │              │  Database   │
└─────────────┘  └─────────────┘              └─────────────┘
```

## Storage Backends

### FileStorage Interface

Both storage backends implement the `FileStorage` interface defined in `internal/storage/storage.go`:

```go
type FileStorage interface {
    // Upload uploads a file and returns the storage key
    Upload(ctx context.Context, filename string, contentType string, body io.Reader) (string, error)

    // GetPresignedURL returns a URL for downloading the file
    GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

    // Delete removes a file from storage
    Delete(ctx context.Context, key string) error
}
```

### S3 Storage (`internal/storage/s3.go`)

**When to use:** Production environments, multi-server deployments, or when you need durable cloud storage.

**Features:**
- Supports AWS S3 and S3-compatible services (MinIO, LocalStack, etc.)
- Uses path-style URLs for compatibility with LocalStack
- Generates time-limited presigned URLs for secure downloads
- Stores content-type metadata with each object

**File Key Format:**
```
{uuid}/{original_filename}
```
Example: `550e8400-e29b-41d4-a716-446655440000/document.pdf`

**Presigned URLs:**
- Generated on-demand when retrieving attachments
- Default expiry: 15 minutes
- Contains cryptographic signature for authorization
- Example: `https://bucket.s3.amazonaws.com/key?X-Amz-Signature=...&X-Amz-Expires=900`

### Local Storage (`internal/storage/local.go`)

**When to use:** Development, single-server deployments, or environments without S3 access.

**Features:**
- Stores files in a configurable directory on the local filesystem
- Creates nested directories automatically
- Serves files via HTTP endpoint `/files/*`
- No URL expiry (files accessible as long as they exist)

**Directory Structure:**
```
{ATTIC_LOCAL_STORAGE_PATH}/
└── {uuid}/
    └── {original_filename}
```
Example: `./uploads/550e8400-e29b-41d4-a716-446655440000/document.pdf`

**Download URLs:**
- Format: `{ATTIC_BASE_URL}/files/{uuid}/{filename}`
- Example: `http://localhost:8080/files/550e8400-e29b-41d4-a716-446655440000/document.pdf`

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ATTIC_S3_ACCESS_KEY` | *(empty)* | S3 access key. If empty, local storage is used. |
| `ATTIC_S3_SECRET_KEY` | *(empty)* | S3 secret key. If empty, local storage is used. |
| `ATTIC_S3_ENDPOINT` | `http://localhost:4566` | S3 endpoint URL (for S3-compatible services) |
| `ATTIC_S3_BUCKET` | `attic-attachments` | S3 bucket name |
| `ATTIC_S3_REGION` | `us-east-1` | AWS region |
| `ATTIC_LOCAL_STORAGE_PATH` | `./uploads` | Directory for local file storage |
| `ATTIC_BASE_URL` | `http://localhost:8080` | Base URL for the application (used for local storage URLs) |

### Storage Backend Selection

The storage backend is selected automatically at startup based on configuration:

```go
func (c *Config) UseS3Storage() bool {
    return c.S3AccessKey != "" && c.S3SecretKey != ""
}
```

**S3 Mode:** Set both `ATTIC_S3_ACCESS_KEY` and `ATTIC_S3_SECRET_KEY`
**Local Mode:** Leave S3 credentials empty (default)

### Example Configurations

**Local Development (Local Storage):**
```bash
# No S3 credentials = local storage
export ATTIC_LOCAL_STORAGE_PATH="./uploads"
export ATTIC_BASE_URL="http://localhost:8080"
```

**Production with AWS S3:**
```bash
export ATTIC_S3_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
export ATTIC_S3_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export ATTIC_S3_BUCKET="my-attic-attachments"
export ATTIC_S3_REGION="us-west-2"
```

**Development with LocalStack:**
```bash
export ATTIC_S3_ACCESS_KEY="test"
export ATTIC_S3_SECRET_KEY="test"
export ATTIC_S3_ENDPOINT="http://localhost:4566"
export ATTIC_S3_BUCKET="attic-attachments"
```

**Production with MinIO:**
```bash
export ATTIC_S3_ACCESS_KEY="minioadmin"
export ATTIC_S3_SECRET_KEY="minioadmin"
export ATTIC_S3_ENDPOINT="http://minio:9000"
export ATTIC_S3_BUCKET="attic-attachments"
```

## API Endpoints

### Upload Attachment

```
POST /api/assets/{assetId}/attachments
Content-Type: multipart/form-data
Authorization: Bearer {token}
```

**Form Fields:**
- `file` (required): The file to upload
- `description` (optional): Text description of the attachment

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "asset_id": "660e8400-e29b-41d4-a716-446655440000",
  "file_key": "770e8400-e29b-41d4-a716-446655440000/document.pdf",
  "file_name": "document.pdf",
  "file_size": 1048576,
  "content_type": "application/pdf",
  "description": "Product manual",
  "created_at": "2026-01-17T10:30:00Z"
}
```

**Constraints:**
- Maximum file size: 50 MB
- Asset must exist

### List Attachments

```
GET /api/assets/{assetId}/attachments
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "asset_id": "660e8400-e29b-41d4-a716-446655440000",
    "file_key": "770e8400-e29b-41d4-a716-446655440000/document.pdf",
    "file_name": "document.pdf",
    "file_size": 1048576,
    "content_type": "application/pdf",
    "description": "Product manual",
    "created_at": "2026-01-17T10:30:00Z"
  }
]
```

### Get Attachment (with Download URL)

```
GET /api/attachments/{attachmentId}
Authorization: Bearer {token}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "asset_id": "660e8400-e29b-41d4-a716-446655440000",
  "file_key": "770e8400-e29b-41d4-a716-446655440000/document.pdf",
  "file_name": "document.pdf",
  "file_size": 1048576,
  "content_type": "application/pdf",
  "description": "Product manual",
  "created_at": "2026-01-17T10:30:00Z",
  "url": "https://bucket.s3.amazonaws.com/770e8400.../document.pdf?X-Amz-Signature=..."
}
```

The `url` field contains:
- **S3 Mode:** A presigned URL valid for 15 minutes
- **Local Mode:** A direct URL to the file server endpoint

### Delete Attachment

```
DELETE /api/attachments/{attachmentId}
Authorization: Bearer {token}
```

**Response:** 204 No Content

## Database Schema

```sql
CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    uploaded_by UUID REFERENCES users(id),
    file_key VARCHAR(500) NOT NULL,  -- Storage key (S3 object key or local path)
    file_name VARCHAR(255) NOT NULL, -- Original filename
    file_size BIGINT NOT NULL,       -- Size in bytes
    content_type VARCHAR(100),       -- MIME type
    description TEXT,                -- Optional description
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attachments_asset ON attachments(asset_id);
```

**Key Fields:**
- `file_key`: The storage-specific identifier. Same format for both backends: `{uuid}/{filename}`
- `file_name`: Original filename as uploaded by user (for display purposes)
- `file_size`: Used for display and potential quota enforcement
- `content_type`: MIME type for proper content serving

**Cascade Delete:** When an asset is deleted, all its attachments are automatically deleted from the database. The storage cleanup happens in the delete handler.

## Upload Flow

```
┌────────┐     ┌─────────┐     ┌─────────────┐     ┌─────────┐     ┌────────┐
│ Client │     │ Handler │     │ FileStorage │     │  Repo   │     │   DB   │
└───┬────┘     └────┬────┘     └──────┬──────┘     └────┬────┘     └───┬────┘
    │               │                 │                 │              │
    │ POST /api/assets/{id}/attachments                 │              │
    │──────────────>│                 │                 │              │
    │               │                 │                 │              │
    │               │ Validate asset exists             │              │
    │               │─────────────────────────────────────────────────>│
    │               │<─────────────────────────────────────────────────│
    │               │                 │                 │              │
    │               │ Parse multipart │                 │              │
    │               │ Detect content-type               │              │
    │               │                 │                 │              │
    │               │ Upload(filename, contentType, body)              │
    │               │────────────────>│                 │              │
    │               │                 │                 │              │
    │               │                 │ Store file      │              │
    │               │                 │ Generate key    │              │
    │               │                 │                 │              │
    │               │<────────────────│                 │              │
    │               │      key        │                 │              │
    │               │                 │                 │              │
    │               │ Create attachment record          │              │
    │               │────────────────────────────────-->│              │
    │               │                 │                 │──────────────>│
    │               │                 │                 │<──────────────│
    │               │<─────────────────────────────────│              │
    │               │                 │                 │              │
    │<──────────────│                 │                 │              │
    │  201 Created  │                 │                 │              │
    │  {attachment} │                 │                 │              │
```

**Steps:**
1. Client sends multipart form with file
2. Handler validates asset exists
3. Handler enforces 50MB size limit
4. Handler detects content-type (from header or file magic bytes)
5. Handler calls `storage.Upload()` with file data
6. Storage generates unique key (`{uuid}/{filename}`) and stores file
7. Handler creates attachment record in database with returned key
8. If database insert fails, handler deletes the uploaded file (cleanup)
9. Handler returns attachment metadata to client

## Download Flow

```
┌────────┐     ┌─────────┐     ┌──────┐     ┌─────────────┐     ┌─────────┐
│ Client │     │ Handler │     │ Repo │     │ FileStorage │     │ Storage │
└───┬────┘     └────┬────┘     └──┬───┘     └──────┬──────┘     └────┬────┘
    │               │             │                │                 │
    │ GET /api/attachments/{id}   │                │                 │
    │──────────────>│             │                │                 │
    │               │             │                │                 │
    │               │ GetByID(id) │                │                 │
    │               │────────────>│                │                 │
    │               │<────────────│                │                 │
    │               │ attachment  │                │                 │
    │               │             │                │                 │
    │               │ GetPresignedURL(key, 15min)  │                 │
    │               │─────────────────────────────>│                 │
    │               │                              │                 │
    │               │<─────────────────────────────│                 │
    │               │            url               │                 │
    │               │             │                │                 │
    │<──────────────│             │                │                 │
    │ 200 OK        │             │                │                 │
    │ {attachment + url}          │                │                 │
    │               │             │                │                 │
    │ GET {url}     │             │                │                 │
    │───────────────────────────────────────────────────────────────>│
    │<───────────────────────────────────────────────────────────────│
    │ File contents │             │                │                 │
```

**Steps:**
1. Client requests attachment metadata
2. Handler fetches attachment record from database
3. Handler generates download URL via `storage.GetPresignedURL()`
4. Handler returns attachment with URL
5. Client downloads file directly from storage URL

**Note:** The client downloads directly from storage (S3 or file server), not through the application. This offloads bandwidth from the application server.

## Delete Flow

```
┌────────┐     ┌─────────┐     ┌──────┐     ┌─────────────┐
│ Client │     │ Handler │     │ Repo │     │ FileStorage │
└───┬────┘     └────┬────┘     └──┬───┘     └──────┬──────┘
    │               │             │                │
    │ DELETE /api/attachments/{id}│                │
    │──────────────>│             │                │
    │               │             │                │
    │               │ GetByID(id) │                │
    │               │────────────>│                │
    │               │<────────────│                │
    │               │ attachment  │                │
    │               │             │                │
    │               │ Delete(key) │                │
    │               │─────────────────────────────>│
    │               │<─────────────────────────────│
    │               │             │                │
    │               │ Delete(id)  │                │
    │               │────────────>│                │
    │               │<────────────│                │
    │               │             │                │
    │<──────────────│             │                │
    │ 204 No Content│             │                │
```

**Steps:**
1. Client requests deletion
2. Handler fetches attachment to get file key
3. Handler deletes file from storage (errors logged but don't fail request)
4. Handler deletes attachment record from database
5. Handler returns 204 No Content

**Note:** Storage deletion errors are logged but don't fail the request. This ensures the database record is cleaned up even if storage is temporarily unavailable.

## Security Considerations

### Authentication
- All attachment endpoints require authentication via Bearer token
- Tokens validated by auth middleware before reaching handlers

### Authorization
- Currently, any authenticated user can access any attachment
- Future: Consider implementing asset-level permissions

### File Validation
- 50MB size limit enforced at handler level
- Content-type detected from file content (magic bytes), not just headers
- Original filename preserved but used only for display

### S3 Security
- Presigned URLs expire after 15 minutes
- URLs contain cryptographic signatures
- Bucket should have public access blocked; only presigned URLs work

### Local Storage Security
- Files served via dedicated `/files/*` endpoint
- No directory listing (files only accessible by exact key)
- Consider placing behind authentication if needed

### Path Traversal Prevention
- File keys use UUIDs, preventing path traversal attacks
- Format: `{uuid}/{filename}` where UUID is server-generated
- Local storage joins paths safely with `filepath.Join()`

## Error Handling

| Scenario | HTTP Status | Response |
|----------|-------------|----------|
| Asset not found | 404 | `{"error": "asset not found"}` |
| Attachment not found | 404 | `{"error": "attachment not found"}` |
| File too large | 400 | `{"error": "file too large or invalid form"}` |
| Missing file in request | 400 | `{"error": "missing file in request"}` |
| Storage not configured | 503 | `{"error": "storage not configured"}` |
| Upload failed | 500 | `{"error": "failed to upload file: {details}"}` |
| Database error | 500 | `{"error": "failed to save attachment record"}` |

### Graceful Degradation

If storage initialization fails at startup:
- Application continues running
- Upload/download endpoints return 503 Service Unavailable
- Logged as warning, not fatal error

## Future Considerations

1. **Virus Scanning**: Integrate with ClamAV or similar for uploaded files
2. **Image Processing**: Generate thumbnails for image attachments
3. **Quota Enforcement**: Limit total storage per organization
4. **Deduplication**: Hash-based deduplication for identical files
5. **Async Processing**: Queue large uploads for background processing
6. **CDN Integration**: Serve files via CloudFront or similar CDN
7. **Encryption at Rest**: Client-side encryption for sensitive files
