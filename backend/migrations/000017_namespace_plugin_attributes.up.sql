-- Namespace plugin-owned attributes and only the corresponding keys in
-- imported assets. Existing user-defined keys on those assets are retained.
UPDATE assets AS asset
SET attributes = COALESCE(mapped.attributes, '{}'::jsonb)
FROM (
    SELECT a.id, jsonb_object_agg(
        CASE
            WHEN item.key LIKE 'plugin.%' THEN item.key
            WHEN EXISTS (
                SELECT 1 FROM attributes definition
                WHERE definition.plugin_id = a.import_plugin_id AND definition.key = item.key
            ) THEN 'plugin.' || a.import_plugin_id || '.' || item.key
            ELSE item.key
        END,
        item.value ORDER BY (item.key LIKE 'plugin.%')
    ) AS attributes
    FROM assets a
    CROSS JOIN LATERAL jsonb_each(CASE WHEN jsonb_typeof(a.attributes) = 'object' THEN a.attributes ELSE '{}'::jsonb END) item
    WHERE a.import_plugin_id IS NOT NULL
    GROUP BY a.id
) mapped
WHERE asset.id = mapped.id;

-- If both legacy and already-namespaced definitions exist, retain the
-- namespaced definition and move every category assignment onto it before the
-- key update. A category linked to both keeps required=true if either link was
-- required and keeps the earliest sort position.
WITH duplicate_definitions AS (
    SELECT legacy.id AS duplicate_id, retained.id AS retained_id
    FROM attributes legacy
    JOIN attributes retained
      ON retained.organization_id = legacy.organization_id
     AND retained.plugin_id = legacy.plugin_id
     AND retained.key = 'plugin.' || legacy.plugin_id || '.' || legacy.key
    WHERE legacy.plugin_id IS NOT NULL
      AND legacy.key NOT LIKE 'plugin.%'
)
INSERT INTO category_attributes (category_id, attribute_id, required, sort_order, created_at)
SELECT assignment.category_id,
       duplicate.retained_id,
       assignment.required,
       assignment.sort_order,
       assignment.created_at
FROM category_attributes assignment
JOIN duplicate_definitions duplicate ON duplicate.duplicate_id = assignment.attribute_id
ON CONFLICT (category_id, attribute_id) DO UPDATE
SET required = category_attributes.required OR EXCLUDED.required,
    sort_order = LEAST(category_attributes.sort_order, EXCLUDED.sort_order),
    created_at = LEAST(category_attributes.created_at, EXCLUDED.created_at);

WITH duplicate_definitions AS (
    SELECT legacy.id AS duplicate_id
    FROM attributes legacy
    JOIN attributes retained
      ON retained.organization_id = legacy.organization_id
     AND retained.plugin_id = legacy.plugin_id
     AND retained.key = 'plugin.' || legacy.plugin_id || '.' || legacy.key
    WHERE legacy.plugin_id IS NOT NULL
      AND legacy.key NOT LIKE 'plugin.%'
)
DELETE FROM category_attributes assignment
USING duplicate_definitions duplicate
WHERE assignment.attribute_id = duplicate.duplicate_id;

WITH duplicate_definitions AS (
    SELECT legacy.id AS duplicate_id
    FROM attributes legacy
    JOIN attributes retained
      ON retained.organization_id = legacy.organization_id
     AND retained.plugin_id = legacy.plugin_id
     AND retained.key = 'plugin.' || legacy.plugin_id || '.' || legacy.key
    WHERE legacy.plugin_id IS NOT NULL
      AND legacy.key NOT LIKE 'plugin.%'
)
DELETE FROM attributes definition
USING duplicate_definitions duplicate
WHERE definition.id = duplicate.duplicate_id;

UPDATE attributes
SET key = 'plugin.' || plugin_id || '.' || key
WHERE plugin_id IS NOT NULL AND key NOT LIKE 'plugin.%';
