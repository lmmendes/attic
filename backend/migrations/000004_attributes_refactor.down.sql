-- Recreate old attribute_definitions table
CREATE TABLE attribute_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    data_type VARCHAR(20) NOT NULL CHECK (data_type IN ('string', 'number', 'boolean', 'enum', 'text', 'date')),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    enum_options JSONB,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(category_id, key)
);

CREATE INDEX idx_attribute_definitions_category ON attribute_definitions(category_id) WHERE deleted_at IS NULL;
CREATE TRIGGER update_attribute_definitions_updated_at BEFORE UPDATE ON attribute_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migrate data back (best effort - some data may be lost if attributes were shared)
INSERT INTO attribute_definitions (category_id, name, key, data_type, required, sort_order, created_at, updated_at)
SELECT
    ca.category_id,
    a.name,
    a.key,
    a.data_type,
    ca.required,
    ca.sort_order,
    a.created_at,
    a.updated_at
FROM category_attributes ca
JOIN attributes a ON a.id = ca.attribute_id
WHERE a.deleted_at IS NULL;

-- Drop new tables
DROP TRIGGER IF EXISTS update_attributes_updated_at ON attributes;
DROP INDEX IF EXISTS idx_category_attributes_attribute;
DROP INDEX IF EXISTS idx_category_attributes_category;
DROP INDEX IF EXISTS idx_attributes_organization;
DROP TABLE category_attributes;
DROP TABLE attributes;
