package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func Test_AttachmentRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAttachmentRepository(testDB.Pool)
	contentType := "image/jpeg"
	attachment := &domain.Attachment{
		AssetID:     asset.ID,
		FileKey:     "attachments/12345/photo.jpg",
		FileName:    "photo.jpg",
		FileSize:    1024 * 1024,
		ContentType: &contentType,
	}

	err := repo.Create(ctx, attachment)
	if err != nil {
		t.Fatalf("failed to create attachment: %v", err)
	}

	if attachment.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if attachment.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_AttachmentRepository_Create_WithUploadedBy(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	user, _ := fixtures.CreateUser(ctx, org.ID, "user@example.com")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAttachmentRepository(testDB.Pool)
	attachment := &domain.Attachment{
		AssetID:    asset.ID,
		UploadedBy: &user.ID,
		FileKey:    "attachments/12345/doc.pdf",
		FileName:   "doc.pdf",
		FileSize:   2048,
	}

	err := repo.Create(ctx, attachment)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, attachment.ID)
	if fetched.UploadedBy == nil || *fetched.UploadedBy != user.ID {
		t.Error("expected uploaded_by to be set")
	}
}

func Test_AttachmentRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAttachmentRepository(testDB.Pool)
	attachment := &domain.Attachment{
		AssetID:  asset.ID,
		FileKey:  "attachments/test.jpg",
		FileName: "test.jpg",
		FileSize: 1024,
	}
	repo.Create(ctx, attachment)

	fetched, err := repo.GetByID(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected attachment to be found")
	}
	if fetched.FileName != "test.jpg" {
		t.Errorf("expected filename 'test.jpg', got '%s'", fetched.FileName)
	}
}

func Test_AttachmentRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewAttachmentRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent attachment")
	}
}

func Test_AttachmentRepository_ListByAsset_ReturnsAttachmentsForAsset(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset1, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 1")
	asset2, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Asset 2")

	repo := NewAttachmentRepository(testDB.Pool)
	repo.Create(ctx, &domain.Attachment{AssetID: asset1.ID, FileKey: "key1", FileName: "file1.jpg", FileSize: 100})
	repo.Create(ctx, &domain.Attachment{AssetID: asset1.ID, FileKey: "key2", FileName: "file2.jpg", FileSize: 200})
	repo.Create(ctx, &domain.Attachment{AssetID: asset2.ID, FileKey: "key3", FileName: "file3.jpg", FileSize: 300})

	attachments, err := repo.ListByAsset(ctx, asset1.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments for asset1, got %d", len(attachments))
	}
}

func Test_AttachmentRepository_ListByAsset_OrderedByCreatedAtDesc(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAttachmentRepository(testDB.Pool)
	repo.Create(ctx, &domain.Attachment{AssetID: asset.ID, FileKey: "key1", FileName: "first.jpg", FileSize: 100})
	repo.Create(ctx, &domain.Attachment{AssetID: asset.ID, FileKey: "key2", FileName: "second.jpg", FileSize: 200})

	attachments, err := repo.ListByAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	// Most recent should be first
	if attachments[0].FileName != "second.jpg" {
		t.Error("expected most recent attachment first")
	}
}

func Test_AttachmentRepository_Delete_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	repo := NewAttachmentRepository(testDB.Pool)
	attachment := &domain.Attachment{
		AssetID:  asset.ID,
		FileKey:  "attachments/to-delete.jpg",
		FileName: "to-delete.jpg",
		FileSize: 1024,
	}
	repo.Create(ctx, attachment)

	err := repo.Delete(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, attachment.ID)
	if fetched != nil {
		t.Error("expected attachment to be deleted")
	}
}

func Test_AttachmentRepository_Delete_CascadesOnAssetDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")
	cat, _ := fixtures.CreateCategory(ctx, org.ID, "Electronics", nil)
	asset, _ := fixtures.CreateAsset(ctx, org.ID, cat.ID, "Test Asset")

	attachmentRepo := NewAttachmentRepository(testDB.Pool)
	attachment := &domain.Attachment{
		AssetID:  asset.ID,
		FileKey:  "attachments/cascade.jpg",
		FileName: "cascade.jpg",
		FileSize: 1024,
	}
	attachmentRepo.Create(ctx, attachment)

	// Hard delete asset directly (to trigger cascade)
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM assets WHERE id = $1", asset.ID)
	if err != nil {
		t.Fatalf("failed to delete asset: %v", err)
	}

	// Attachment should be cascade deleted
	fetched, _ := attachmentRepo.GetByID(ctx, attachment.ID)
	if fetched != nil {
		t.Error("expected attachment to be cascade deleted with asset")
	}
}
