-- Remove purchase_price field from assets
ALTER TABLE assets DROP COLUMN IF EXISTS purchase_price;
