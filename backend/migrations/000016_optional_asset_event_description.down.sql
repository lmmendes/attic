ALTER TABLE asset_events
    DROP CONSTRAINT asset_events_description_check;

UPDATE asset_events
SET description = title
WHERE length(trim(description)) = 0;

ALTER TABLE asset_events
    ALTER COLUMN description DROP DEFAULT,
    ADD CONSTRAINT asset_events_description_check
        CHECK (length(trim(description)) > 0 AND length(description) <= 2000);
