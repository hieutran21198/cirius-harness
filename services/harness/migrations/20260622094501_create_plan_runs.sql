-- +goose Up
-- Run/progress state for driving a council plan (ADR-0021). The Plan is the immutable spec; a
-- run records execution, so an approved plan is never rewritten. One live run per plan.
CREATE TABLE plan_runs (
    id         TEXT PRIMARY KEY NOT NULL,
    plan_id    TEXT NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    session_id TEXT REFERENCES sessions (id) ON DELETE SET NULL,
    status     TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE (plan_id)
);

CREATE INDEX idx_plan_runs_plan ON plan_runs (plan_id);

-- Per-task progress within a run, keyed by the plan-local task ref (T1). task_ref is the
-- business ref the wire uses, not an FK — ref integrity is enforced in the domain when the run
-- is seeded from the plan's refs.
CREATE TABLE plan_task_runs (
    id          TEXT PRIMARY KEY NOT NULL,
    plan_run_id TEXT NOT NULL REFERENCES plan_runs (id) ON DELETE CASCADE,
    task_ref    TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    summary     TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME NOT NULL,
    UNIQUE (plan_run_id, task_ref)
);

CREATE INDEX idx_plan_task_runs_run ON plan_task_runs (plan_run_id);

-- +goose Down
DROP TABLE plan_task_runs;
DROP TABLE plan_runs;
