-- name: CreateKindVersion :one
INSERT INTO kind_versions (id, kind_id, schema)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetKindVersion :one
SELECT * FROM kind_versions
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListKindVersionsByKind :many
SELECT * FROM kind_versions
WHERE kind_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetLatestKindVersion :one
SELECT * FROM kind_versions
WHERE kind_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;
