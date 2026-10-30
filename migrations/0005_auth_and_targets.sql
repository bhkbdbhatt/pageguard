-- Governance v2: API keys for authentication/tenancy, extended rule targets,
-- alert dedup interval, journey run screenshots.

BEGIN;

-- B1: API keys. The plaintext secret ("pg_<hex>") is shown once; only the
-- SHA-256 digest is stored. project_id NULL + admin role = unrestricted.
CREATE TABLE IF NOT EXISTS pageguard_api_keys (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name         text        NOT NULL,
    project_id   uuid        REFERENCES projects(id) ON DELETE CASCADE,
    prefix       text        NOT NULL,
    key_hash     text        NOT NULL UNIQUE,
    roles        text[]      NOT NULL DEFAULT '{reader}',
    enabled      boolean     NOT NULL DEFAULT true,
    expires_at   timestamptz,
    last_used_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS api_keys_by_project
    ON pageguard_api_keys(project_id, created_at);

-- B5: rules gained beacon/network/page targets.
ALTER TABLE audit_rules DROP CONSTRAINT IF EXISTS audit_rules_target_check;
ALTER TABLE audit_rules
    ADD CONSTRAINT audit_rules_target_check
    CHECK (target IN ('request','event','tag','cookie','beacon','network','page'));

-- B2: alert dedup window (0 = every qualifying run triggers).
ALTER TABLE alert_rules
    ADD COLUMN IF NOT EXISTS min_interval_minutes integer NOT NULL DEFAULT 0;

-- B7: journey runs keep failure screenshots for the UI.
ALTER TABLE journey_runs
    ADD COLUMN IF NOT EXISTS screenshot_uri text NOT NULL DEFAULT '';

COMMIT;
