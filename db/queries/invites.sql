-- name: CreateInvite :one
INSERT INTO invites (id, code_hash, tenant_id, note, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListActiveInvites :many
SELECT * FROM invites
WHERE used_at IS NULL AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC;

-- name: ConsumeInviteByHash :one
UPDATE invites
SET used_at = NOW(),
    used_by_identity_id = $2,
    updated_at = NOW()
WHERE code_hash = $1
  AND used_at IS NULL
  AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
RETURNING *;

-- ClaimInviteByHash reserves an invite for later linking. It atomically marks
-- used_at NOW() so a race between two concurrent callbacks resolves cleanly.
-- The identity id is filled in afterwards via SetInviteUsedBy.

-- name: ClaimInviteByHash :one
UPDATE invites
SET used_at = NOW(),
    updated_at = NOW()
WHERE code_hash = $1
  AND used_at IS NULL
  AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
RETURNING *;

-- name: SetInviteUsedBy :exec
UPDATE invites
SET used_by_identity_id = $2,
    updated_at = NOW()
WHERE id = $1;
