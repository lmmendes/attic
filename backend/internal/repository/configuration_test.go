package repository

import (
	"context"
	"testing"

	"github.com/lmmendes/attic/internal/testutil"
)

func Test_ConfigurationRepository_DefaultsAndUpdate(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}

	fixtures := testutil.NewFixtures(testDB.Pool)
	organization, err := fixtures.CreateOrganization(ctx, "Test Org")
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	repository := NewConfigurationRepository(testDB.Pool)
	configuration, err := repository.Get(ctx, organization.ID)
	if err != nil {
		t.Fatalf("failed to get configuration: %v", err)
	}
	if !configuration.CollectionsEnabled || !configuration.PluginsEnabled || !configuration.ConditionsEnabled || !configuration.LocationsEnabled {
		t.Fatal("expected every feature to be enabled by default")
	}

	configuration.PluginsEnabled = false
	configuration.LocationsEnabled = false
	if err := repository.Update(ctx, configuration); err != nil {
		t.Fatalf("failed to update configuration: %v", err)
	}

	fetched, err := repository.Get(ctx, organization.ID)
	if err != nil {
		t.Fatalf("failed to reload configuration: %v", err)
	}
	if fetched.PluginsEnabled || fetched.LocationsEnabled {
		t.Fatal("expected disabled feature settings to persist")
	}
	if !fetched.CollectionsEnabled || !fetched.ConditionsEnabled {
		t.Fatal("expected unchanged feature settings to remain enabled")
	}
}
