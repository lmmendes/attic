UPDATE assets s SET attributes = (
 SELECT jsonb_object_agg(v.key, CASE WHEN d.data_type='url' AND jsonb_typeof(v.value)='object' THEN COALESCE(v.value->'url', '""'::jsonb) ELSE v.value END)
 FROM jsonb_each(s.attributes) v
 LEFT JOIN attributes d ON d.organization_id=s.organization_id AND d.key=v.key
) WHERE jsonb_typeof(s.attributes)='object' AND EXISTS (
 SELECT 1 FROM attributes d WHERE d.organization_id=s.organization_id AND d.data_type='url' AND s.attributes ? d.key
);
UPDATE attributes SET data_type='string' WHERE data_type='url';
ALTER TABLE attributes DROP CONSTRAINT attributes_data_type_check;
ALTER TABLE attributes ADD CONSTRAINT attributes_data_type_check CHECK (data_type IN ('string','number','boolean','date','text','select'));
