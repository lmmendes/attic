
-- Refuse rollback until select definitions have been removed explicitly.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM attributes WHERE data_type = 'select' AND deleted_at IS NULL) THEN
  RAISE EXCEPTION 'Remove select attributes before rolling back this migration';
 END IF;
END $$;
DROP TRIGGER organizations_attribute_impact_secret ON organizations;
DROP FUNCTION create_attribute_impact_secret();
DROP TABLE attribute_impact_secrets;
DROP TABLE retired_attribute_keys;
DROP TABLE attribute_options;
ALTER TABLE attributes DROP CONSTRAINT attributes_selection_mode_check;
ALTER TABLE attributes DROP CONSTRAINT attributes_data_type_check;
UPDATE attributes SET data_type = 'string' WHERE data_type = 'select';
ALTER TABLE attributes DROP COLUMN selection_mode;
ALTER TABLE attributes ADD CONSTRAINT attributes_data_type_check
 CHECK (data_type IN ('string', 'number', 'boolean', 'date', 'text'));
