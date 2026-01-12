-- Import Plugins: Add plugin support to categories, attributes, and assets

-- Add plugin_id to categories (NULL = user-created, non-NULL = plugin-managed)
ALTER TABLE categories ADD COLUMN plugin_id VARCHAR(50);

-- Add plugin_id to attributes (NULL = user-defined/reusable, non-NULL = plugin-owned/namespaced)
ALTER TABLE attributes ADD COLUMN plugin_id VARCHAR(50);

-- Add import source tracking to assets
ALTER TABLE assets ADD COLUMN import_plugin_id VARCHAR(50);
ALTER TABLE assets ADD COLUMN import_external_id VARCHAR(255);

-- Index for finding assets by import source
CREATE INDEX idx_assets_import_source ON assets(import_plugin_id, import_external_id)
    WHERE import_plugin_id IS NOT NULL;

-- Index for finding plugin-managed categories
CREATE INDEX idx_categories_plugin ON categories(plugin_id)
    WHERE plugin_id IS NOT NULL;

-- Index for finding plugin-owned attributes
CREATE INDEX idx_attributes_plugin ON attributes(plugin_id)
    WHERE plugin_id IS NOT NULL;
