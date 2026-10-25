CREATE TABLE oidc_sessions (
 session_id_hash TEXT PRIMARY KEY,
 access_token TEXT NOT NULL,
 refresh_token TEXT NOT NULL,
 id_token TEXT NOT NULL,
 subject TEXT NOT NULL,
 email TEXT NOT NULL,
 name TEXT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX oidc_sessions_expires_at_idx ON oidc_sessions (expires_at);
