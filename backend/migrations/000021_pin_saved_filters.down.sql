DROP INDEX IF EXISTS saved_filters_owner_pinned;

ALTER TABLE saved_filters
    DROP COLUMN IF EXISTS pinned;
