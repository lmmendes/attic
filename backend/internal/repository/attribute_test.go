package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/domain"
	"github.com/mendelui/attic/internal/testutil"
)

func Test_AttributeRepository_Create_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Brand",
		Key:            "brand",
		DataType:       domain.AttributeTypeString,
	}

	err := repo.Create(ctx, attr)
	if err != nil {
		t.Fatalf("failed to create attribute: %v", err)
	}

	if attr.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if attr.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func Test_AttributeRepository_Create_WithPluginID(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	pluginID := "google_books"
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		PluginID:       &pluginID,
		Name:           "ISBN",
		Key:            "isbn",
		DataType:       domain.AttributeTypeString,
	}

	err := repo.Create(ctx, attr)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, attr.ID)
	if fetched.PluginID == nil || *fetched.PluginID != pluginID {
		t.Error("expected plugin ID to be set")
	}
}

func Test_AttributeRepository_GetByID_Exists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Model",
		Key:            "model",
		DataType:       domain.AttributeTypeString,
	}
	repo.Create(ctx, attr)

	fetched, err := repo.GetByID(ctx, attr.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected attribute to be found")
	}
	if fetched.Name != "Model" {
		t.Errorf("expected name 'Model', got '%s'", fetched.Name)
	}
}

func Test_AttributeRepository_GetByID_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	repo := NewAttributeRepository(testDB.Pool)
	fetched, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent attribute")
	}
}

func Test_AttributeRepository_GetByKey(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Serial Number",
		Key:            "serial_number",
		DataType:       domain.AttributeTypeString,
	}
	repo.Create(ctx, attr)

	fetched, err := repo.GetByKey(ctx, org.ID, "serial_number")
	if err != nil {
		t.Fatalf("failed to get by key: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected attribute to be found")
	}
	if fetched.ID != attr.ID {
		t.Error("expected same attribute")
	}
}

func Test_AttributeRepository_GetByKey_NotExists(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	fetched, err := repo.GetByKey(ctx, org.ID, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for non-existent key")
	}
}

func Test_AttributeRepository_List_ReturnsAttributesForOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org1, _ := fixtures.CreateOrganization(ctx, "Org 1")
	org2, _ := fixtures.CreateOrganization(ctx, "Org 2")

	repo := NewAttributeRepository(testDB.Pool)
	repo.Create(ctx, &domain.Attribute{OrganizationID: org1.ID, Name: "Brand", Key: "brand", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org1.ID, Name: "Model", Key: "model", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org2.ID, Name: "Other", Key: "other", DataType: domain.AttributeTypeString})

	attributes, err := repo.List(ctx, org1.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if len(attributes) != 2 {
		t.Errorf("expected 2 attributes for org1, got %d", len(attributes))
	}
}

func Test_AttributeRepository_List_OrderedByName(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "Zebra", Key: "zebra", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "Alpha", Key: "alpha", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "Beta", Key: "beta", DataType: domain.AttributeTypeString})

	attributes, err := repo.List(ctx, org.ID)
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}

	if attributes[0].Name != "Alpha" {
		t.Error("expected attributes to be ordered by name")
	}
}

func Test_AttributeRepository_ListByPluginID(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	pluginID := "google_books"
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, PluginID: &pluginID, Name: "ISBN", Key: "isbn", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, PluginID: &pluginID, Name: "Author", Key: "author", DataType: domain.AttributeTypeString})
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "User Defined", Key: "user_defined", DataType: domain.AttributeTypeString})

	attributes, err := repo.ListByPluginID(ctx, org.ID, pluginID)
	if err != nil {
		t.Fatalf("failed to list by plugin: %v", err)
	}

	if len(attributes) != 2 {
		t.Errorf("expected 2 plugin attributes, got %d", len(attributes))
	}
}

func Test_AttributeRepository_Update_Success(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "Old Name",
		Key:            "old_key",
		DataType:       domain.AttributeTypeString,
	}
	repo.Create(ctx, attr)

	// Update name and data type (key is immutable in the update function)
	attr.Name = "New Name"
	attr.DataType = domain.AttributeTypeNumber

	err := repo.Update(ctx, attr)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, attr.ID)
	if fetched.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", fetched.Name)
	}
	if fetched.DataType != domain.AttributeTypeNumber {
		t.Errorf("expected data type 'number', got '%s'", fetched.DataType)
	}
}

func Test_AttributeRepository_Delete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	attr := &domain.Attribute{
		OrganizationID: org.ID,
		Name:           "To Delete",
		Key:            "to_delete",
		DataType:       domain.AttributeTypeString,
	}
	repo.Create(ctx, attr)

	err := repo.Delete(ctx, attr.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	fetched, _ := repo.GetByID(ctx, attr.ID)
	if fetched != nil {
		t.Error("expected deleted attribute not to be found")
	}
}

func Test_AttributeRepository_UniqueKeyPerOrg(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)
	repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "Brand", Key: "brand", DataType: domain.AttributeTypeString})

	// Try to create duplicate key
	err := repo.Create(ctx, &domain.Attribute{OrganizationID: org.ID, Name: "Another Brand", Key: "brand", DataType: domain.AttributeTypeString})
	if err == nil {
		t.Error("expected error for duplicate key within same org")
	}
}

func Test_AttributeRepository_AllDataTypes(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	org, _ := fixtures.CreateOrganization(ctx, "Test Org")

	repo := NewAttributeRepository(testDB.Pool)

	dataTypes := []domain.AttributeDataType{
		domain.AttributeTypeString,
		domain.AttributeTypeNumber,
		domain.AttributeTypeBoolean,
		domain.AttributeTypeText,
		domain.AttributeTypeDate,
	}

	for _, dt := range dataTypes {
		attr := &domain.Attribute{
			OrganizationID: org.ID,
			Name:           "Attr " + string(dt),
			Key:            string(dt),
			DataType:       dt,
		}
		err := repo.Create(ctx, attr)
		if err != nil {
			t.Errorf("failed to create attribute with data type %s: %v", dt, err)
		}

		fetched, err := repo.GetByID(ctx, attr.ID)
		if err != nil {
			t.Errorf("failed to get attribute with data type %s: %v", dt, err)
		}
		if fetched.DataType != dt {
			t.Errorf("expected data type %s, got %s", dt, fetched.DataType)
		}
	}
}
