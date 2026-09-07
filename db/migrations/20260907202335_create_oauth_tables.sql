-- migrate:up

CREATE TABLE oauth_clients (
    id                 CHAR(20)    PRIMARY KEY,
    client_secret_hash TEXT,
    redirect_uris      JSONB       NOT NULL,
    client_name        TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE TABLE oauth_codes (
    code_hash             TEXT        PRIMARY KEY,
    client_id             CHAR(20)    NOT NULL REFERENCES oauth_clients(id),
    tenant_id             CHAR(20)    NOT NULL REFERENCES tenants(id),
    redirect_uri          TEXT        NOT NULL,
    code_challenge        TEXT        NOT NULL,
    code_challenge_method TEXT        NOT NULL,
    scope                 TEXT,
    expires_at            TIMESTAMPTZ NOT NULL,
    consumed_at           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_tokens (
    id           CHAR(20)    PRIMARY KEY,
    token_hash   TEXT        NOT NULL UNIQUE,
    client_id    CHAR(20)    NOT NULL REFERENCES oauth_clients(id),
    tenant_id    CHAR(20)    NOT NULL REFERENCES tenants(id),
    scope        TEXT,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX oauth_tokens_hash_active
    ON oauth_tokens (token_hash) WHERE revoked_at IS NULL;

-- migrate:down

DROP TABLE oauth_tokens;
DROP TABLE oauth_codes;
DROP TABLE oauth_clients;
