-- Add purchase fields to assets
ALTER TABLE assets ADD COLUMN purchase_at DATE;
ALTER TABLE assets ADD COLUMN purchase_note TEXT;
