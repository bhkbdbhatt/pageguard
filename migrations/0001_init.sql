-- Pageguard initial schema.
--
-- Conventions:
--   * UUID primary keys (gen_random_uuid from pgcrypto).
--   * Booleans/enums modelled as text + CHECK for portable migrations.
--   * Raw cookie values are NEVER stored; only a monotonic value hash.
--   * JSONB is used for open-ended structures (scan config, evidence, stats).
--
-- Apply with: psql "$DATABASE_URL" -f migrations/0001_init.sql
-- Repeatable migrations (0002+) should be added rather than editing this file.

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------- projects --
CREATE TABLE IF NOT EXISTS projects (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    slug       text        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------------- scans --
CREATE TABLE IF NOT EXISTS scans (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         text        NOT NULL,
    start_urls   text[]      NOT NULL,
    config       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    rule_set_ids text[]      NOT NULL DEFAULT '{}',
    enabled      boolean     NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS scans_project_idx ON scans(project_id);

-- --------------------------------------------------------------- scan_runs --
CREATE TABLE IF NOT EXISTS scan_runs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id     uuid        NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status      text        NOT NULL DEFAULT 'queued'
                CHECK (status IN ('queued','running','succeeded','failed','canceled')),
    stats       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    error       text,
    started_at  timestamptz,
    finished_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS scan_runs_scan_idx ON scan_runs(scan_id, created_at DESC);
CREATE INDEX IF NOT EXISTS scan_runs_active_idx ON scan_runs(status) WHERE status IN ('queued','running');

-- --------------------------------------------------------------- run_pages --
CREATE TABLE IF NOT EXISTS run_pages (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id       uuid        NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    url          text        NOT NULL,
    final_url    text,
    status       int,
    depth        int         NOT NULL DEFAULT 0,
    rendered     boolean     NOT NULL DEFAULT false,
    title        text,
    content_type text,
    duration_ms  bigint,
    error        text,
    scanned_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (run_id, url)
);
CREATE INDEX IF NOT EXISTS run_pages_run_idx ON run_pages(run_id);
CREATE INDEX IF NOT EXISTS run_pages_status_idx ON run_pages(run_id, status);

-- ----------------------------------------------------------- page_requests --
CREATE TABLE IF NOT EXISTS page_requests (
    id               bigserial PRIMARY KEY,
    page_id          uuid    NOT NULL REFERENCES run_pages(id) ON DELETE CASCADE,
    run_id           uuid    NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    method           text,
    url              text    NOT NULL,
    host             text,
    status           int,
    resource_type    text,
    initiator        text,
    request_headers  jsonb,
    response_headers jsonb,
    started_at_ms    bigint,
    duration_ms      bigint,
    size_bytes       bigint,
    third_party      boolean NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS page_requests_page_idx ON page_requests(page_id);
CREATE INDEX IF NOT EXISTS page_requests_run_host_idx ON page_requests(run_id, host);

-- ------------------------------------------------------------ page_cookies --
CREATE TABLE IF NOT EXISTS page_cookies (
    id           bigserial PRIMARY KEY,
    page_id      uuid    NOT NULL REFERENCES run_pages(id) ON DELETE CASCADE,
    run_id       uuid    NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    name         text    NOT NULL,
    domain       text,
    path         text,
    value_hash   text,
    secure       boolean NOT NULL DEFAULT false,
    http_only    boolean NOT NULL DEFAULT false,
    same_site    text,
    expires_unix bigint,
    is_session   boolean NOT NULL DEFAULT false,
    first_party  boolean NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS page_cookies_page_idx ON page_cookies(page_id);
CREATE INDEX IF NOT EXISTS page_cookies_name_idx ON page_cookies(run_id, name);

-- --------------------------------------------------------------- page_tags --
CREATE TABLE IF NOT EXISTS page_tags (
    id         bigserial PRIMARY KEY,
    page_id    uuid    NOT NULL REFERENCES run_pages(id) ON DELETE CASCADE,
    run_id     uuid    NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    tag_id     text    NOT NULL,
    vendor     text,
    name       text,
    category   text,
    version    text,
    evidence   text,
    confidence real,
    UNIQUE (page_id, tag_id)
);
CREATE INDEX IF NOT EXISTS page_tags_run_idx ON page_tags(run_id, tag_id);

-- ---------------------------------------------------------------- findings --
CREATE TABLE IF NOT EXISTS findings (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      uuid        NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    page_id     uuid        REFERENCES run_pages(id) ON DELETE SET NULL,
    page_url    text,
    plugin_id   text        NOT NULL,
    rule_id     text        NOT NULL,
    severity    text        NOT NULL
                CHECK (severity IN ('info','low','medium','high','critical')),
    title       text        NOT NULL,
    description text,
    evidence    jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS findings_run_idx ON findings(run_id);
CREATE INDEX IF NOT EXISTS findings_run_sev_idx ON findings(run_id, severity);

-- -------------------------------------------------------------- run_events --
CREATE TABLE IF NOT EXISTS run_events (
    id      bigserial PRIMARY KEY,
    run_id  uuid        NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    type    text        NOT NULL,
    at      timestamptz NOT NULL DEFAULT now(),
    payload jsonb
);
CREATE INDEX IF NOT EXISTS run_events_run_idx ON run_events(run_id, id);

-- ---------------------------------------------------------------- journeys --
CREATE TABLE IF NOT EXISTS journeys (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       text        NOT NULL,
    steps      jsonb       NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS journeys_project_idx ON journeys(project_id);

-- ------------------------------------------------------------- alert_rules --
CREATE TABLE IF NOT EXISTS alert_rules (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       text        NOT NULL,
    enabled    boolean     NOT NULL DEFAULT true,
    condition  jsonb       NOT NULL DEFAULT '{}'::jsonb,
    channels   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS alert_rules_project_idx ON alert_rules(project_id);

-- ------------------------------------------------------------- alert_events --
CREATE TABLE IF NOT EXISTS alert_events (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id    uuid        NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    run_id     uuid        REFERENCES scan_runs(id) ON DELETE SET NULL,
    message    text        NOT NULL,
    payload    jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS alert_events_rule_idx ON alert_events(rule_id, created_at DESC);

-- --------------------------------------------------- plugin_registrations --
CREATE TABLE IF NOT EXISTS plugin_registrations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id  text        NOT NULL UNIQUE,
    name       text        NOT NULL,
    version    text        NOT NULL,
    kind       text        NOT NULL,
    runtime    text        NOT NULL,
    manifest   jsonb       NOT NULL DEFAULT '{}'::jsonb,
    enabled    boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

COMMIT;
