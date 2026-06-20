-- +goose Up
CREATE TABLE models (
    id       TEXT PRIMARY KEY NOT NULL,
    provider TEXT    NOT NULL,
    slug     TEXT    NOT NULL,
    enabled  INTEGER NOT NULL DEFAULT 1,
    UNIQUE (provider, slug)
);

-- An agent is a pure role: identity + archetype, no model. Which model plays the role is
-- bound per session (see session_agents.model_id).
CREATE TABLE agents (
    id             TEXT PRIMARY KEY NOT NULL,
    name           TEXT    NOT NULL UNIQUE,
    archetype      TEXT    NOT NULL,
    responsibility TEXT    NOT NULL DEFAULT '',
    description    TEXT    NOT NULL DEFAULT '',
    source         TEXT    NOT NULL,
    enabled        INTEGER NOT NULL DEFAULT 1
);

-- The capability catalog.
CREATE TABLE tools (
    id          TEXT PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT ''
);

-- Pure junction: which tools an agent is granted.
CREATE TABLE agent_tools (
    agent_id TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
    tool_id  TEXT NOT NULL REFERENCES tools (id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, tool_id)
);

CREATE TABLE projects (
    id          TEXT PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL UNIQUE,
    root_path   TEXT NOT NULL UNIQUE,
    kind        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

-- An execution environment, sibling to a worktree.
CREATE TABLE containers (
    id         TEXT PRIMARY KEY NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    image      TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL
);

CREATE INDEX idx_containers_project ON containers (project_id);

CREATE TABLE worktrees (
    id         TEXT PRIMARY KEY NOT NULL,
    path       TEXT NOT NULL UNIQUE,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    branch     TEXT NOT NULL,
    status     TEXT NOT NULL
);

CREATE INDEX idx_worktrees_project ON worktrees (project_id);

-- A session is scoped to a project and runs in a polymorphic environment: env_id points at a
-- container or a worktree (or is empty when env_type = 'unset'). env_id has no FK because the
-- target table varies; the domain validates it.
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY NOT NULL,
    project_id  TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    env_type    TEXT     NOT NULL DEFAULT 'unset',
    env_id      TEXT     NOT NULL DEFAULT '',
    title       TEXT     NOT NULL DEFAULT '',
    status      TEXT     NOT NULL,
    created_at  DATETIME NOT NULL,
    started_at  DATETIME,
    ended_at    DATETIME
);

CREATE INDEX idx_sessions_project ON sessions (project_id);

-- Carries model_id (the model the agent ran with), so it takes a surrogate id rather than a
-- composite junction PK.
CREATE TABLE session_agents (
    id         TEXT PRIMARY KEY NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    agent_id   TEXT NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
    model_id   TEXT          REFERENCES models (id),
    UNIQUE (session_id, agent_id)
);

CREATE INDEX idx_session_agents_session ON session_agents (session_id);

-- +goose Down
DROP TABLE session_agents;
DROP TABLE sessions;
DROP TABLE worktrees;
DROP TABLE containers;
DROP TABLE projects;
DROP TABLE agent_tools;
DROP TABLE tools;
DROP TABLE agents;
DROP TABLE models;
