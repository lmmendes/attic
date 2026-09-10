package handler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
)

func TestApplyConfigurationUpdate_PreservesOmittedValues(t *testing.T) {
	disabled := false
	configuration := &domain.FeatureConfiguration{
		OrganizationID:     uuid.New(),
		CollectionsEnabled: true,
		PluginsEnabled:     true,
		ConditionsEnabled:  true,
		LocationsEnabled:   true,
		WarrantiesEnabled:  true,
	}

	applyConfigurationUpdate(configuration, UpdateConfigurationRequest{PluginsEnabled: &disabled})

	if configuration.PluginsEnabled {
		t.Fatal("expected plugins to be disabled")
	}
	if !configuration.CollectionsEnabled || !configuration.ConditionsEnabled || !configuration.LocationsEnabled {
		t.Fatal("omitted feature settings must be preserved")
	}
}

func TestFeatureEnabled(t *testing.T) {
	configuration := &domain.FeatureConfiguration{
		CollectionsEnabled: true,
		PluginsEnabled:     false,
		ConditionsEnabled:  true,
		LocationsEnabled:   false,
		WarrantiesEnabled:  true,
	}

	tests := map[string]bool{
		"collections": true,
		"plugins":     false,
		"conditions":  true,
		"locations":   false,
		"warranties":  true,
		"unknown":     false,
	}
	for feature, expected := range tests {
		if actual := featureEnabled(configuration, feature); actual != expected {
			t.Errorf("featureEnabled(%q) = %v, want %v", feature, actual, expected)
		}
	}
}
