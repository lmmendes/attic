-- Drop triggers
DROP TRIGGER IF EXISTS update_warranties_updated_at ON warranties;
DROP TRIGGER IF EXISTS update_assets_updated_at ON assets;
DROP TRIGGER IF EXISTS update_locations_updated_at ON locations;
DROP TRIGGER IF EXISTS update_attribute_definitions_updated_at ON attribute_definitions;
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TRIGGER IF EXISTS update_conditions_updated_at ON conditions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS warranties;
DROP TABLE IF EXISTS asset_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS attribute_definitions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS conditions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
