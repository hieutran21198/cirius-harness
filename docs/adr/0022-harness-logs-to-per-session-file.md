# 0022. Harness logs to the per-session file by default (no console tee)

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0018](0018-harness-observability-logging-audit-session.md) (which decided logs
  "tee to stderr **and** a per-session file"; this changes the destination — the file is now the
  sole sink by default).

## Context

ADR-0018 made `serve` log to a per-session file **and** tee every line to stderr. But the harness
runs as a child the AI client launches, and a client relays the child's stderr into its **own
terminal UI** — the Pi extension does `proc.stderr.on("data", c => console.error("[harness] " + c))`
(`apps/pi-harness-extension/src/index.ts`), and Pi renders `console.error` in its TUI. The result:
harness log records (startup, every command via the logging decorator, GORM warnings) are
interleaved with the client's UI. ADR-0018 even noted the relay made the logs "hard to find" — but
it kept the tee, so the records still land in the UI. opencode (no adapter yet) would behave the
same.

stdout was never the problem: it is the protocol channel and `pilink.Serve` only writes NDJSON
frames there; diagnostics already go through the injected `slog.Logger`. The single place that
chooses the log destination is `newLogger` in `cmd/harness/main.go`.

## Decision

**Logs go to the per-session file only. The console (stderr) is used as the sink only when the
file is disabled.** In `newLogger`, when a log file is active the writer is the file alone
(`w = f`) — not `io.MultiWriter(os.Stderr, f)`. The console writer is injected (so it is
unit-testable) and used only when `HARNESS_LOG_FILE="-"` explicitly disables the file — that stays
the deliberate escape hatch for "I want console logs". stdout is never touched. File path
(`.cirius-harness/state/logging/<session_id>.log`), level (config + `HARNESS_LOG_LEVEL`), and
format (`HARNESS_LOG_FORMAT`) are unchanged from ADR-0018.

Fixing this at the source (the harness) rather than in each client adapter is intentional: the
harness simply stops emitting log records on a console stream, so **every** client — Pi today,
opencode tomorrow — gets a clean UI for free, regardless of how it pipes the child's streams.

## Consequences

- **Positive**: a client's UI is no longer polluted by harness log records; the per-session file
  remains the complete, discoverable record. One-line change at a single chokepoint fixes every
  downstream consumer (app handlers, the logging/audit decorators, GORM, pilink) because they share
  the one logger. No wire change; no client/extension change.
- **Negative**: with file-only logging, output written to stderr *before* the logger exists (a Go
  panic, a pre-`newLogger` startup error) still reaches stderr and a client may still relay it —
  but that is rare and is exactly the fatal case worth surfacing. Tailing logs live now means
  tailing the file (or `HARNESS_LOG_FILE="-"`), not watching the client console.
- **Neutral**: no migration, no new config field. `HARNESS_LOG_FILE="-"` (console only) and a
  custom `HARNESS_LOG_FILE=<path>` both still work.

## Alternatives considered

- **Keep teeing to stderr** — rejected: that is the cause; it puts log records in the client UI.
- **Fix only in the client adapter** (drop/divert the relayed stderr in the Pi extension) —
  rejected: every current and future client would need the same patch; the harness is the one
  place to stop it. (Left as a possible later polish for crash-time stderr, not needed here.)
- **A `logging.console` config field** — rejected as scope creep: the existing `HARNESS_LOG_FILE`
  env already toggles console vs file.

## References

- [ADR-0018](0018-harness-observability-logging-audit-session.md) (logging/audit/session — refined
  here), [ADR-0008](0008-pi-client-integration-stdio.md) (stdout = protocol, stderr = logs/
  diagnostics), [ADR-0009](0009-deployment-topology-per-client-harness.md) (one session per
  child-harness).
- `services/harness/cmd/harness/main.go` (`newLogger`), `cmd/harness/main_test.go`,
  `apps/pi-harness-extension/src/index.ts` (the stderr relay, unchanged),
  [conventions/api.md](../conventions/api.md).
