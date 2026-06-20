# 0008. Pi client integration over a stdio child process

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

The harness is a control plane **over** a coding client ([ADR-0001](0001-harness-layout.md)),
and the reference client is now **Pi** (`@earendil-works/pi-coding-agent`, pi.dev) — a
TypeScript-extensible terminal coding harness. Until now the inbound "wire" between the
harness and a client ([conventions/api.md](../conventions/api.md): cmd / MCP / events) was
unbuilt: only the domain and DB exist.

We need the first concrete connection so later governance (model handoff, permission
gating, tool grants) has a live channel. Pi's extension model fixes the shape of that
connection:

- Pi loads TypeScript **extensions** from `.pi/extensions/` (after the project is trusted).
  An extension's factory runs at startup; Pi awaits async factories before `session_start`.
- Extensions must **not** spawn background processes in the factory — Pi's docs direct that
  to `session_start`, with an idempotent `session_shutdown` for teardown.
- Pi already speaks strict **LF-delimited JSON** over stdio for its own RPC mode, and warns
  that generic line readers (e.g. Node `readline`) are not protocol-compliant because they
  split on U+2028/U+2029.

The decision is narrow: **which direction the connection runs, and what carries it.** Not
what the harness governs (deferred).

## Decision

We will integrate with Pi via a **project-local Pi extension that launches the harness as a
child process and talks to it over stdio** — direction **Pi-launches-harness**, transport
**one harness process per Pi session**.

- The harness exposes a `harness serve` subcommand (an inbound adapter under
  `internal/adapter/inbound/pilink`, per [ADR-0004](0004-ports-and-adapters-topology.md))
  that speaks newline-delimited JSON (NDJSON) on stdin/stdout: **stdout is the protocol
  channel, stderr is for logs**, framing is **LF-only**.
- The Pi extension (`.pi/extensions/harness/`) spawns `harness serve` on `session_start`,
  performs a hello/ready handshake, and kills it on `session_shutdown`.

This realizes the **cmd / process-integration** wire from `conventions/api.md`; it is not a
new transport (no HTTP/gRPC), so no further transport ADR is required.

## Consequences

- **Positive**: simplest possible live channel — no daemon, no ports, no auth; the harness
  lifecycle is bound to the Pi session, so there is nothing to leak or reap across runs.
  Reuses Pi's own stdio/LF-JSON discipline, so the framing is battle-tested.
- **Negative**: one harness process per session means **no cross-session view** — Module 2's
  concurrency control will likely need a long-running daemon, at which point the transport
  (not the contract) is swapped. Per-session process spawn adds small startup cost.
- **Neutral**: the extension lives in the repo (`.pi/extensions/` is tracked, while Pi's
  runtime state under `.pi/` stays gitignored). The harness binary is built ahead of time
  (`devenv tasks run harness:build`) rather than `go run` on each launch.

## Alternatives considered

- **Harness launches Pi (RPC/SDK)** — a `harness run` launcher spawns `pi --mode rpc` or
  embeds the Pi SDK and drives it. Rejected for v1: it moves the entrypoint away from `pi`,
  which is the command the user actually runs; revisit if the harness needs to own headless
  runs.
- **Long-running harness daemon over a socket** — a persistent process holding DB/state that
  Pi connects to each session. Rejected for v1 as premature; it is the natural Module 2
  evolution, deferred until the cross-session view needs it.
- **Harness as an MCP server** — register the harness under Pi's MCP integration. Rejected:
  MCP is model-facing tools/resources, a poor fit for governing the session itself
  (model selection, permission gates) which Pi exposes through its extension API, not MCP.

## References

- [ADR-0001](0001-harness-layout.md) (layout / control-plane mission),
  [ADR-0004](0004-ports-and-adapters-topology.md) (inbound adapter placement),
  [conventions/api.md](../conventions/api.md) (cmd / MCP / events wire).
- Pi docs (installed with the binary): `docs/extensions.md` (lifecycle, `session_start` /
  `session_shutdown`), `docs/rpc.md` (LF-JSONL framing), `examples/extensions/{subagent,
  interactive-shell,ssh,sandbox}` (long-lived child spawn pattern).
