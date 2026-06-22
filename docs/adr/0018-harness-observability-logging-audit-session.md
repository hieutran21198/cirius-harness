# 0018. Harness observability: structured logging, audit log, session recording

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

The harness needed to be observable — to answer "did it launch and is it working?" and to
keep a durable record of what it did. Before this, logging was a bare `slog.TextHandler` to
stderr (hardcoded level, no file), GORM logging was off, transport frames were not logged, and
nothing was persisted about a run: the `sessions` / `session_agents` tables existed but were
never written, and there was no audit trail. The Pi extension forwards the child's stderr into
its own console, so even those logs were hard to find.

## Decision

Ship observability in three coherent parts.

1. **Configurable structured logging to a per-session file.** `serve` builds the logger via
   `packages/go/slogx`. The **level** comes from config — the system base
   `.cirius-harness/00-system.yaml` (`logging.level: info`) overridden by the user overlay
   `.cirius-harness/config.yaml`; `HARNESS_LOG_LEVEL` is a final ad-hoc override.
   `HARNESS_LOG_FORMAT` selects text/json. Logs tee to stderr **and** a per-session file at
   `.cirius-harness/state/logging/<session_id>.log` (overridable via `HARNESS_LOG_FILE`, `-`
   disables); every line is tagged with the session id. stdout stays the protocol channel.
   Reading the config is the **first slice of the deferred config loader** (`internal/infra/config`,
   logging only; the agent/model resolver and deep-merge of ADR-0011 remain deferred).
2. **Persisted append-only audit log.** A new `events` table and `domain.Event` aggregate,
   written through a **command audit decorator** (sibling to the logging decorator): every
   command records one event (kind = command name, ok/error status, actor from the context via
   `internal/app/appctx`). Audit is observational — a failed append is logged, never propagated.
3. **Session recording.** On `hello` the harness ensures the project (from the client cwd) and
   saves a `Session` under the id minted at startup (the same id naming the log file); on
   `resolve_agent` it records the agent as a session member (`session_agents`, model NULL until
   model governance lands). Both are best-effort — a recording failure never aborts the wire
   handshake. New driven ports `domain.EventWriter`, `ProjectWriter`, `SessionWriter` on the
   `command.UnitOfWork` (ADR-0013).

## Consequences

- **Positive**: launch and per-frame function are visible in a discoverable file; commands leave
  a queryable audit trail; runs (who connected, which agent ran) are persisted. The audit
  decorator covers every command automatically, including the new session commands. No new wire
  frame; the extension is unchanged.
- **Negative**: introduces a yaml dependency and the first config-file read ahead of the full
  loader — scoped to logging to contain it. The audit event payload is generic (command name +
  status) for now; richer per-command detail is future work. `session_agents.model_id` is NULL
  until model resolution exists.
- **Neutral**: one new migration (`events`); `sessions`/`session_agents`/`projects` tables were
  already present (initialize migration) and are now written.

## Alternatives considered

- **Log level via env only** — rejected: the user wanted it in config, with the system default in
  `00-system.yaml`. Env remains as an override.
- **Single shared log file** — rejected: a per-session file isolates concurrent/successive runs
  and matches the one-session-per-`serve` model (ADR-0009).
- **Audit inside each handler** — rejected: a decorator keeps it cross-cutting and uniform; the
  handlers stay pure (ADR-0012).
- **Build the full config loader now** — deferred: only logging is needed; the agent/model
  resolver + deep-merge stay a separate milestone (ADR-0011).

## References

- [ADR-0008](0008-pi-client-integration-stdio.md) (stdio frames / channel discipline),
  [ADR-0009](0009-deployment-topology-per-client-harness.md) (one session per child-harness),
  [ADR-0011](0011-client-reported-model-catalog.md) (the config merge this begins),
  [ADR-0012](0012-cqrs-application-layer.md) / [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md)
  (decorators, driven ports).
- `cmd/harness/main.go`, `internal/infra/config`, `internal/app/{appctx,decorator}`,
  `internal/domain/{event,session,project}.go`, `internal/infra/sqlite/repo`,
  `migrations/…_create_events.sql`.
