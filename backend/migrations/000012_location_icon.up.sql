-- Add icon column to locations table
ALTER TABLE locations ADD COLUMN IF NOT EXISTS icon VARCHAR(100);
