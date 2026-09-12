
ALTER TABLE attributes DROP CONSTRAINT attributes_data_type_check;
ALTER TABLE attributes ADD COLUMN selection_mode VARCHAR(20);
ALTER TABLE attributes ADD CONSTRAINT attributes_data_type_check
 CHECK (data_type IN ('string', 'number', 'boolean', 'date', 'text', 'select'));
ALTER TABLE attributes ADD CONSTRAINT attributes_selection_mode_check
 CHECK ((data_type = 'select' AND selection_mode IS NOT NULL AND selection_mode IN ('single', 'multiple'))
 OR (data_type <> 'select' AND selection_mode IS NULL));

CREATE TABLE attribute_options (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 attribute_id UUID NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
 label VARCHAR(255) NOT NULL CHECK (length(btrim(label)) > 0),
 value VARCHAR(255) NOT NULL CHECK (length(btrim(value)) > 0),
 sort_order INTEGER NOT NULL DEFAULT 0,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(attribute_id, value)
);
CREATE UNIQUE INDEX attribute_options_labels ON attribute_options(attribute_id, lower(label));
CREATE INDEX attribute_options_order ON attribute_options(attribute_id, sort_order, id);
CREATE TRIGGER update_attribute_options_updated_at BEFORE UPDATE ON attribute_options
 FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Persist signing material so previews work across restarts and replicas.
CREATE TABLE attribute_impact_secrets (
 organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
 secret BYTEA NOT NULL DEFAULT decode(replace(gen_random_uuid()::text || gen_random_uuid()::text, '-', ''), 'hex')
);
INSERT INTO attribute_impact_secrets(organization_id) SELECT id FROM organizations;
CREATE FUNCTION create_attribute_impact_secret() RETURNS TRIGGER AS $$
BEGIN
 INSERT INTO attribute_impact_secrets(organization_id) VALUES (NEW.id);
 RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER organizations_attribute_impact_secret AFTER INSERT ON organizations
 FOR EACH ROW EXECUTE FUNCTION create_attribute_impact_secret();

-- Retain renamed/deleted keys so stale clients cannot reintroduce removed data.
CREATE TABLE retired_attribute_keys (
 organization_id UUID NOT NULL REFERENCES organizations(id),
 key VARCHAR(100) NOT NULL,
 PRIMARY KEY(organization_id, key)
);
