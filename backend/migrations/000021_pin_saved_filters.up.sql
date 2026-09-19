ALTER TABLE saved_filters
    ADD COLUMN pinned BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX saved_filters_owner_pinned
    ON saved_filters(organization_id, user_id, pinned)
    WHERE pinned = TRUE;
