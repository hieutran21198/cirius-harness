# services/harness

**Module 1** — the harness engine (control plane over an AI coding client). Module:
`harness-workspace/services/harness`. Hexagonal + DDD (see `services/AGENTS.md`).

## Layout

```
cmd/harness/                 # CLI entrypoint (cmd contract): `serve` stdio handshake (ADR-0008)
cmd/migrate/                 # DB migration CLI (embedded goose)
internal/
├── domain/
│   ├── agent/               # agent bounded context: Agent aggregate + enums (pure).
│   │                        #   Pure, stdlib only. NO permissions (see authz).
│   ├── project/ session/ worktree/   # orchestration aggregates + value objects (pure).
│   └── authz/               # authorization value objects: Decision (allow|ask|deny), Action.
├── port/
│   ├── inbound/             # driving ports (use-case interfaces) — future
│   └── outbound/            # driven ports: Agents/Projects/Sessions/Worktrees repos + Authorizer.
├── application/             # use cases — future
└── adapter/
    ├── inbound/             # driving adapters (CLI/MCP/events)
    │   └── pilink/          # Pi client wire: NDJSON-over-stdio serve loop (ADR-0008).
    │                        #   Transport only; Handler implemented by cmd/harness.
    └── outbound/
        └── casbinauthz/     # Casbin-backed outbound.Authorizer; embeds model.conf, stores
                             #   policy in casbin_rule via packages/go/casbinx.
migrations/                  # seed system agents + policies — future
```

## Persistence & authz

- **Store**: SQLite via `packages/go/gormdb`, at `.cirius-harness/state/harness.sqlite`.
- **Repositories**: each aggregate has a repository **interface** (plural name, e.g.
  `outbound.Agents`) in `internal/port/outbound`. GORM-backed implementations live under
  `internal/adapter/outbound` (the `agent` store is not yet implemented — interface only).
- **Authorization is Casbin ABAC**, not agent table columns. The agent is the **principal**
  (Casbin subject = agent name). Each policy line carries a `dec` value
  (`allow|ask|deny`), read via `EnforceEx` on the matched rule — so the three-valued
  decision survives Casbin's binary enforce. No match ⇒ `deny`. The `obj` matcher uses
  `keyMatch`, leaving room for path/url-scoped policies later.

## Status

Domain types + repository interfaces for `agent`/`project`/`session`/`worktree`/`container`/
`model`/`tool`; authz domain + Casbin adapter; seed migrations; and the `cmd/harness serve`
Pi handshake (`adapter/inbound/pilink`, [ADR-0008](../../docs/adr/0008-pi-client-integration-stdio.md))
implemented. Deferred: the GORM repository stores, policy/Casbin **seeding**, MCP / events
adapters, all client **governance** (model handoff, permission gating, tool grants), and the
unit-of-work (added with the use cases).
