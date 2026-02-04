package plugin

import (
	"fmt"
	"sync"

	"github.com/lmmendes/attic/internal/domain"
)

// Registry holds all registered import plugins
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]domain.ImportPlugin
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]domain.ImportPlugin),
	}
}

// Register adds a plugin to the registry
func (r *Registry) Register(plugin domain.ImportPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := plugin.ID()
	if _, exists := r.plugins[id]; exists {
		return fmt.Errorf("plugin %q already registered", id)
	}

	r.plugins[id] = plugin
	return nil
}

// Get retrieves a plugin by ID
func (r *Registry) Get(id string) (domain.ImportPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[id]
	return plugin, exists
}

// List returns all registered plugins
func (r *Registry) List() []domain.ImportPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]domain.ImportPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListInfo returns plugin info for all registered plugins
func (r *Registry) ListInfo() []domain.PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]domain.PluginInfo, 0, len(r.plugins))
	for _, p := range r.plugins {
		infos = append(infos, domain.PluginToInfo(p))
	}
	return infos
}
