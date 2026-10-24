CREATE OR REPLACE FUNCTION assets_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.notes, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER assets_search_vector_trigger ON assets;
CREATE TRIGGER assets_search_vector_trigger
    BEFORE INSERT OR UPDATE OF name, description, notes ON assets
    FOR EACH ROW
    EXECUTE FUNCTION assets_search_vector_update();

-- Rebuild existing vectors without changing asset timestamps.
ALTER TABLE assets DISABLE TRIGGER update_assets_updated_at;
UPDATE assets SET search_vector =
        setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(notes, '')), 'C');

ALTER TABLE assets ENABLE TRIGGER update_assets_updated_at;
