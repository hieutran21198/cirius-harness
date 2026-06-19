# 0002. Persistence engine + migration tooling

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

> **Refined by [ADR-0004](0004-ports-and-adapters-topology.md):** the repository *ports*
> described here now live in `internal/port/outbound` (not the domain) and their impls in
> `internal/adapter/outbound`. The repository pattern and naming are unchanged.

## Context

[ADR-0001](0001-harness-layout.md) decided the layout and the research→code pipeline, and
explicitly deferred the **database engine and migration tooling** to this ADR. The pipeline
ends in "a seed migration writes the system agents into a database", so Module 1
(`services/harness`) needs a persistence stack before the domains can be stored.

Constraints that shaped the choice:

- **No CGO.** The harness must cross-compile and run in slim CI/containers without a C
  toolchain. This rules out the common `mattn/go-sqlite3` driver.
- **Single-file, zero-ops storage.** Per-service state is local
  (`.cirius-harness/state/{service}.sqlite`); there is no database server to operate.
- **Domain stays pure.** Hexagonal architecture (ADR-0001) requires repository *ports* in
  the domain with infrastructure impls in `adapters/` — the persistence choice must not
  leak into `internal/domain`.

## Decision

We will persist harness state in **SQLite**, accessed through **GORM**, with the
pure-Go **`glebarez/sqlite`** driver (no CGO). Migrations are managed by **goose v3**, run
**embedded** from Go (no goose CLI, no CGO).

Concretely:

- **Repository pattern.** Each domain exposes one aggregate repository *port* (an
  interface in the domain package), named by the **plural** of the aggregate — `Agents`,
  `Sessions`, `Projects`, `Worktrees`. Implementations live in `internal/adapters`.
- **Unit of work is deferred.** Cross-repository transactions will be introduced with the
  second domain that needs them; today each repository owns its own writes.
- **Shared building blocks** live in `packages/go`:
  - `gormdb` — **dialect-agnostic** GORM bootstrap: `New(ctx, dialector) → *gorm.DB`
    (shared slog logging + a liveness ping); it imports no driver of its own.
  - `gormdb/sqlite` — builds the SQLite `gorm.Dialector` (DSN pragmas
    `foreign_keys`/`busy_timeout`/`journal_mode=WAL`, plus `MkdirAll`).
  - `migrate` — an **instance-based** wrapper over goose's `Provider` (no process-wide
    globals): `New(db, fsys, dialect) → *Migrator` with `Up`/`Down`/`Status`/`Version`,
    plus a standalone `Create` for new files.
- **Migration files** are embedded via `//go:embed` and named with goose's **timestamp**
  scheme `yyyymmddhhMMss_snake_case_purpose.sql` (created via `migrate create <purpose>`).
- **DB location**: `.cirius-harness/state/{service}.sqlite` (gitignored runtime state),
  chosen by the caller.

## Consequences

- **Positive**: pure-Go end to end ⇒ trivial cross-compilation and minimal container
  images; no C toolchain in CI.
- **Positive**: the dialect lives only in `gormdb/sqlite` and the goose dialect is a
  parameter, so the generic machinery can back another engine later without a rewrite.
- **Positive**: migrations are an embedded asset of the binary — `migrate up` needs no
  external tool and the schema ships with the service.
- **Negative**: the Casbin gorm-adapter (see [ADR-0003](0003-authorization-casbin-abac.md))
  transitively pulls MySQL/Postgres/MSSQL drivers into the module graph as **indirect**
  deps; they are unused but present. Accepted as the cost of the official adapter.
- **Neutral**: the `casbin_rule` table is created and owned by the gorm-adapter
  (AutoMigrate), so it is deliberately **absent** from the goose migrations.

## Alternatives considered

- **CGO `mattn/go-sqlite3`** — the most common driver. Rejected: requires CGO, defeating
  the no-C-toolchain constraint.
- **Raw `database/sql` + `sqlc`** — generated typed queries, no ORM. Rejected: the team
  chose GORM + the repository pattern for ergonomics over hand-written SQL at this stage.
- **goose CLI / `golang-migrate`** — external migration binaries. Rejected: we want
  migrations embedded in the service binary and driven from Go, with no extra tool to
  install and no CGO.

## References

- [ADR-0001](0001-harness-layout.md) — layout + research→code pipeline (deferred this ADR).
- `docs/conventions/persistence.md` — the data/repository/migration rules that follow.
- `docs/specs/harness-data-model.md` — the schema this stack persists.
- Versions: `glebarez/sqlite` v1.11.0, `gorm.io/gorm` v1.31.1, `pressly/goose/v3` v3.27.1.
