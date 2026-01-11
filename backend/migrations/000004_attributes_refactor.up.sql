-- Create standalone attributes table (organization-level, reusable)
CREATE TABLE attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    data_type VARCHAR(20) NOT NULL CHECK (data_type IN ('string', 'number', 'boolean', 'date', 'text')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(organization_id, key)
);

-- Create junction table linking categories to attributes
CREATE TABLE category_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    attribute_id UUID NOT NULL REFERENCES attributes(id) ON DELETE CASCADE,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(category_id, attribute_id)
);

-- Create indexes
CREATE INDEX idx_attributes_organization ON attributes(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_category_attributes_category ON category_attributes(category_id);
CREATE INDEX idx_category_attributes_attribute ON category_attributes(attribute_id);

-- Add updated_at trigger for attributes
CREATE TRIGGER update_attributes_updated_at BEFORE UPDATE ON attributes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migrate existing attribute_definitions to new structure
-- First, create attributes from unique (organization_id via category, name, key, data_type) combinations
INSERT INTO attributes (id, organization_id, name, key, data_type, created_at, updated_at)
SELECT DISTINCT ON (c.organization_id, ad.key)
    ad.id,
    c.organization_id,
    ad.name,
    ad.key,
    ad.data_type,
    ad.created_at,
    ad.updated_at
FROM attribute_definitions ad
JOIN categories c ON c.id = ad.category_id
WHERE ad.deleted_at IS NULL;

-- Then, create category_attributes relationships
INSERT INTO category_attributes (category_id, attribute_id, required, sort_order, created_at)
SELECT
    ad.category_id,
    ad.id,
    ad.required,
    ad.sort_order,
    ad.created_at
FROM attribute_definitions ad
WHERE ad.deleted_at IS NULL;

-- Drop old table and related objects
DROP TRIGGER IF EXISTS update_attribute_definitions_updated_at ON attribute_definitions;
DROP INDEX IF EXISTS idx_attribute_definitions_category;
DROP TABLE attribute_definitions;
