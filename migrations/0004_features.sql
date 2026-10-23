-- Backend persistence for inventory schedules, alerting, journeys, privacy
-- batches/reports. Every statement is idempotent because the migrator
-- re-applies all migration files on each startup.

BEGIN;

-- --------------------------------------------------------------- schedules --
CREATE TABLE IF NOT EXISTS schedules (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    scan_id       uuid        REFERENCES scans(id) ON DELETE SET NULL,
    name          text        NOT NULL,
    url           text        NOT NULL,
    cron          text        NOT NULL,
    timezone      text        NOT NULL DEFAULT 'UTC',
    enabled       boolean     NOT NULL DEFAULT true,
    max_pages     int         NOT NULL DEFAULT 100,
    max_depth     int         NOT NULL DEFAULT 3,
    consent       text        NOT NULL DEFAULT 'pre_consent',
    owner         text        NOT NULL DEFAULT '',
    last_run_id   uuid        REFERENCES scan_runs(id) ON DELETE SET NULL,
    last_status   text,
    last_run_at   timestamptz,
    next_run_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS schedules_project_idx ON schedules(project_id, created_at);
CREATE INDEX IF NOT EXISTS schedules_due_idx ON schedules(next_run_at) WHERE enabled;

-- ------------------------------------------------------------------ alerts --
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS severity text NOT NULL DEFAULT 'medium';
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS condition_text text NOT NULL DEFAULT '';
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS channel_ids uuid[] NOT NULL DEFAULT '{}';
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS last_triggered_at timestamptz;
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS trigger_count int NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS alert_channels (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name             text        NOT NULL,
    kind             text        NOT NULL CHECK (kind IN ('slack','webhook','email','pagerduty')),
    target           text        NOT NULL DEFAULT '',
    secret_ref       text        NOT NULL DEFAULT '',
    connected        boolean     NOT NULL DEFAULT true,
    last_delivery_at timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS alert_channels_project_idx ON alert_channels(project_id, created_at);

CREATE TABLE IF NOT EXISTS alert_incidents (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rule_id         uuid        REFERENCES alert_rules(id) ON DELETE SET NULL,
    run_id          uuid        REFERENCES scan_runs(id) ON DELETE SET NULL,
    severity        text        NOT NULL DEFAULT 'medium'
                    CHECK (severity IN ('info','low','medium','high','critical')),
    title           text        NOT NULL,
    message         text        NOT NULL DEFAULT '',
    rule_name       text        NOT NULL DEFAULT '',
    channel         text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','acknowledged','resolved')),
    acknowledged_at timestamptz,
    acknowledged_by text        NOT NULL DEFAULT '',
    resolved_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS alert_incidents_project_idx ON alert_incidents(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS alert_incidents_rule_idx ON alert_incidents(rule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS alert_incidents_status_idx ON alert_incidents(status, created_at DESC);

CREATE TABLE IF NOT EXISTS alert_deliveries (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id  uuid        NOT NULL REFERENCES alert_incidents(id) ON DELETE CASCADE,
    channel_id   uuid        NOT NULL REFERENCES alert_channels(id) ON DELETE CASCADE,
    status       text        NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','delivered','failed')),
    attempt      int         NOT NULL DEFAULT 1,
    error        text        NOT NULL DEFAULT '',
    attempted_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz
);
CREATE INDEX IF NOT EXISTS alert_deliveries_incident_idx ON alert_deliveries(incident_id, attempted_at DESC);

-- ---------------------------------------------------------------- journeys --
ALTER TABLE journeys ADD COLUMN IF NOT EXISTS definition jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE journeys ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';
ALTER TABLE journeys ADD COLUMN IF NOT EXISTS owner text NOT NULL DEFAULT '';
ALTER TABLE journeys ADD COLUMN IF NOT EXISTS base_url text NOT NULL DEFAULT '';
ALTER TABLE journeys ADD COLUMN IF NOT EXISTS enabled boolean NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS journey_runs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    journey_id    uuid        NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    project_id    uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status        text        NOT NULL DEFAULT 'running'
                  CHECK (status IN ('running','passed','failed','canceled')),
    steps_passed  int         NOT NULL DEFAULT 0,
    steps_total   int         NOT NULL DEFAULT 0,
    duration_ms   bigint      NOT NULL DEFAULT 0,
    error         text        NOT NULL DEFAULT '',
    report        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS journey_runs_journey_idx ON journey_runs(journey_id, created_at DESC);
CREATE INDEX IF NOT EXISTS journey_runs_project_idx ON journey_runs(project_id, created_at DESC);

-- ----------------------------------------------------------------- privacy --
CREATE TABLE IF NOT EXISTS privacy_profiles (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            text        NOT NULL,
    region          text        NOT NULL DEFAULT '',
    regulation_ids  text[]      NOT NULL DEFAULT '{}',
    config          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    active          boolean     NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS privacy_profiles_project_idx ON privacy_profiles(project_id, created_at);

CREATE TABLE IF NOT EXISTS privacy_batches (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    profile_id       uuid        REFERENCES privacy_profiles(id) ON DELETE SET NULL,
    start_url        text        NOT NULL,
    status           text        NOT NULL DEFAULT 'queued'
                     CHECK (status IN ('queued','running','succeeded','failed','canceled')),
    total_pages      int         NOT NULL DEFAULT 1,
    completed_pages  int         NOT NULL DEFAULT 0,
    failed_pages     int         NOT NULL DEFAULT 0,
    concurrency      int         NOT NULL DEFAULT 2,
    error            text        NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    started_at       timestamptz,
    finished_at      timestamptz
);
CREATE INDEX IF NOT EXISTS privacy_batches_project_idx ON privacy_batches(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS privacy_batches_active_idx ON privacy_batches(status) WHERE status IN ('queued','running');

CREATE TABLE IF NOT EXISTS privacy_evaluations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     uuid        NOT NULL REFERENCES privacy_batches(id) ON DELETE CASCADE,
    project_id   uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    profile_id   uuid        REFERENCES privacy_profiles(id) ON DELETE SET NULL,
    url          text        NOT NULL,
    status       text        NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued','running','succeeded','failed','canceled')),
    report       jsonb       NOT NULL DEFAULT '{}'::jsonb,
    error        text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    finished_at  timestamptz,
    UNIQUE (batch_id, url)
);
CREATE INDEX IF NOT EXISTS privacy_evaluations_batch_idx ON privacy_evaluations(batch_id, created_at);
CREATE INDEX IF NOT EXISTS privacy_evaluations_open_idx ON privacy_evaluations(batch_id) WHERE status = 'queued';

-- ----------------------------------------------------------------- reports --
CREATE TABLE IF NOT EXISTS reports (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    template_id  text        NOT NULL,
    name         text        NOT NULL,
    scope_type   text        NOT NULL DEFAULT 'project'
                 CHECK (scope_type IN ('project','run','schedule','journey','privacy_batch','inventory')),
    scope_id     text        NOT NULL DEFAULT '',
    scope_label  text        NOT NULL DEFAULT '',
    format       text        NOT NULL DEFAULT 'json'
                 CHECK (format IN ('json','csv','html')),
    status       text        NOT NULL DEFAULT 'generating'
                 CHECK (status IN ('ready','generating','failed')),
    owner        text        NOT NULL DEFAULT '',
    content_type text        NOT NULL DEFAULT 'application/json; charset=utf-8',
    content      bytea,
    size_bytes   bigint      NOT NULL DEFAULT 0,
    error        text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS reports_project_idx ON reports(project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS report_schedules (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    template_id    text        NOT NULL,
    name           text        NOT NULL,
    cadence        text        NOT NULL DEFAULT '',
    cron           text        NOT NULL,
    timezone       text        NOT NULL DEFAULT 'UTC',
    recipients     text[]      NOT NULL DEFAULT '{}',
    scope_type     text        NOT NULL DEFAULT 'project',
    scope_id       text        NOT NULL DEFAULT '',
    scope_label    text        NOT NULL DEFAULT '',
    format         text        NOT NULL DEFAULT 'html'
                   CHECK (format IN ('json','csv','html')),
    enabled        boolean     NOT NULL DEFAULT true,
    next_run_at    timestamptz,
    last_run_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS report_schedules_project_idx ON report_schedules(project_id, created_at);
CREATE INDEX IF NOT EXISTS report_schedules_due_idx ON report_schedules(next_run_at) WHERE enabled;

COMMIT;
