-- migrate:up

CREATE TABLE tenants (
    id         CHAR(20)    PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE kinds (
    id          CHAR(20)    PRIMARY KEY,
    tenant_id   CHAR(20)    NOT NULL REFERENCES tenants(id),
    name        TEXT        NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE kind_versions (
    id         CHAR(20)    PRIMARY KEY,
    kind_id    CHAR(20)    NOT NULL REFERENCES kinds(id),
    schema     JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE models (
    id              CHAR(20)    PRIMARY KEY,
    tenant_id       CHAR(20)    NOT NULL REFERENCES tenants(id),
    kind_id         CHAR(20)    NOT NULL REFERENCES kinds(id),
    kind_version_id CHAR(20)    NOT NULL REFERENCES kind_versions(id),
    slug            TEXT        NOT NULL,
    body            JSONB       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX models_tenant_kind_slug_active
    ON models (tenant_id, kind_id, slug)
    WHERE deleted_at IS NULL;

-- migrate:down

DROP TABLE models;
DROP TABLE kind_versions;
DROP TABLE kinds;
DROP TABLE tenants;
