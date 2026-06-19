-- +goose Up
CREATE TABLE agents (
    name           TEXT PRIMARY KEY,
    model          TEXT    NOT NULL DEFAULT '',
    responsibility TEXT    NOT NULL DEFAULT '',
    archetype      TEXT    NOT NULL,
    description    TEXT    NOT NULL DEFAULT '',
    source         TEXT    NOT NULL,
    enabled        INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE agent_tools (
    agent_name TEXT NOT NULL REFERENCES agents (name) ON DELETE CASCADE,
    tool       TEXT NOT NULL,
    PRIMARY KEY (agent_name, tool)
);

CREATE TABLE agent_fallbacks (
    agent_name TEXT    NOT NULL REFERENCES agents (name) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    model      TEXT    NOT NULL,
    PRIMARY KEY (agent_name, position)
);

CREATE TABLE projects (
    name        TEXT PRIMARY KEY,
    root_path   TEXT NOT NULL UNIQUE,
    kind        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE worktrees (
    path    TEXT PRIMARY KEY,
    project TEXT NOT NULL REFERENCES projects (name) ON DELETE CASCADE,
    branch  TEXT NOT NULL,
    status  TEXT NOT NULL
);

CREATE INDEX idx_worktrees_project ON worktrees (project);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    worktree   TEXT NOT NULL REFERENCES worktrees (path) ON DELETE CASCADE,
    title      TEXT     NOT NULL DEFAULT '',
    status     TEXT     NOT NULL,
    created_at DATETIME NOT NULL,
    started_at DATETIME,
    ended_at   DATETIME
);

CREATE INDEX idx_sessions_worktree ON sessions (worktree);

CREATE TABLE session_agents (
    session_id TEXT     NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    agent_name TEXT     NOT NULL REFERENCES agents (name) ON DELETE CASCADE,
    role       TEXT     NOT NULL DEFAULT '',
    active     INTEGER  NOT NULL DEFAULT 1,
    joined_at  DATETIME NOT NULL,
    PRIMARY KEY (session_id, agent_name)
);

-- +goose Down
DROP TABLE session_agents;
DROP TABLE sessions;
DROP TABLE worktrees;
DROP TABLE projects;
DROP TABLE agent_fallbacks;
DROP TABLE agent_tools;
DROP TABLE agents;
