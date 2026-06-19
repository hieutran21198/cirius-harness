# 0001. Harness layout + research→code pipeline

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

> **Refined by [ADR-0004](0004-ports-and-adapters-topology.md):** ports now live in
> `internal/port/{inbound,outbound}` and adapters in `internal/adapter/{inbound,outbound}`
> (singular), superseding this ADR's `adapters/{…}` layout detail.

## Context

cirius-harness is a **control plane** that governs a **client** (opencode is the
reference) through a declarative agent–model schema. It exists for three goals:
**harness AI-coding** (govern the client), **AI-agents orchestration** (the declarative
agent team), and **concurrency works** (view/control concurrent work across modules —
Module 2, reserved). Before writing engine code we need two things settled:

1. **A home for well-researched knowledge** — which AI models are good/bad at what,
   which tools and clients to use — and a defined path from that knowledge to **real Go
   code**. The model-personality reasoning already living in `.cirius-harness/00-system.yaml`
   comments is evidence of this need; it has no formal home today.
2. **A module structure.** The repo is an umbrella monorepo. The first module is the
   harness engine (built now); a second module — wiring multiple modules to view/control
   concurrency — is anticipated but not yet designed.

This ADR is **one decision**: the top-level layout and the research→code pipeline. It is
doc-only. The DB engine, migration tooling, and the Go engine itself are deferred (see
Consequences / References).

## Decision

We will adopt an **umbrella polyglot monorepo** (the layout already described in
`AGENTS.md`) and add a **research→code pipeline** with two new documentation layers:

```
cirius-harness/
├── .cirius-harness/00-system.yaml   # system default team = human-readable SOURCE SPEC
├── docs/
│   ├── research/                     # evidence: model good/bad, tools, clients
│   ├── pdr/                          # Provider Decision Records (decisions from research)
│   └── adr/  specs/  conventions/  glossary/
├── packages/go/<contract>/          # shared cmd/mcp/events contracts (future)
└── services/
    ├── harness/                      # MODULE 1 — built now (hexagonal + DDD)
    │   ├── cmd/harness/              #   CLI entrypoint (cmd contract)
    │   ├── internal/
    │   │   ├── domain/agent/         #   ai-agent domain: Agent aggregate
    │   │   │                         #     (model, permissions, tools, fallbacks)
    │   │   ├── application/          #   use cases: load config, resolve agent, fallback, route
    │   │   └── adapters/{store,client,config,mcp,events}
    │   └── migrations/               #   seed system agents (council, planner, … prayer)
    └── <module-2>/                   # FUTURE — concurrency view/control (reserved)
```

The **research→code pipeline** flows in one direction:

1. **`docs/research/`** — gather evidence on models, tools, clients.
2. **`docs/pdr/`** — record provider/model/tool/client decisions, citing the research.
3. **`.cirius-harness/00-system.yaml`** — encode those decisions as the declarative
   default team (the human-readable source spec).
4. **`services/harness`** — model agents as the `ai-agent` DDD domain; a **seed
   migration** writes the system agents into a database.
5. **Runtime** — the user's config file overlays/enables agents on top of the
   DB-seeded system defaults.

There is **no codegen**: the schema is realized as a hand-modeled domain plus a seed
migration, not generated source.

## Consequences

- **Positive**: research, decisions, declarative schema, and runtime state each have one
  clear home, and the path between them is explicit. Module 1 can be built in isolation;
  Module 2 has a reserved slot without being over-designed now.
- **Positive**: the system default team becomes queryable, auditable DB state (seeded by
  migration) rather than a file parsed at boot — supporting the observability/audit pillar.
- **Negative**: the system defaults now live in **two places** — `00-system.yaml` (source
  spec) and the seed migration (runtime truth). They must be kept in sync; the migration
  is authoritative at runtime.
- **Negative**: introduces a database dependency for what could otherwise be a
  file-only tool. Engine and tooling choices are pushed to a follow-up ADR.
- **Neutral**: the umbrella monorepo carries TS/nx/pnpm plumbing the harness module does
  not yet use; it stays dormant until Module 2 or a UI needs it.

## Alternatives considered

- **File-only, schema read at runtime (no DB)** — the harness loads `00-system.yaml`
  + user overlay at boot, no persistence. Rejected: the user wants agents as a persisted
  `ai-agent` domain with migrations, and persisted state serves audit/observability.
- **Codegen from schema** — generate typed Go from the YAML/research. Rejected: more
  machinery than a hand-modeled domain + seed migration warrants at this stage.
- **Go-first single-module repo** (`cmd/ internal/ pkg/`, drop the polyglot template) —
  simpler for a mainly-Go engine. Rejected: a second module (concurrency view/control) is
  anticipated, so the umbrella monorepo is kept.

## References

- `AGENTS.md` — umbrella monorepo layout and conventions.
- `.cirius-harness/README.md` — the agent schema (default team + overlay) this pipeline feeds.
- `docs/conventions/api.md` — the cmd / MCP / events contract surface.
- `docs/research/README.md`, `docs/pdr/README.md` — the two new doc layers.
- [ADR-0002](0002-persistence-and-migrations.md) — database engine + migration tooling (the deferral resolved); [ADR-0003](0003-authorization-casbin-abac.md) — authorization. Still deferred: Module 2 design.
