# Harness data model

- Status: Implemented
- Owner: hieu
- Reviewers: -
- Related ADRs: [ADR-0001](../adr/0001-harness-layout.md), [ADR-0002](../adr/0002-persistence-and-migrations.md), [ADR-0003](../adr/0003-authorization-casbin-abac.md)

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

**agent** — the declarative team member. `Agent` aggregate (Name, Model, Responsibility,
Archetype, Description, Source, Enabled, Tools, Fallbacks) with typed-string enums
`Archetype` (communicator | principle-driven | utility-runner | none), `Tool`, `Source`
(system | user). Repository port `outbound.Agents` (in `internal/port/outbound`, per
[ADR-0004](../adr/0004-ports-and-adapters-topology.md)). Permissions are **not** here — authorization is
Casbin ([ADR-0003](../adr/0003-authorization-casbin-abac.md)), with `outbound.Authorizer`
returning `authz.Decision` (allow | ask | deny) for an `authz.Action`.

**orchestration** — the runtime. Three aggregates, one folder each, plus a membership join:

```
Project ──1─N──> Worktree ──1─N──> Session ──N─N──> Agent
 (name PK)        (path PK)        (uuid v7 PK)   (via session_agents)
```

- `Project` (Name PK, RootPath, Kind {single | monorepo}, Description) — repo `Projects`.
- `Worktree` (Path PK, Project, Branch, Status {active | stale}) — repo `Worktrees`.
- `Session` (ID = UUID v7, Worktree, Title, Status {pending | running | completed | failed
  | cancelled}, timestamps, Members) — repo `Sessions`.
- `Member` (Agent, Role, Active, JoinedAt) — the live agent↔session join.

### Schema (SQLite)

`agents`, `agent_tools`, `agent_fallbacks`, `projects`, `worktrees`, `sessions`,
`session_agents` — all created by one goose migration in FK order; child/join tables use
composite PKs and `ON DELETE CASCADE`. `casbin_rule` is created and owned by the Casbin
gorm-adapter, **not** the migration. Schema/repository/migration rules:
[conventions/persistence.md](../conventions/persistence.md).

### Shared packages (`packages/go`)

`gormdb` (dialect-agnostic GORM bootstrap), `gormdb/sqlite` (the pure-Go SQLite
dialector), `migrate` (instance-based goose `Provider` wrapper + `Create`), `casbinx`
(enforcer over the shared `*gorm.DB`).

## Rollout / migration

The schema ships as embedded goose migrations applied via `services/harness/cmd/migrate`
(`up`/`down`/`status`/`version`/`create`) against `.cirius-harness/state/harness.sqlite`.
There is no production history yet, so the initial schema is a single `…_initialize.sql`.

## Open questions

- GORM repository **adapters** in `internal/adapter/outbound` implementing the four
  `port/outbound` repositories (deferred).
- The **seed migration** writing the default team into `agents` + agent policies into
  `casbin_rule`.
- **Unit of work** for cross-repository transactions (lands with the use cases).
- **Path-scoped permissions** for `scribe` (knowledge store only) via Casbin `keyMatch`.
- `cmd/harness` entrypoint (the `migrate` CLI may fold into a subcommand).

## References

- ADR-0001 / 0002 / 0003 above; [conventions/persistence.md](../conventions/persistence.md);
  [glossary](../glossary/README.md); `.cirius-harness/README.md`.
