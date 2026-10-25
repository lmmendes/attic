ALTER TABLE organizations ADD COLUMN currency TEXT NOT NULL DEFAULT 'USD' CHECK (currency ~ '^[A-Z]{3}$');
