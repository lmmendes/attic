CREATE TABLE feature_configurations (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    collections_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    plugins_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    conditions_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    locations_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO feature_configurations (organization_id)
SELECT id FROM organizations
ON CONFLICT (organization_id) DO NOTHING;

CREATE TRIGGER update_feature_configurations_updated_at
    BEFORE UPDATE ON feature_configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
