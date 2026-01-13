-- Remove user authentication fields
DROP INDEX IF EXISTS idx_users_oidc_subject;
DROP INDEX IF EXISTS idx_users_email_unique;

ALTER TABLE users DROP COLUMN IF EXISTS role;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

-- Restore oidc_subject as NOT NULL (may fail if null values exist)
ALTER TABLE users ALTER COLUMN oidc_subject SET NOT NULL;

-- Restore original unique constraint
ALTER TABLE users ADD CONSTRAINT users_oidc_subject_key UNIQUE (oidc_subject);

DROP TYPE IF EXISTS user_role;
