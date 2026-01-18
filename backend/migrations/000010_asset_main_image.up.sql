-- Add main_attachment_id to assets table
ALTER TABLE assets ADD COLUMN main_attachment_id UUID REFERENCES attachments(id) ON DELETE SET NULL;

-- Index for quick lookup
CREATE INDEX idx_assets_main_attachment ON assets(main_attachment_id) WHERE main_attachment_id IS NOT NULL;
