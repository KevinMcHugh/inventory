-- migrate:up

ALTER TABLE oauth_tokens ADD COLUMN refresh_token_hash TEXT UNIQUE;
ALTER TABLE oauth_tokens ADD COLUMN refresh_expires_at TIMESTAMPTZ;

-- migrate:down

ALTER TABLE oauth_tokens DROP COLUMN refresh_expires_at;
ALTER TABLE oauth_tokens DROP COLUMN refresh_token_hash;
