package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/plugin"
)

type startupTestPlugin struct{}

func (startupTestPlugin) ID() string                           { return "test_plugin" }
func (startupTestPlugin) Name() string                         { return "Test Plugin" }
func (startupTestPlugin) Description() string                  { return "Test plugin" }
func (startupTestPlugin) Enabled() bool                        { return true }
func (startupTestPlugin) DisabledReason() string               { return "" }
func (startupTestPlugin) CategoryName() string                 { return "Test Category" }
func (startupTestPlugin) CategoryDescription() string          { return "Test category" }
func (startupTestPlugin) Attributes() []domain.PluginAttribute { return nil }
func (startupTestPlugin) SearchFields() []domain.SearchField   { return nil }
func (startupTestPlugin) Search(context.Context, string, string, int) ([]domain.SearchResult, error) {
	return nil, nil
}
func (startupTestPlugin) Fetch(context.Context, string) (*domain.ImportData, error) {
	return nil, nil
}

func TestRegisterImportPluginLogsAPIKeyPresenceWithoutValue(t *testing.T) {
	const apiKeyEnvironmentVariable = "ATTIC_TEST_API_KEY"
	const apiKey = "secret-test-api-key"
	t.Setenv(apiKeyEnvironmentVariable, apiKey)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	registry := plugin.NewRegistry()
	registerImportPlugin(registry, startupTestPlugin{}, apiKeyEnvironmentVariable)

	logOutput := output.String()
	if !strings.Contains(logOutput, `"api_key_environment_variable":"ATTIC_TEST_API_KEY"`) {
		t.Fatalf("expected API key environment variable in log, got %s", logOutput)
	}
	if !strings.Contains(logOutput, `"api_key_set":true`) {
		t.Fatalf("expected configured API key status in log, got %s", logOutput)
	}
	if strings.Contains(logOutput, apiKey) {
		t.Fatal("API key value must not be written to logs")
	}
}
