-- name: GetUserIdentityByProviderSubject :one
SELECT * FROM user_identities
WHERE provider = $1 AND subject = $2 AND deleted_at IS NULL;

-- name: CreateUserIdentity :one
INSERT INTO user_identities (id, tenant_id, provider, subject, email, display_name, last_login_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING *;

-- name: TouchUserIdentity :exec
UPDATE user_identities
SET last_login_at = NOW(),
    email = $2,
    display_name = COALESCE($3, display_name),
    updated_at = NOW()
WHERE id = $1;

-- name: CreateSSOLoginIntent :one
INSERT INTO sso_login_intents (
    state, provider, return_to,
    client_id, redirect_uri, code_challenge, code_challenge_method, downstream_state, scope,
    expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ConsumeSSOLoginIntent :one
DELETE FROM sso_login_intents
WHERE state = $1 AND expires_at > NOW()
RETURNING *;

-- name: PurgeExpiredSSOIntents :exec
DELETE FROM sso_login_intents WHERE expires_at < NOW();
