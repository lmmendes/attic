DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM location_photos GROUP BY location_id HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'Cannot roll back the photo gallery while locations have multiple photos';
    END IF;
END $$;
ALTER TABLE locations ADD COLUMN photo_key text;
UPDATE locations SET photo_key = p.file_key FROM location_photos p WHERE p.location_id = locations.id;
DROP TABLE location_photos;
