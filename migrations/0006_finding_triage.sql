-- Finding triage: status, owner and note for the audit -> assign -> fix loop.

BEGIN;

ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'open';
ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS owner text NOT NULL DEFAULT '';
ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS note text NOT NULL DEFAULT '';
ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE findings DROP CONSTRAINT IF EXISTS findings_status_check;
ALTER TABLE findings
    ADD CONSTRAINT findings_status_check
    CHECK (status IN ('open','acknowledged','in_progress','resolved','ignored'));

CREATE INDEX IF NOT EXISTS findings_run_status_idx ON findings(run_id, status);

COMMIT;
