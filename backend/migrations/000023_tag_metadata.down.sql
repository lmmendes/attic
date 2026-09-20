DROP TRIGGER IF EXISTS update_tags_updated_at ON tags;
DROP INDEX IF EXISTS tags_organization_name_ci;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_name_not_blank;
ALTER TABLE tags ADD CONSTRAINT tags_organization_id_name_key UNIQUE (organization_id, name);
ALTER TABLE tags DROP COLUMN updated_at, DROP COLUMN description;
