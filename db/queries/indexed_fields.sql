-- name: CreateIndexedField :one
INSERT INTO indexed_fields (
    id, tenant_id, kind_id, model_id, field_key,
    string_value, numeric_value, int_value, bool_value, date_value
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: DeleteIndexedFieldsByModel :exec
DELETE FROM indexed_fields
WHERE tenant_id = $1 AND model_id = $2;

-- name: DeleteIndexedFieldsByModelSlug :exec
DELETE FROM indexed_fields idx
WHERE idx.tenant_id = $1
  AND idx.model_id = (
    SELECT m.id FROM models m
    WHERE m.tenant_id = $1 AND m.kind_id = $2 AND m.slug = $3
  );

-- name: ListIndexedFieldsByModel :many
SELECT * FROM indexed_fields
WHERE tenant_id = $1 AND model_id = $2
ORDER BY field_key;
