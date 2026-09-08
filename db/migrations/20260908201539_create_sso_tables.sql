-- migrate:up

-- IDP identities linked to a tenant. One tenant can host multiple identities
-- (Google + GitHub for the same person, family sharing, etc.); a given
-- (provider, subject) pair belongs to exactly one tenant.
CREATE TABLE user_identities (
    id            CHAR(20)    PRIMARY KEY,
    tenant_id     CHAR(20)    NOT NULL REFERENCES tenants(id),
    provider      TEXT        NOT NULL,
    subject       TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    display_name  TEXT,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX user_identities_provider_subject_active
    ON user_identities (provider, subject) WHERE deleted_at IS NULL;

-- Short-lived state carried across the IDP roundtrip. When the user starts
-- an SSO flow we insert one row keyed by the state we send to the IDP; the
-- callback looks it up (with an expiry check) and clears it.
--
-- Some rows carry a pending OAuth 2.1 authorize request from a downstream
-- MCP client so that after the Google roundtrip we can complete the client
-- flow with the newly known identity. Others carry only a return_to path
-- for the browser web app case.
CREATE TABLE sso_login_intents (
    state                       TEXT        PRIMARY KEY,
    provider                    TEXT        NOT NULL,
    return_to                   TEXT,
    -- pending oauth authorize request (nullable when this is a web-only login)
    client_id                   CHAR(20),
    redirect_uri                TEXT,
    code_challenge              TEXT,
    code_challenge_method       TEXT,
    downstream_state            TEXT,
    scope                       TEXT,
    expires_at                  TIMESTAMPTZ NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX sso_login_intents_expiry ON sso_login_intents (expires_at);

-- migrate:down

DROP TABLE sso_login_intents;
DROP TABLE user_identities;
