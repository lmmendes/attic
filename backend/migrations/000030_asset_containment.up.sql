ALTER TABLE assets ADD COLUMN parent_id UUID;
ALTER TABLE assets ADD CONSTRAINT assets_parent_fk FOREIGN KEY (parent_id) REFERENCES assets(id) ON DELETE SET NULL;
ALTER TABLE assets ADD CONSTRAINT assets_not_own_parent CHECK (parent_id IS DISTINCT FROM id);
CREATE INDEX idx_assets_parent ON assets(organization_id, parent_id) WHERE deleted_at IS NULL;
