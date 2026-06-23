-- +goose Up
-- Structured task results and council's post-execution decision (ADR-0023). A driven worker emits
-- a schema'd report envelope; the harness validates and stores it here (with the raw output kept
-- for audit). Council's decision stage consumes these normalized envelopes and records a verdict.

-- One structured report per driven task, keyed by the run and the plan-local task ref (T1). The
-- envelope is the validated TaskReportEnvelope JSON; raw is the worker's full output, for
-- audit/debug. A retried task UPSERTs in place on (plan_run_id, task_ref).
CREATE TABLE task_reports (
    id          TEXT PRIMARY KEY NOT NULL,
    plan_run_id TEXT NOT NULL REFERENCES plan_runs (id) ON DELETE CASCADE,
    task_ref    TEXT NOT NULL,
    agent       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT '',
    envelope    TEXT NOT NULL,
    raw         TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL,
    UNIQUE (plan_run_id, task_ref)
);

CREATE INDEX idx_task_reports_run ON task_reports (plan_run_id);

-- Council's verdict over a run, append-only: each iteration of a drive records its own decision;
-- the latest by created_at is the current verdict. decision is the validated CouncilDecision JSON.
CREATE TABLE council_decisions (
    id          TEXT PRIMARY KEY NOT NULL,
    plan_run_id TEXT NOT NULL REFERENCES plan_runs (id) ON DELETE CASCADE,
    decision    TEXT NOT NULL,
    created_at  DATETIME NOT NULL
);

CREATE INDEX idx_council_decisions_run ON council_decisions (plan_run_id);

-- +goose Down
DROP TABLE council_decisions;
DROP TABLE task_reports;
