-- name: CreateKind :one
INSERT INTO kinds (id, tenant_id, name, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetKind :one
SELECT * FROM kinds
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: ListKindsByTenant :many
SELECT * FROM kinds
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateKind :one
UPDATE kinds
SET name = $3,
    description = $4,
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteKind :exec
UPDATE kinds
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;
