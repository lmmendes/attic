DROP INDEX IF EXISTS idx_assets_main_attachment;
ALTER TABLE assets DROP COLUMN IF EXISTS main_attachment_id;
