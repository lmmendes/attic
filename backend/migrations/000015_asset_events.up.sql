CREATE TABLE asset_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL CHECK (length(trim(title)) > 0),
    description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 2000),
    icon VARCHAR(100) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_asset_events_asset_occurred_at
    ON asset_events(asset_id, occurred_at DESC, created_at DESC);

CREATE TRIGGER update_asset_events_updated_at
    BEFORE UPDATE ON asset_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
