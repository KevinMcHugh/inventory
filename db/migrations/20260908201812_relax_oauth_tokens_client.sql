-- migrate:up

-- Web-app sessions minted from SSO do not have a downstream OAuth client
-- driving them. Relax the client_id column to allow those tokens.
ALTER TABLE oauth_tokens ALTER COLUMN client_id DROP NOT NULL;

-- Same for oauth_codes so a future SSO-then-authz flow can leave client_id
-- empty during handshake and populate it before persistence. (Not needed
-- today but keeps the two tables symmetric.)
ALTER TABLE oauth_codes ALTER COLUMN client_id DROP NOT NULL;

-- migrate:down

ALTER TABLE oauth_codes ALTER COLUMN client_id SET NOT NULL;
ALTER TABLE oauth_tokens ALTER COLUMN client_id SET NOT NULL;
