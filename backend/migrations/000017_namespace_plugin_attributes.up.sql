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

UPDATE attributes
SET key = 'plugin.' || plugin_id || '.' || key
WHERE plugin_id IS NOT NULL AND key NOT LIKE 'plugin.%';
