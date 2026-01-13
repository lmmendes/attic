-- Add user authentication fields
CREATE TYPE user_role AS ENUM ('user', 'admin');

-- Make oidc_subject nullable (users can exist without OIDC)
ALTER TABLE users ALTER COLUMN oidc_subject DROP NOT NULL;

-- Add new authentication columns
ALTER TABLE users ADD COLUMN password_hash VARCHAR(255);
ALTER TABLE users ADD COLUMN role user_role NOT NULL DEFAULT 'user';

-- Add index for email lookups (for login)
CREATE UNIQUE INDEX idx_users_email_unique ON users(LOWER(email)) WHERE deleted_at IS NULL;

-- Drop the existing UNIQUE constraint on oidc_subject and create a partial unique index instead
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_oidc_subject_key;
CREATE UNIQUE INDEX idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL AND deleted_at IS NULL;
