# Harness data model

- Status: Implemented
- Owner: hieu
- Reviewers: -
- Related ADRs: [ADR-0001](../adr/0001-harness-layout.md), [ADR-0002](../adr/0002-persistence-and-migrations.md), [ADR-0003](../adr/0003-authorization-casbin-abac.md), [ADR-0004](../adr/0004-ports-and-adapters-topology.md), [ADR-0005](../adr/0005-surrogate-uuid-v7-keys.md), [ADR-0006](../adr/0006-model-catalog-and-agent-profiles.md) (superseded), [ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)

## Problem

Module 1 (`services/harness`) needs a persisted model for two things: the **declarative
agent team** (who can work) and the **runtime orchestration state** (work actually
happening). This spec is the reference for those domains, their schema, and the shared
packages that persist them.

## Goals / non-goals

- **Goals**: document the aggregates, their relationships, the SQLite schema, and the
  authorization model — as built.
- **Non-goals**: the application use cases, the client/MCP/events adapters, and the seed
  migration (see Open questions). Model *choices* live in
  [PDR-0001](../pdr/0001-default-team-model-assignments.md), not here.

## Design

### Two domains (hexagonal; pure domain, ports in `internal/port/outbound`)

**agent** — the declarative team, as **roles** bound to models at session time
([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)):

- `Agent` aggregate (ID, Name, Archetype, Responsibility, Description, Source, Enabled, ToolIDs)
  — a pure **role**; it carries **no model** and no fallbacks. `ToolIDs` are grants into the
  tool catalog (persisted via `agent_tools`). Typed-string enums `Archetype`
  (communicator | principle-driven | utility-runner | none) and `Source` (system | user).
  Repository port `outbound.Agents`.
- `Tool` aggregate (ID, Name, Description) — the capability **catalog**
  (read | grep | glob | list | edit | bash | webfetch | websearch), `tool.Tool`
  (`internal/domain/tool`). Repository port `outbound.Tools`.
- `Model` aggregate (ID, Provider, Slug, Enabled) — the first-class catalog of available
  provider/model-ids (`Slug` is the provider's model name), `model.Model`
  (`internal/domain/model`). Repository port `outbound.Models`. Which model an agent uses is
  bound **per session** (see `Member.ModelID`), not stored on the agent.

Permissions are **not** here — authorization is Casbin
([ADR-0003](../adr/0003-authorization-casbin-abac.md)), with `outbound.Authorizer` returning
`authz.Decision` (allow | ask | deny) for an `authz.Action`; the principal stays the agent
**name**.

**orchestration** — the runtime. A session is scoped to a project and runs in a polymorphic
environment (a container or a worktree, or none yet). See the [ERD](#schema-sqlite) below for
the full relationship picture.

Every aggregate has a **UUID v7 surrogate PK**; the natural keys below are `UNIQUE`
attributes, and references travel by UUID id ([ADR-0005](../adr/0005-surrogate-uuid-v7-keys.md)).

- `Project` (ID, Name UNIQUE, RootPath UNIQUE, Kind {single | monorepo}, Description) — repo `Projects`.
- `Worktree` (ID, Path UNIQUE, ProjectID, Branch, Status {active | stale}) — repo `Worktrees`.
- `Container` (ID, ProjectID, Image, Status) — an execution environment, sibling to worktree;
  repo `Containers`.
- `Session` (ID, ProjectID, EnvType {container | worktree | unset}, EnvID, Title,
  Status {pending | running | completed | failed | cancelled}, timestamps, Members) — repo
  `Sessions`. `EnvID` is a **polymorphic** reference (the container/worktree id, or empty when
  unset); it has no FK and is validated in the domain.
- `Member` (ID, AgentID, ModelID) — the agent↔session join (`session_agents`); `ModelID` is
  the model that agent ran with (empty for model-less `prayer`).

### Schema (SQLite)

```mermaid
erDiagram
    models {
        TEXT    id       PK
        TEXT    provider
        TEXT    slug         "provider model name"
        INTEGER enabled
    }
    agents {
        TEXT    id    PK
        TEXT    name  UK
        TEXT    archetype
        TEXT    responsibility
        TEXT    description
        TEXT    source
        INTEGER enabled
    }
    tools {
        TEXT id   PK
        TEXT name UK
        TEXT description
    }
    agent_tools {
        TEXT agent_id PK,FK
        TEXT tool_id  PK,FK
    }
    projects {
        TEXT id        PK
        TEXT name      UK
        TEXT root_path UK
        TEXT kind
        TEXT description
    }
    containers {
        TEXT id         PK
        TEXT project_id FK
        TEXT image
        TEXT status
    }
    worktrees {
        TEXT id         PK
        TEXT path       UK
        TEXT project_id FK
        TEXT branch
        TEXT status
    }
    sessions {
        TEXT     id          PK
        TEXT     project_id  FK
        TEXT     env_type        "container | worktree | unset"
        TEXT     env_id          "container.id | worktree.id | '' (polymorphic, no FK)"
        TEXT     title
        TEXT     status
        DATETIME created_at
        DATETIME started_at
        DATETIME ended_at
    }
    session_agents {
        TEXT id         PK
        TEXT session_id FK
        TEXT agent_id   FK
        TEXT model_id   FK "model this agent ran with; null for prayer"
    }

    agents    ||--o{ agent_tools    : "granted"
    tools     ||--o{ agent_tools    : ""
    projects  ||--o{ containers     : "contains"
    projects  ||--o{ worktrees      : "contains"
    projects  ||--o{ sessions       : "scopes"
    sessions  ||--o{ session_agents : "includes"
    agents    ||--o{ session_agents : "joins"
    models    ||--o{ session_agents : "ran with"
    worktrees  }o..o{ sessions      : "env (polymorphic, no FK)"
    containers }o..o{ sessions      : "env (polymorphic, no FK)"
```

`models`, `agents`, `tools`, `agent_tools`, `projects`, `containers`, `worktrees`,
`sessions`, `session_agents` — all created by one goose migration in FK order. Each aggregate
table has an `id TEXT PRIMARY KEY NOT NULL` (UUID v7) with the natural key as `UNIQUE NOT NULL`;
foreign keys reference the parent **id** (`agent_id`, `tool_id`, `model_id`, `project_id`,
`session_id`) with `ON DELETE CASCADE`. `agent_tools` is a **pure junction** (composite PK
`(agent_id, tool_id)`); `session_agents` carries `model_id` so it takes a surrogate `id` PK
with `UNIQUE(session_id, agent_id)`. `sessions.env_id` is a **polymorphic** reference keyed by
`sessions.env_type` and therefore has **no FK** (validated in the domain). `casbin_rule` is
created and owned by the Casbin gorm-adapter, **not** the migration.
Schema/repository/migration rules: [conventions/persistence.md](../conventions/persistence.md).

### Shared packages (`packages/go`)

`gormdb` (dialect-agnostic GORM bootstrap), `gormdb/sqlite` (the pure-Go SQLite
dialector), `migrate` (instance-based goose `Provider` wrapper + `Create`), `casbinx`
(enforcer over the shared `*gorm.DB`).

## Rollout / migration

The schema ships as embedded goose migrations applied via `services/harness/cmd/migrate`
(`up`/`down`/`status`/`version`/`create`) against `.cirius-harness/state/harness.sqlite`.
There is no production history yet, so the initial schema is a single `…_initialize.sql`.

## Open questions

- GORM repository **adapters** in `internal/adapter/outbound` implementing the
  `port/outbound` repositories (deferred).
- The **seed migration** is done: it normalizes `.cirius-harness/00-system.yaml` into the
  `models` catalog, the `tools` catalog, the `agents` (roles), and their `agent_tools` grants.
  The per-agent **model** lines are not seeded — model is bound per session
  (`session_agents.model_id`). **Fallbacks** are not modeled yet. Agent **policies** into
  `casbin_rule` remain deferred (Casbin-owned, [ADR-0003](../adr/0003-authorization-casbin-abac.md)).
- **Unit of work** for cross-repository transactions (lands with the use cases).
- **Path-scoped permissions** for `scribe` (knowledge store only) via Casbin `keyMatch`.
- `cmd/harness` entrypoint exists with a `serve` subcommand — the Pi client stdio handshake
  ([ADR-0008](../adr/0008-pi-client-integration-stdio.md)). The `migrate` CLI may still fold
  into a subcommand later; MCP / events adapters and client **governance** remain deferred.

## References

- ADR-0001 / 0002 / 0003 above; [conventions/persistence.md](../conventions/persistence.md);
  [glossary](../glossary/README.md); `.cirius-harness/README.md`.
