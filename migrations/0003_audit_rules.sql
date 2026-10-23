-- OpenTagValidator audit rules.
--
-- Rules are user-authored, project-scoped governance checks evaluated against
-- every fresh page scan. They are intentionally data, not code.

BEGIN;

CREATE TABLE IF NOT EXISTS audit_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    vendor      text        NOT NULL DEFAULT '',
    enabled     boolean     NOT NULL DEFAULT true,
    url_pattern text        NOT NULL DEFAULT '',
    target      text        NOT NULL
                CHECK (target IN ('request','event','tag','cookie')),
    parameter   text        NOT NULL DEFAULT '',
    event_name  text        NOT NULL DEFAULT '',
    operator    text        NOT NULL
                CHECK (operator IN ('exists','not_exists','equals','not_equals','matches','not_matches')),
    expected    text        NOT NULL DEFAULT '',
    severity    text        NOT NULL DEFAULT 'medium'
                CHECK (severity IN ('info','low','medium','high','critical')),
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_rules_project_idx
    ON audit_rules(project_id, created_at);

COMMIT;
