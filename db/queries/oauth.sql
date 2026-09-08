-- name: CreateOAuthClient :one
INSERT INTO oauth_clients (id, client_secret_hash, redirect_uris, client_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateOAuthCode :one
INSERT INTO oauth_codes (
    code_hash, client_id, tenant_id, redirect_uri,
    code_challenge, code_challenge_method, scope, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ConsumeOAuthCode :one
UPDATE oauth_codes
SET consumed_at = NOW()
WHERE code_hash = $1
  AND consumed_at IS NULL
  AND expires_at > NOW()
RETURNING *;

-- name: CreateOAuthToken :one
INSERT INTO oauth_tokens (id, token_hash, client_id, tenant_id, scope, expires_at)
VALUES ($1, $2, sqlc.narg(client_id), $3, $4, $5)
RETURNING *;

-- name: GetOAuthTokenByHash :one
SELECT * FROM oauth_tokens
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW();
