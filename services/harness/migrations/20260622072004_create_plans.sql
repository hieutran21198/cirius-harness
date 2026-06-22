-- +goose Up
-- A council orchestration plan, persisted after a human approves it (ADR-0019). The request
-- analysis (intent/goal) and the small leaf structures (scope, assumptions, report) ride as
-- JSON on the plan row; the task DAG, risks, approvals, and waves are fully relational children.
-- session_id is a nullable FK: a plan is attached to the session it was produced in, but
-- survives (SET NULL) if that session is later removed.
CREATE TABLE plans (
    id          TEXT PRIMARY KEY NOT NULL,
    session_id  TEXT          REFERENCES sessions (id) ON DELETE SET NULL,
    agent       TEXT     NOT NULL,
    intent      TEXT     NOT NULL,
    goal        TEXT     NOT NULL DEFAULT '',
    status      TEXT     NOT NULL,
    created_at  DATETIME NOT NULL,
    scope       TEXT     NOT NULL DEFAULT '{}',
    assumptions TEXT     NOT NULL DEFAULT '[]',
    report      TEXT     NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plans_session ON plans (session_id);

-- One node of the plan's task DAG. ref is the plan-local id ("T1"); depends_on / dod / inputs
-- are JSON string arrays.
CREATE TABLE plan_tasks (
    id              TEXT PRIMARY KEY NOT NULL,
    plan_id         TEXT NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    ref             TEXT NOT NULL,
    category        TEXT NOT NULL DEFAULT '',
    assignee_agent  TEXT NOT NULL DEFAULT '',
    assignee_lens   TEXT NOT NULL DEFAULT '',
    objective       TEXT NOT NULL DEFAULT '',
    expected_output TEXT NOT NULL DEFAULT '',
    gate            TEXT NOT NULL DEFAULT '',
    risk_level      TEXT NOT NULL DEFAULT '',
    inputs          TEXT NOT NULL DEFAULT '[]',
    depends_on      TEXT NOT NULL DEFAULT '[]',
    dod             TEXT NOT NULL DEFAULT '[]',
    UNIQUE (plan_id, ref)
);

CREATE INDEX idx_plan_tasks_plan ON plan_tasks (plan_id);

CREATE TABLE plan_risks (
    id          TEXT PRIMARY KEY NOT NULL,
    plan_id     TEXT NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    level       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_plan_risks_plan ON plan_risks (plan_id);

CREATE TABLE plan_approvals (
    id              TEXT PRIMARY KEY NOT NULL,
    plan_id         TEXT NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    type            TEXT NOT NULL DEFAULT '',
    required_before TEXT NOT NULL DEFAULT '',
    reason          TEXT NOT NULL DEFAULT '',
    question        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_plan_approvals_plan ON plan_approvals (plan_id);

CREATE TABLE plan_waves (
    id          TEXT PRIMARY KEY NOT NULL,
    plan_id     TEXT    NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    wave_number INTEGER NOT NULL,
    UNIQUE (plan_id, wave_number)
);

CREATE INDEX idx_plan_waves_plan ON plan_waves (plan_id);

-- The wave→task membership join: which tasks belong to a wave.
CREATE TABLE plan_wave_tasks (
    plan_wave_id TEXT NOT NULL REFERENCES plan_waves (id) ON DELETE CASCADE,
    plan_task_id TEXT NOT NULL REFERENCES plan_tasks (id) ON DELETE CASCADE,
    PRIMARY KEY (plan_wave_id, plan_task_id)
);

-- +goose Down
DROP TABLE plan_wave_tasks;
DROP TABLE plan_waves;
DROP TABLE plan_approvals;
DROP TABLE plan_risks;
DROP TABLE plan_tasks;
DROP TABLE plans;
