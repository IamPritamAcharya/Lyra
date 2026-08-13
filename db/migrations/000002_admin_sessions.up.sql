CREATE TABLE admin_sessions (
  id BIGSERIAL PRIMARY KEY,
  token_hash BYTEA UNIQUE NOT NULL,
  csrf_token_hash BYTEA NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX admin_sessions_expires_at_idx ON admin_sessions(expires_at);
