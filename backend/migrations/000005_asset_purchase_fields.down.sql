-- Remove purchase fields from assets
ALTER TABLE assets DROP COLUMN IF EXISTS purchase_at;
ALTER TABLE assets DROP COLUMN IF EXISTS purchase_note;
