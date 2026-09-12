CREATE TABLE organization_feature_settings (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    locations BOOLEAN NOT NULL DEFAULT TRUE,
    collections BOOLEAN NOT NULL DEFAULT TRUE,
    categories BOOLEAN NOT NULL DEFAULT TRUE,
    attributes BOOLEAN NOT NULL DEFAULT TRUE,
    conditions BOOLEAN NOT NULL DEFAULT TRUE,
    warranties BOOLEAN NOT NULL DEFAULT TRUE,
    plugins BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO organization_feature_settings (organization_id)
SELECT id FROM organizations WHERE deleted_at IS NULL
ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION create_default_organization_features()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO organization_feature_settings (organization_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER organizations_feature_settings
AFTER INSERT ON organizations FOR EACH ROW
EXECUTE FUNCTION create_default_organization_features();
