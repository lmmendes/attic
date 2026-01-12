-- Revert import plugins changes

DROP INDEX IF EXISTS idx_attributes_plugin;
DROP INDEX IF EXISTS idx_categories_plugin;
DROP INDEX IF EXISTS idx_assets_import_source;

ALTER TABLE assets DROP COLUMN IF EXISTS import_external_id;
ALTER TABLE assets DROP COLUMN IF EXISTS import_plugin_id;
ALTER TABLE attributes DROP COLUMN IF EXISTS plugin_id;
ALTER TABLE categories DROP COLUMN IF EXISTS plugin_id;
