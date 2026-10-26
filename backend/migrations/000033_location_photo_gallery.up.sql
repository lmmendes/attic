CREATE TABLE location_photos (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    file_key text NOT NULL UNIQUE,
    file_name text NOT NULL,
    is_cover boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX location_photos_cover ON location_photos(location_id) WHERE is_cover;
CREATE INDEX location_photos_order ON location_photos(location_id, created_at, id);
INSERT INTO location_photos(location_id, file_key, file_name, is_cover, created_at)
SELECT id, photo_key, 'Location photo', true, updated_at FROM locations WHERE photo_key IS NOT NULL;
ALTER TABLE locations DROP COLUMN photo_key;
