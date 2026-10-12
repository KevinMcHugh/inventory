-- migrate:up

-- indexed_fields is a derived search index over model bodies: one row per
-- (model, indexed field) with the value pulled out of the JSONB body into a
-- typed column so Postgres can filter/sort on it directly. Which fields get
-- indexed is declared on kind_versions.schema (Field.indexed); the app layer
-- keeps this table in sync on every model write. Being purely derived, it
-- does not carry deleted_at like other tables -- rows are hard-deleted and
-- rebuilt whenever the owning model is written or removed.
CREATE TABLE indexed_fields (
    id            CHAR(20)         PRIMARY KEY,
    tenant_id     CHAR(20)         NOT NULL REFERENCES tenants(id),
    kind_id       CHAR(20)         NOT NULL REFERENCES kinds(id),
    model_id      CHAR(20)         NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    field_key     TEXT             NOT NULL,
    string_value  TEXT,
    numeric_value DOUBLE PRECISION,
    int_value     BIGINT,
    bool_value    BOOLEAN,
    date_value    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    UNIQUE (model_id, field_key)
);

CREATE INDEX indexed_fields_string_lookup
    ON indexed_fields (tenant_id, kind_id, field_key, string_value);

CREATE INDEX indexed_fields_numeric_lookup
    ON indexed_fields (tenant_id, kind_id, field_key, numeric_value);

CREATE INDEX indexed_fields_int_lookup
    ON indexed_fields (tenant_id, kind_id, field_key, int_value);

CREATE INDEX indexed_fields_bool_lookup
    ON indexed_fields (tenant_id, kind_id, field_key, bool_value);

CREATE INDEX indexed_fields_date_lookup
    ON indexed_fields (tenant_id, kind_id, field_key, date_value);

-- migrate:down

DROP TABLE indexed_fields;
