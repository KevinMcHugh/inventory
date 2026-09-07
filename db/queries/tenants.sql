-- name: CreateTenant :one
INSERT INTO tenants (id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTenants :many
SELECT * FROM tenants
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateTenant :one
UPDATE tenants
SET name = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteTenant :exec
UPDATE tenants
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;
