DROP TRIGGER IF EXISTS update_asset_events_updated_at ON asset_events;
DROP INDEX IF EXISTS idx_asset_events_asset_date;
DROP TABLE IF EXISTS asset_events;
