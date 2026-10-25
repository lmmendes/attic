ALTER TABLE attributes DROP CONSTRAINT attributes_data_type_check;
ALTER TABLE attributes ADD CONSTRAINT attributes_data_type_check CHECK (data_type IN ('string','number','boolean','date','text','select','url'));
