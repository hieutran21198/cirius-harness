# Persistence conventions

Data, repository, and migration rules for every Go service that stores state. Decided in
[ADR-0002](../adr/0002-persistence-and-migrations.md) (engine + tooling) and
[ADR-0003](../adr/0003-authorization-casbin-abac.md) (authorization).

## Domain ↔ storage boundary

- **The domain is pure.** Aggregates and value types in `internal/domain` carry **no GORM
  tags and no GORM import**, and **no infrastructure interfaces** — only aggregates, value
  objects, and validation errors. Mapping to rows is the adapter's job.
- **Repository ports live in `internal/port/outbound`; impls live in `internal/adapter/outbound`**
  ([ADR-0004](../adr/0004-ports-and-adapters-topology.md)). Dependencies point inward
  (hexagonal) — `port` imports `domain`, adapters import `port`, never the reverse.
- **One aggregate repository per domain, named by the plural** of the aggregate: `Agents`,
  `Sessions`, `Projects`, `Worktrees`. It carries the aggregate's whole graph (e.g.
  `Sessions.Save` persists a session and its members together).
- **Cross-aggregate references are by key value** (a string), not embedded structs — e.g.
  `session.Member.Agent` holds an agent name, `worktree.Worktree.Project` a project name.
- **Unit of work is deferred** until a use case needs a transaction spanning two
  repositories; introduce it with that domain, not before.

## Domain types

- **Enums are typed strings with a `Valid()` method**, not empty structs or bare strings —
  see [go.md](go.md) ("no poor-man's enums"). Each aggregate has a `Validate() error` and a
  package-level `ErrInvalid<Aggregate>` in its domain package; the repository `ErrNotFound`
  lives with the ports in `internal/port/outbound`.

## Schema

- **snake_case** table and column names.
- **Child and join tables** use a composite primary key and a foreign key back to the
  parent with **`ON DELETE CASCADE`** (e.g. `agent_tools`, `agent_fallbacks`,
  `session_agents`). Order rows with an explicit `position` column when order matters.
- **Keys**: use a **natural key** where the identity is stable and addressable — agent
  name, project name, worktree path. Use a **UUID v7** surrogate where there is no natural
  key (session id). Generation/clock stamping happens in the application/adapter, not the
  domain.
- **`casbin_rule` is owned by the Casbin gorm-adapter** (AutoMigrate). Never hand-write it
  in a migration.

## Migrations

- **goose**, embedded via `//go:embed *.sql`, run from Go (no CLI, no CGO).
- **Filenames** use goose's timestamp scheme `yyyymmddhhMMss_snake_case_purpose.sql`.
  Create one with `go run ./services/<svc>/cmd/migrate create <purpose>` (writes to the
  source dir; rebuild re-embeds it).
- **One logical change per migration**; `Up` creates in dependency (FK) order, `Down` drops
  in reverse.

## Bootstrapping

Build the dialect first, then wrap it; drive migrations through an instance:

```go
dialect, _ := sqlite.New(path)              // packages/go/gormdb/sqlite
db, _ := gormdb.New(ctx, dialect)           // packages/go/gormdb (dialect-agnostic)
m, _ := migrate.New(sqlDB, fsys, migrate.DialectSQLite)  // packages/go/migrate
```

- **DB path**: `.cirius-harness/state/{service}.sqlite` (gitignored runtime state), chosen
  by the caller.
- Authorization uses `casbinx.NewEnforcer(db, modelText)` over the **same** `*gorm.DB`.

## Anti-patterns

- GORM tags or imports inside `internal/domain`.
- Permission/authorization columns on a domain table — authz is Casbin
  ([ADR-0003](../adr/0003-authorization-casbin-abac.md)).
- Hand-writing `casbin_rule` in a goose migration.
- Sequential (`00001_`) migration prefixes — use the timestamp scheme.
