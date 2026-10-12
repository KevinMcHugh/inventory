-- Search/sort model queries. There is one filtered+sorted query per
-- indexed_fields value type (string/numeric/int/bool/date) plus one
-- sort-only pair with no filter join, and each comes in an Asc/Desc variant
-- since ORDER BY direction cannot be bound as a parameter in static SQL.
--
-- Every query orders by all five typed columns of the sort join (sf) in
-- sequence. Because a given field_key is only ever written to one of those
-- columns (see internal/indexedfields), the other four are uniformly NULL
-- across the whole result set and act as no-op tiebreakers -- so one query
-- shape sorts correctly regardless of the sort field's actual type. A
-- final m.created_at DESC keeps ordering deterministic when the sort field
-- is absent or ties.

-- name: SortModelsByFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SortModelsByFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByStringFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::text IS NULL OR f.string_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::text IS NULL OR f.string_value != sqlc.narg('ne'))
  AND (sqlc.narg('contains')::text IS NULL OR f.string_value ILIKE '%' || sqlc.narg('contains') || '%')
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByStringFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::text IS NULL OR f.string_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::text IS NULL OR f.string_value != sqlc.narg('ne'))
  AND (sqlc.narg('contains')::text IS NULL OR f.string_value ILIKE '%' || sqlc.narg('contains') || '%')
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByNumericFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::float8 IS NULL OR f.numeric_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::float8 IS NULL OR f.numeric_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::float8 IS NULL OR f.numeric_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::float8 IS NULL OR f.numeric_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::float8 IS NULL OR f.numeric_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::float8 IS NULL OR f.numeric_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByNumericFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::float8 IS NULL OR f.numeric_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::float8 IS NULL OR f.numeric_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::float8 IS NULL OR f.numeric_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::float8 IS NULL OR f.numeric_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::float8 IS NULL OR f.numeric_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::float8 IS NULL OR f.numeric_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByIntFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::bigint IS NULL OR f.int_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::bigint IS NULL OR f.int_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::bigint IS NULL OR f.int_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::bigint IS NULL OR f.int_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::bigint IS NULL OR f.int_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::bigint IS NULL OR f.int_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByIntFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::bigint IS NULL OR f.int_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::bigint IS NULL OR f.int_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::bigint IS NULL OR f.int_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::bigint IS NULL OR f.int_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::bigint IS NULL OR f.int_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::bigint IS NULL OR f.int_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByBoolFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::bool IS NULL OR f.bool_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::bool IS NULL OR f.bool_value != sqlc.narg('ne'))
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByBoolFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::bool IS NULL OR f.bool_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::bool IS NULL OR f.bool_value != sqlc.narg('ne'))
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByDateFieldAsc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::timestamptz IS NULL OR f.date_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::timestamptz IS NULL OR f.date_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::timestamptz IS NULL OR f.date_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::timestamptz IS NULL OR f.date_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::timestamptz IS NULL OR f.date_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::timestamptz IS NULL OR f.date_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value ASC NULLS LAST,
  sf.numeric_value ASC NULLS LAST,
  sf.int_value ASC NULLS LAST,
  sf.date_value ASC NULLS LAST,
  sf.bool_value ASC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: SearchModelsByDateFieldDesc :many
SELECT m.id, m.tenant_id, m.kind_id, m.kind_version_id, m.slug, m.body, m.created_at, m.updated_at, m.deleted_at
FROM models m
JOIN indexed_fields f ON f.model_id = m.id AND f.field_key = @filter_field
LEFT JOIN indexed_fields sf ON sf.model_id = m.id AND sf.field_key = @sort_field
WHERE m.tenant_id = @tenant_id
  AND m.kind_id = @kind_id
  AND m.deleted_at IS NULL
  AND (sqlc.narg('eq')::timestamptz IS NULL OR f.date_value = sqlc.narg('eq'))
  AND (sqlc.narg('ne')::timestamptz IS NULL OR f.date_value != sqlc.narg('ne'))
  AND (sqlc.narg('lt')::timestamptz IS NULL OR f.date_value < sqlc.narg('lt'))
  AND (sqlc.narg('lte')::timestamptz IS NULL OR f.date_value <= sqlc.narg('lte'))
  AND (sqlc.narg('gt')::timestamptz IS NULL OR f.date_value > sqlc.narg('gt'))
  AND (sqlc.narg('gte')::timestamptz IS NULL OR f.date_value >= sqlc.narg('gte'))
ORDER BY
  sf.string_value DESC NULLS LAST,
  sf.numeric_value DESC NULLS LAST,
  sf.int_value DESC NULLS LAST,
  sf.date_value DESC NULLS LAST,
  sf.bool_value DESC NULLS LAST,
  m.created_at DESC
LIMIT @limit_count OFFSET @offset_count;
