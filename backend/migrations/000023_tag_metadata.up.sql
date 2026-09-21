ALTER TABLE tags
    ADD COLUMN description TEXT,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE tags
SET name = CASE
    WHEN btrim(name) = '' THEN 'Tag ' || left(id::text, 8)
    ELSE btrim(name)
END;

CREATE TEMP TABLE tag_merge ON COMMIT DROP AS
SELECT id AS duplicate_id,
       first_value(id) OVER (
           PARTITION BY organization_id, lower(name)
           ORDER BY created_at, id
       ) AS canonical_id
FROM tags;

DELETE FROM tag_merge WHERE duplicate_id = canonical_id;

INSERT INTO asset_tags (asset_id, tag_id)
SELECT at.asset_id, tm.canonical_id
FROM asset_tags at
JOIN tag_merge tm ON tm.duplicate_id = at.tag_id
ON CONFLICT DO NOTHING;

DELETE FROM asset_tags
WHERE tag_id IN (SELECT duplicate_id FROM tag_merge);

DELETE FROM tags
WHERE id IN (SELECT duplicate_id FROM tag_merge);

ALTER TABLE tags DROP CONSTRAINT tags_organization_id_name_key;
ALTER TABLE tags ADD CONSTRAINT tags_name_not_blank CHECK (length(btrim(name)) BETWEEN 1 AND 100);
CREATE UNIQUE INDEX tags_organization_name_ci ON tags (organization_id, lower(name));
CREATE TRIGGER update_tags_updated_at BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
