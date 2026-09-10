ALTER TABLE asset_events
    DROP CONSTRAINT asset_events_description_check;

ALTER TABLE asset_events
    ALTER COLUMN description SET DEFAULT '',
    ADD CONSTRAINT asset_events_description_check CHECK (length(description) <= 2000);
