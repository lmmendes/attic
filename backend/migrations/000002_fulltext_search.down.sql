DROP TRIGGER IF EXISTS assets_search_vector_trigger ON assets;
DROP FUNCTION IF EXISTS assets_search_vector_update();
DROP INDEX IF EXISTS idx_assets_attributes;
DROP INDEX IF EXISTS idx_assets_search;
ALTER TABLE assets DROP COLUMN IF EXISTS search_vector;
