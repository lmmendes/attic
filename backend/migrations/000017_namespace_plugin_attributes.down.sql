UPDATE attributes
SET key = regexp_replace(key, '^plugin\.[^.]+\.', '')
WHERE plugin_id IS NOT NULL AND key LIKE 'plugin.%';

UPDATE assets AS asset
SET attributes = mapped.attributes
FROM (
    SELECT a.id, jsonb_object_agg(
        CASE
            WHEN item.key LIKE 'plugin.' || a.import_plugin_id || '.%'
                THEN regexp_replace(item.key, '^plugin\.[^.]+\.', '')
            ELSE item.key
        END,
        item.value
    ) AS attributes
    FROM assets a
    CROSS JOIN LATERAL jsonb_each(CASE WHEN jsonb_typeof(a.attributes) = 'object' THEN a.attributes ELSE '{}'::jsonb END) item
    WHERE a.import_plugin_id IS NOT NULL
    GROUP BY a.id
) mapped
WHERE asset.id = mapped.id;
