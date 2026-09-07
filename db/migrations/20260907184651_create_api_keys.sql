-- migrate:up

CREATE TABLE api_keys (
    id           CHAR(20)    PRIMARY KEY,
    tenant_id    CHAR(20)    NOT NULL REFERENCES tenants(id),
    name         TEXT        NOT NULL,
    key_hash     TEXT        NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX api_keys_hash_active
    ON api_keys (key_hash)
    WHERE deleted_at IS NULL;

-- migrate:down

DROP TABLE api_keys;
