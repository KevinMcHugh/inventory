-- name: CreateModel :one
INSERT INTO models (id, tenant_id, kind_id, kind_version_id, slug, body)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetModelByID :one
SELECT * FROM models
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: GetModelBySlug :one
SELECT * FROM models
WHERE tenant_id = $1 AND kind_id = $2 AND slug = $3 AND deleted_at IS NULL;

-- name: ListModelsByKind :many
SELECT * FROM models
WHERE tenant_id = $1 AND kind_id = $2 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateModelBySlug :one
UPDATE models
SET body = $4,
    kind_version_id = $5,
    updated_at = NOW()
WHERE tenant_id = $1 AND kind_id = $2 AND slug = $3 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteModelBySlug :exec
UPDATE models
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE tenant_id = $1 AND kind_id = $2 AND slug = $3 AND deleted_at IS NULL;
