-- migrate:up

-- Single-use invites minted from the CLI. An invite is the way a new person
-- can create the FIRST identity for a tenant on this server; a Google
-- callback with an unknown (provider, subject) is rejected unless a valid
-- invite is presented.
--
-- tenant_id is nullable: null means "on redeem, create a fresh tenant for
-- the caller", non-null means "link the caller into this existing tenant"
-- (useful for adding a family member's Google to an existing wardrobe).
CREATE TABLE invites (
    id                    CHAR(20)    PRIMARY KEY,
    code_hash             TEXT        NOT NULL UNIQUE,
    tenant_id             CHAR(20)    REFERENCES tenants(id),
    created_by_identity_id CHAR(20),
    expires_at            TIMESTAMPTZ,
    used_at               TIMESTAMPTZ,
    used_by_identity_id   CHAR(20)    REFERENCES user_identities(id),
    note                  TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ
);

CREATE INDEX invites_active
    ON invites (created_at)
    WHERE used_at IS NULL AND deleted_at IS NULL;

-- The redeem code travels through the Google login roundtrip inside the
-- login intent so we can consume it after Google callback.
ALTER TABLE sso_login_intents ADD COLUMN invite_code TEXT;

-- migrate:down

ALTER TABLE sso_login_intents DROP COLUMN invite_code;
DROP TABLE invites;
