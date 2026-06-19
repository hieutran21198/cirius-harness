# packages/go

**One Go module** (`harness-workspace/packages/go`) whose sub-directories are
**packages, not modules**. Shared, internal-only Go code imported by services as
`harness-workspace/packages/go/<name>`.

## Packages

| Package   | Purpose                                                                 |
| --------- | ---------------------------------------------------------------------- |
| `gormdb`  | Dialect-agnostic GORM bootstrap: `New(ctx, dialector, …) → *gorm.DB` (shared slog logging + a liveness ping). No engine import of its own. |
| `gormdb/sqlite` | Builds the pure-Go SQLite `gorm.Dialector` (`glebarez/sqlite`, no CGO): `New(path, …)` owns the DSN pragmas (foreign_keys, busy_timeout, WAL) + MkdirAll. Pair with `gormdb.New`. |
| `slogx`   | Helpers over `log/slog`: a handler factory (`New`), `ParseLevel`, and context carry (`WithContext`/`FromContext`). |
| `casbinx` | Bootstraps a Casbin enforcer + gorm-adapter over an existing `*gorm.DB` (the `casbin_rule` table). Model + decision semantics belong to the caller. |
| `migrate` | Instance-based goose wrapper: `New(db, fsys, dialect) → *Migrator` (`Up`/`Down`/`Status`/`Version`, no goose globals) + standalone `Create` for timestamped (`yyyymmddhhMMss_purpose.sql`) files. |

## Conventions

- **One module, many packages.** Shared internal code has no external consumers and
  doesn't need independent versioning. Add a package as a new sub-directory; do **not**
  add a `go.mod`.
- **No service-specific logic.** These packages are reusable infrastructure. Anything that
  knows about a domain (agents, authz model.conf, …) belongs in that service.
- **Escape hatch.** If a package genuinely needs independent release cadence or external
  consumers, promote it to its own module — but that is an ADR-worthy decision, not a
  default. Until then, keep the single module.

## Gotchas

- `casbinx` depends on `casbin/gorm-adapter/v3`, which transitively pulls the MySQL /
  Postgres / SQL Server GORM drivers as **indirect** deps. They are unused at runtime
  (we only open SQLite) but appear in `go.sum`.
- Builds resolve via the root `go.work`; never add `replace` directives here.
