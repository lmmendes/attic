ALTER TABLE organizations ADD COLUMN mcp_enabled boolean NOT NULL DEFAULT false;
CREATE TABLE api_keys (
    id uuid PRIMARY KEY,
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    description varchar(255) NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 255),
    permissions jsonb NOT NULL CHECK (jsonb_typeof(permissions) = 'object' AND permissions <> '{}'::jsonb),
    secret_hash bytea NOT NULL UNIQUE CHECK (octet_length(secret_hash) = 32),
    prefix text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    revoked_at timestamptz,
    last_used_at timestamptz,
    CHECK (expires_at IS NULL OR expires_at > created_at)
);
CREATE INDEX api_keys_organization_owner ON api_keys(organization_id, owner_id);
