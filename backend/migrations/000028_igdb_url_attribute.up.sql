BEGIN;

-- Temporary helpers keep the historical migration self-contained. The behavior
-- is checked against domain.NormalizeURLAttribute by migration integration tests.
CREATE FUNCTION pg_temp.igdb_url_trim(value text) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    SELECT btrim(value, E' \t\n\r\f' || chr(11)
        || U&'\0085\00A0\1680\2000\2001\2002\2003\2004\2005\2006\2007\2008\2009\200A\2028\2029\202F\205F\3000');
$$;

CREATE FUNCTION pg_temp.igdb_url_value(value jsonb) RETURNS jsonb
LANGUAGE plpgsql AS $$
DECLARE
    destination text;
    label text;
    authority text;
    hostname text;
    suffix text;
    zone text;
    escape_code text;
    decoded integer;
BEGIN
    IF value = 'null'::jsonb THEN RETURN '{}'::jsonb; END IF;
    IF jsonb_typeof(value) = 'string' THEN
        value := jsonb_build_object('url', value #>> '{}');
    END IF;
    IF jsonb_typeof(value) IS DISTINCT FROM 'object'
       OR jsonb_typeof(value->'url') IS DISTINCT FROM 'string'
       OR EXISTS (SELECT 1 FROM jsonb_object_keys(value) k WHERE k NOT IN ('url', 'label'))
       OR (value ? 'label' AND jsonb_typeof(value->'label') IS DISTINCT FROM 'string') THEN
        RETURN NULL;
    END IF;
    destination := pg_temp.igdb_url_trim(value->>'url');
    label := pg_temp.igdb_url_trim(COALESCE(value->>'label', ''));
    IF destination = '' AND label = '' THEN RETURN '{}'::jsonb; END IF;
    IF destination !~* '^https?://'
       OR destination ~ '[\x01-\x20\x7f\\]'
       OR regexp_replace(regexp_replace(destination, '^([^?#]*)\?[^#]*', '\1'), '%[[:xdigit:]]{2}', '', 'g') LIKE '%\%%' ESCAPE '\' THEN
        RETURN NULL;
    END IF;
    authority := substring(destination FROM '(?i)^https?://([^/?#]*)');
    -- Userinfo and characters rejected by Go's net/url host parser.
    IF authority = '' OR authority ~ '[@{}|^`]' THEN
        RETURN NULL;
    END IF;
    IF left(authority, 1) = '[' THEN
        hostname := substring(authority FROM '^\[([^]]+)\]');
        suffix := substring(authority FROM '^\[[^]]+\](.*)$');
        IF hostname IS NULL OR suffix IS NULL OR suffix !~ '^(:[0-9]*)?$' THEN
            RETURN NULL;
        END IF;
        IF position('%25' IN hostname) > 0 THEN
            zone := substring(hostname FROM position('%25' IN hostname) + 3);
            IF zone = '' THEN RETURN NULL; END IF;
            FOR escape_code IN SELECT (regexp_matches(zone, '%([[:xdigit:]]{2})', 'g'))[1] LOOP
                decoded := get_byte(decode(escape_code, 'hex'), 0);
                IF decoded = 0 OR decoded > 127 THEN RETURN NULL; END IF;
                IF decoded < 128 AND decoded <> 32 AND decoded <> 37
                   AND chr(decoded) !~ '^[A-Za-z0-9!$&''()*+,;=:\[\]<>"._~-]$' THEN
                    RETURN NULL;
                END IF;
            END LOOP;
        END IF;
        BEGIN
            IF family(split_part(hostname, '%25', 1)::inet) <> 6 THEN RETURN NULL; END IF;
        EXCEPTION WHEN invalid_text_representation THEN
            RETURN NULL;
        END;
    ELSE
        hostname := regexp_replace(authority, ':[0-9]*$', '');
        IF hostname = '' OR hostname LIKE '%:%' OR hostname ~ '[\[\]]' THEN
            RETURN NULL;
        END IF;
    END IF;
    -- Go permits escaped non-ASCII bytes and %25 in non-bracketed hosts.
    IF left(authority, 1) <> '[' AND regexp_replace(authority, '%25|%[89A-Fa-f][[:xdigit:]]', '', 'g') LIKE '%\%%' ESCAPE '\' THEN
        RETURN NULL;
    END IF;
    RETURN jsonb_build_object('url', destination)
        || CASE WHEN label <> '' THEN jsonb_build_object('label', label) ELSE '{}'::jsonb END;
END;
$$;

DO $$
DECLARE
    item record;
    normalized jsonb;
BEGIN
    FOR item IN
        SELECT a.id, a.organization_id, d.key, a.attributes->d.key AS value
        FROM assets a
        JOIN attributes d ON d.organization_id = a.organization_id
        WHERE d.plugin_id = 'igdb_games'
          AND d.key = 'plugin.igdb_games.games.url'
          AND d.data_type IN ('string', 'url')
          AND jsonb_typeof(a.attributes) = 'object'
          AND a.attributes ? d.key
        FOR UPDATE OF a
    LOOP
        normalized := pg_temp.igdb_url_value(item.value);
        IF normalized IS NULL OR normalized = '{}'::jsonb THEN
            IF normalized IS NULL THEN
                RAISE WARNING 'plugin URL migration: organization_id=% asset_id=% attribute_key=% reason=invalid URL value; removed',
                    item.organization_id, item.id, item.key;
            END IF;
            UPDATE assets SET attributes = attributes - item.key WHERE id = item.id;
        ELSIF normalized IS DISTINCT FROM item.value THEN
            UPDATE assets SET attributes = jsonb_set(attributes, ARRAY[item.key], normalized) WHERE id = item.id;
        END IF;
    END LOOP;
    UPDATE attributes SET data_type = 'url'
    WHERE plugin_id = 'igdb_games' AND key = 'plugin.igdb_games.games.url' AND data_type = 'string';
END;
$$;

DROP FUNCTION pg_temp.igdb_url_value(jsonb);
DROP FUNCTION pg_temp.igdb_url_trim(text);
COMMIT;
