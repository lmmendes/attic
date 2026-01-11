-- Add full-text search column to assets
ALTER TABLE assets ADD COLUMN search_vector tsvector;

-- Create GIN index for full-text search
CREATE INDEX idx_assets_search ON assets USING GIN(search_vector);

-- Create GIN index for JSONB attributes
CREATE INDEX idx_assets_attributes ON assets USING GIN(attributes);

-- Function to update search vector
CREATE OR REPLACE FUNCTION assets_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update search vector
CREATE TRIGGER assets_search_vector_trigger
    BEFORE INSERT OR UPDATE OF name, description ON assets
    FOR EACH ROW
    EXECUTE FUNCTION assets_search_vector_update();

-- Update existing rows
UPDATE assets SET search_vector =
    setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B');
