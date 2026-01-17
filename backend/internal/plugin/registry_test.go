package plugin

import (
	"context"
	"sync"
	"testing"

	"github.com/mendelui/attic/internal/domain"
)

// mockPlugin implements domain.ImportPlugin for testing
type mockPlugin struct {
	id          string
	name        string
	description string
}

func (m *mockPlugin) ID() string                                { return m.id }
func (m *mockPlugin) Name() string                              { return m.name }
func (m *mockPlugin) Description() string                       { return m.description }
func (m *mockPlugin) CategoryName() string                      { return "Test Category" }
func (m *mockPlugin) CategoryDescription() string               { return "Test category description" }
func (m *mockPlugin) Attributes() []domain.PluginAttribute      { return nil }
func (m *mockPlugin) SearchFields() []domain.SearchField        { return nil }
func (m *mockPlugin) Search(_ context.Context, _, _ string, _ int) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockPlugin) Fetch(_ context.Context, _ string) (*domain.ImportData, error) {
	return nil, nil
}

func newMockPlugin(id, name, description string) *mockPlugin {
	return &mockPlugin{id: id, name: name, description: description}
}

func Test_NewRegistry_ReturnsEmptyRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	plugins := registry.List()
	if len(plugins) != 0 {
		t.Errorf("expected empty plugin list, got %d plugins", len(plugins))
	}
}

func Test_Register_NewPlugin_Succeeds(t *testing.T) {
	registry := NewRegistry()
	plugin := newMockPlugin("test_plugin", "Test Plugin", "A test plugin")

	err := registry.Register(plugin)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func Test_Register_DuplicatePlugin_ReturnsError(t *testing.T) {
	registry := NewRegistry()
	plugin1 := newMockPlugin("duplicate_id", "Plugin 1", "First plugin")
	plugin2 := newMockPlugin("duplicate_id", "Plugin 2", "Second plugin")

	err := registry.Register(plugin1)
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	err = registry.Register(plugin2)

	if err == nil {
		t.Fatal("expected error for duplicate plugin registration")
	}
}

func Test_Get_ExistingPlugin_ReturnsPlugin(t *testing.T) {
	registry := NewRegistry()
	plugin := newMockPlugin("my_plugin", "My Plugin", "Description")
	registry.Register(plugin)

	result, exists := registry.Get("my_plugin")

	if !exists {
		t.Fatal("expected plugin to exist")
	}
	if result.ID() != "my_plugin" {
		t.Errorf("expected ID 'my_plugin', got '%s'", result.ID())
	}
	if result.Name() != "My Plugin" {
		t.Errorf("expected name 'My Plugin', got '%s'", result.Name())
	}
}

func Test_Get_NonExistentPlugin_ReturnsFalse(t *testing.T) {
	registry := NewRegistry()

	_, exists := registry.Get("non_existent")

	if exists {
		t.Fatal("expected exists to be false for non-existent plugin")
	}
}

func Test_List_MultiplePlugins_ReturnsAll(t *testing.T) {
	registry := NewRegistry()
	registry.Register(newMockPlugin("plugin1", "Plugin 1", "Desc 1"))
	registry.Register(newMockPlugin("plugin2", "Plugin 2", "Desc 2"))
	registry.Register(newMockPlugin("plugin3", "Plugin 3", "Desc 3"))

	plugins := registry.List()

	if len(plugins) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(plugins))
	}
}

func Test_ListInfo_ReturnsPluginInfo(t *testing.T) {
	registry := NewRegistry()
	registry.Register(newMockPlugin("info_plugin", "Info Plugin", "Plugin for info test"))

	infos := registry.ListInfo()

	if len(infos) != 1 {
		t.Fatalf("expected 1 plugin info, got %d", len(infos))
	}
	if infos[0].ID != "info_plugin" {
		t.Errorf("expected ID 'info_plugin', got '%s'", infos[0].ID)
	}
	if infos[0].Name != "Info Plugin" {
		t.Errorf("expected name 'Info Plugin', got '%s'", infos[0].Name)
	}
	if infos[0].Description != "Plugin for info test" {
		t.Errorf("expected description 'Plugin for info test', got '%s'", infos[0].Description)
	}
}

func Test_Registry_ConcurrentAccess_IsThreadSafe(t *testing.T) {
	registry := NewRegistry()
	const numGoroutines = 100

	// Pre-register some plugins
	for i := 0; i < 10; i++ {
		registry.Register(newMockPlugin(
			"preexisting_"+string(rune('a'+i)),
			"Preexisting",
			"Desc",
		))
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent reads with Get
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			registry.Get("preexisting_a")
		}(i)
	}

	// Concurrent reads with List
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			registry.List()
		}(i)
	}

	// Concurrent reads with ListInfo
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			registry.ListInfo()
		}(i)
	}

	wg.Wait()
}

func Test_Registry_ConcurrentRegister_HandlesRaceCondition(t *testing.T) {
	registry := NewRegistry()
	const numGoroutines = 50

	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	// Multiple goroutines trying to register the same plugin ID
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := registry.Register(newMockPlugin("same_id", "Plugin", "Desc"))
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Only one registration should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful registration, got %d", successCount)
	}
}
