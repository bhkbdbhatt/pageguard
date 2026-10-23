-- Pageguard scan results (per-page checkpoints).
--
-- Each row is a fully-formed domain.ScanResult as JSONB, keyed by (run_id, url).
-- Writing a row is the checkpoint that makes a run resumable and idempotent:
-- on resume, the orchestrator loads the existing result instead of re-scanning
-- the page.

BEGIN;

CREATE TABLE IF NOT EXISTS scan_results (
    run_id     uuid        NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    url        text        NOT NULL,
    result     jsonb       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, url)
);

CREATE INDEX IF NOT EXISTS scan_results_run_idx ON scan_results(run_id, created_at);

COMMIT;
