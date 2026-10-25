BEGIN;
UPDATE assets a
SET attributes = jsonb_set(a.attributes, ARRAY[d.key], a.attributes->d.key->'url')
FROM attributes d
WHERE d.organization_id = a.organization_id
  AND d.plugin_id = 'igdb_games' AND d.key = 'plugin.igdb_games.games.url'
  AND d.data_type = 'url'
  AND jsonb_typeof(a.attributes) = 'object'
  AND jsonb_typeof(a.attributes->d.key->'url') = 'string';

UPDATE attributes SET data_type = 'string'
WHERE plugin_id = 'igdb_games' AND key = 'plugin.igdb_games.games.url' AND data_type = 'url';
-- Invalid values removed during upgrade cannot be recovered by rollback.
COMMIT;
