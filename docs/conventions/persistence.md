# Persistence conventions

Data, repository, and migration rules for every Go service that stores state. Decided in
[ADR-0002](../adr/0002-persistence-and-migrations.md) (engine + tooling),
[ADR-0003](../adr/0003-authorization-casbin-abac.md) (authorization), and
[ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md) (Reader/Writer + UnitOfWork).

## Domain ↔ storage boundary

- **The domain is pure and encapsulated.** The single `internal/domain` package
  ([ADR-0014](../adr/0014-domain-encapsulation-single-package.md)) carries **no GORM tags and no
  GORM import**; aggregates hold **unexported** fields. The adapter never reads or sets a
  domain field: it persists an aggregate through its grouped view (`m.Snapshot()` → row) and
  reconstitutes one through the `RehydrateXxx` constructor (row → aggregate). Both the view and
  `RehydrateXxx` validate, so a row that violates an invariant fails loudly on the way in.
- **Per-aggregate `Reader`/`Writer` interfaces live in the `domain` package** (e.g.
  `domain.ModelWriter`); they speak only domain types
  ([ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md)). Commands obtain `Writer`s
  from a **`UnitOfWork`** (in `app/command`); queries obtain `Reader`s from a **`ReadStore`**
  (in `app/query`). GORM implementations live under `internal/infra`. Dependencies point
  inward — `domain`/`app` never import `infra`.
- **One `Writer` (and, when a query needs it, `Reader`) per aggregate.** A `Writer` save carries
  the aggregate's whole graph (e.g. a session writer persists a session and its members
  together; an agent writer persists an agent and its tool grants). Methods are sized to the
  use case — `domain.ModelWriter` batches: `SaveAll` upserts many at once, `Existing(refs)` does
  a targeted lookup (a `(client, provider, slug)` tuple `IN` over the reported `domain.Ref`s,
  keyed by the comparable `Ref`) for a membership check before the batch — its cost scales with
  the request, not the catalog. The catalog's natural key is `(client, provider, slug)`: model
  names are client-specific, so the client is part of identity
  ([ADR-0015](../adr/0015-client-aware-model-catalog.md)). Add a Reader/Writer when a use case
  needs it, not speculatively.
- **Cross-aggregate references are by the UUID id**, not embedded structs and not the natural
  key — e.g. a session member's agent id and model id, a session's project id, a worktree's
  project id, a container's project id
  ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)). The id is the **owning
  aggregate's typed `~string`** (`AgentID`, `ModelID`, `ProjectID`, …), not a bare `string`, so
  a swap is a compile error ([go.md](go.md), [ADR-0014](../adr/0014-domain-encapsulation-single-package.md)).
- **A polymorphic reference** (e.g. a session's env id keyed by its env type) has **no foreign
  key** — the target table varies, so referential integrity is enforced in the domain's
  `Validate()`, not the schema. It is also the one id that stays a bare `string` (it is a
  `WorktreeID` *or* a `ContainerID`, so no single typed id fits).
- **Commands mutate through a `UnitOfWork`**: `DoTx(ctx, func(tx TransactionalUnitOfWork) error)`
  runs the closure in one transaction (commit on nil, rollback on error), the writers inside it
  bound to that transaction ([ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md)).
  Implemented by `infra/sqlite/unitofwork` (composing the GORM repos in `infra/sqlite/repo`)
  over GORM's `db.Transaction`. The **read side** (`ReadStore` + domain `Reader`s, →
  `infra/sqlite/readstore`) is deferred until the first query.

## Domain types

- **Enums are typed strings with a `Valid()` method**, not empty structs or bare strings —
  see [go.md](go.md) ("no poor-man's enums"). Each aggregate has a `Validate() error` and a
  package-level `ErrInvalid<Aggregate>` in its domain package. A repository `ErrNotFound`
  sentinel is defined where the read side consumes it, reintroduced with the first `Reader`
  ([ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md)).

## Schema

- **snake_case** table and column names.
- **Pure junction tables** use a composite primary key and a foreign key back to each parent
  with **`ON DELETE CASCADE`** (e.g. `agent_tools`). A join that **carries its own attribute**
  takes a surrogate `id` PK instead, with a `UNIQUE` over the pair (e.g.
  `session_agents(id, …, UNIQUE(session_id, agent_id))`, which carries `model_id`) — see
  [ADR-0007](../adr/0007-roles-and-per-session-model-binding.md). Order rows with an explicit
  `position` column when order matters.
- **Keys**: every aggregate has a **UUID v7 surrogate** primary key
  (`id TEXT PRIMARY KEY NOT NULL`), **minted inside the `NewXxx` constructor** (via the domain's
  shared `newID()` → `uuid.Must(uuid.NewV7())`) — the id format is a domain policy, so the app
  supplies only business attributes and never imports `uuid`
  ([ADR-0005](../adr/0005-surrogate-uuid-v7-keys.md)). Natural keys (agent name, project name,
  worktree path) are `UNIQUE NOT NULL` attributes — the business/lookup key, not the identity.
  SQLite generates nothing: the id exists on the in-memory aggregate before the insert (the
  caller reads it back via `Snapshot()`), never from the DB. The id is a **typed `~string`**
  (`ModelID`, …); the grouped view carries the typed id and the GORM row struct holds a plain
  `string`, so the single `string(id)` cast lives in the repo's row mapper. Clock stamping still
  happens in the application/adapter, not the domain.
- **Per-run config is bound on the run, not the definition.** Rather than versioning a shared
  definition, record the choice on the runtime row that uses it — e.g. the model an agent ran
  with lives on `session_agents.model_id`, not on `agents`
  ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)). Editing the definition then
  cannot rewrite past runs, with no version/`is_current` machinery.
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

- Natural-key primary keys, or a DB `DEFAULT` / GORM `BeforeCreate` hook that mints ids in
  the persistence layer — the UUID v7 is minted inside the aggregate's `NewXxx` constructor
  ([go.md](go.md)), so it exists on the in-memory aggregate before the write; never assign an
  id to a field after the fact.
- The app importing `uuid` to mint ids — the id format is a domain policy; construct via
  `domain.NewXxx(...)` (which mints internally) instead.
- Storing a per-run choice (e.g. the model) on a shared, editable definition row — record it
  on the runtime row that uses it (`session_agents.model_id`), so editing the definition can't
  rewrite history ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)).
- GORM tags or imports inside `internal/domain`.
- Permission/authorization columns on a domain table — authz is Casbin
  ([ADR-0003](../adr/0003-authorization-casbin-abac.md)).
- Hand-writing `casbin_rule` in a goose migration.
- Sequential (`00001_`) migration prefixes — use the timestamp scheme.
