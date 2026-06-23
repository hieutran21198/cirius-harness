# API conventions

Contract rules for the three interfaces between the harness and its clients:
**CLI command (cmd)**, **MCP**, and **async events**.

HTTP and gRPC are out of scope — the harness is a control plane over a client, not a
network service. Adding either is an architectural decision and needs an ADR.

## Source of truth

Every contract has **one** authoritative definition:

- **CLI command (cmd)** → the command's flags, arguments, and structured (JSON)
  output schema, defined in one place per command. The Pi-facing inbound contract is
  `harness serve` ([ADR-0008](../adr/0008-pi-client-integration-stdio.md)): a stdio
  process spoken in **newline-delimited JSON** — **stdout is the protocol channel**;
  diagnostics go to the per-session log file, with stderr used only as the fallback when
  the file is disabled ([ADR-0022](../adr/0022-harness-logs-to-per-session-file.md)).
  Framing is **LF-only** (split on `\n`; do not use readers that also break on
  U+2028/U+2029).
- **MCP** → tool and resource schemas (JSON Schema) published by the MCP server.
- **Async events** → event schemas (JSON Schema or AsyncAPI) for the event stream.

Consumers generate from the authoritative schema; they do **not** hand-roll types.
(Concrete file locations land with the layout decision in ADR-0001.)

### `harness serve` stdio frames

Each frame is one LF-delimited JSON object with a `type` and an optional `id` the client
sets and the harness echoes on the reply (so a client can correlate request↔response).

| Direction | `type` | Purpose |
| --- | --- | --- |
| in  | `hello` | client announces itself (`cwd`, `pid`) |
| out | `ready` | handshake accepted (`schemaVersion`, `dbPath`, `pid`) |
| in  | `ping` | liveness probe |
| out | `pong` | reply to ping |
| in  | `models` | client reports its enabled models (`client` **required**, `models: [{provider, slug}]`) — synced into the catalog keyed per-client `(client, provider, slug)` ([ADR-0011](../adr/0011-client-reported-model-catalog.md), [ADR-0015](../adr/0015-client-aware-model-catalog.md)); an unknown/missing `client` is an `error` frame |
| out | `models_synced` | sync result (`added`, `total`) |
| in  | `resolve_agent` | client asks the harness to resolve an agent (`agent`, `client`) so it can govern a turn as that agent ([ADR-0016](../adr/0016-harness-owned-agent-persona-governed-turn.md)); an unknown/missing `client` or unknown `agent` is an `error` frame |
| out | `agent_resolved` | resolved agent (`name`, `persona`; `model` empty until the resolver milestone) |
| in  | `submit_plan` | client submits a human-approved council plan to persist (`agent`, `client`, `plan` — the plan object matching the harness contract) ([ADR-0019](../adr/0019-persist-council-orchestration-plan.md)); an unknown/missing `client`, no active session, or an invalid plan is an `error` frame |
| out | `plan_recorded` | plan persisted (`planId`, `taskCount`) |
| in  | `get_plan` | client fetches a persisted plan to drive (`client`; `planId` optional — empty fetches the session's latest) ([ADR-0021](../adr/0021-drive-the-council-plan.md)); an unknown/missing `client` or no plan is an `error` frame |
| out | `plan` | the fetched plan (`planId`, `status`, `plan` — the OrchestrationPlan contract, `taskIds` — ref→task-id) |
| in  | `report_run` | client records drive progress (`client`, `planId`, optional `planStatus`, optional `task: {ref, status, summary}`) ([ADR-0021](../adr/0021-drive-the-council-plan.md)); an unknown status or illegal transition is an `error` frame |
| out | `run_reported` | drive progress recorded (`planRunId`, `status`) |
| out | `error` | a frame could not be handled (`message`) |

New frames are additive (same rule as cmd output). The Go contract lives in
`services/harness/internal/delivery/pilink`.

## Versioning

- **cmd**: flags and output fields are additive. A breaking change ships a new
  command (or `v2` subcommand); the old one stays until consumers migrate.
- **MCP**: tool schemas evolve additively. Renaming/removing a tool or field is breaking.
- **Events**: additive only. Required fields are append-only; deprecated fields stay
  reserved.

## Errors

- **cmd**: non-zero exit code + a structured error on stdout (`code`, `message`,
  optional `detail`).
- **MCP**: return an error result from the tool call, not a thrown transport error.
- **Events**: never represent errors as events. Errors flow back through the request
  channel that produced them.

## Pagination

List outputs are bounded today (the default team is small, sessions are scoped), so commands
return full result sets. When a list can grow unbounded, add **opaque cursor** pagination
(`--cursor` / `next_cursor` in the JSON output) — never numeric offsets, which drift under
concurrent writes. Adding it is an additive, non-breaking change.

## Anti-patterns

- **Adding an HTTP or gRPC surface without an ADR.** The harness speaks cmd / MCP /
  events; a new transport is an architectural decision.
- **Hand-rolling contract types** instead of generating from the authoritative schema.
