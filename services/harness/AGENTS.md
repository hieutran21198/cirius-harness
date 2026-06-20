# services/harness

**Module 1** — the harness engine (control plane over an AI coding client). Module:
`harness-workspace/services/harness`. Idiomatic-Go layout + DDD (see `services/AGENTS.md`,
[ADR-0013](../../docs/adr/0013-idiomatic-go-layout-and-unit-of-work.md)).

## Layout

```
cmd/harness/                 # CLI entrypoint (cmd contract): `serve` stdio handshake (ADR-0008)
cmd/migrate/                 # DB migration CLI (embedded goose)
internal/
├── domain/
│   ├── agent/               # agent bounded context: Agent aggregate + enums (pure).
│   │                        #   Pure, stdlib only. NO permissions (see authz).
│   ├── model/               # Model aggregate + model.Writer (domain-owned driven port).
│   ├── project/ session/ worktree/   # orchestration aggregates + value objects (pure).
│   └── authz/               # authorization value objects: Decision (allow|ask|deny), Action.
├── app/                     # use cases — CQRS (ADR-0012); owns its driven ports (ADR-0013)
│   ├── app.go              #   Application{Commands, Queries} + New(uow, logger) wiring
│   ├── command/            #   write handlers + port.go (UnitOfWork). First: SyncModels.
│   ├── query/              #   read handlers + (later) ReadStore — none yet
│   └── decorator/          #   generic CommandHandler/QueryHandler + logging decorator (slog)
├── delivery/               # driving adapters; declare the app-usecase interface they call
│   └── pilink/             #   Pi client wire: NDJSON-over-stdio serve loop (ADR-0008).
│                           #   pilink.Handler is the driving port; cmd/harness implements it.
│                           #   TS client half: apps/pi-harness-extension (ADR-0010).
└── infra/                  # driven adapters implementing the app's driven ports
    ├── sqlite/             #   GORM persistence, layered:
    │   ├── repo/           #     Reader/Writer impls (model.Writer; SaveAll upserts on (provider,slug))
    │   ├── unitofwork/     #     composes repo writers → command.UnitOfWork (DoTx)
    │   └── readstore/      #     composes repo readers → query.ReadStore (with the first query)
    └── casbin/             #   Casbin authorizer (concrete Decide); model.conf + casbinx.
migrations/                  # seed system agents + policies — future
```

## Persistence & authz

- **Aggregates are constructed via a validating `New(...)` factory** (e.g. `model.New`,
  `session.New`) that applies creation defaults and validates; `Validate()` enforces a
  non-empty surrogate `ID`. The app mints the id (and stamps the clock) and passes it into
  `New` — it never sets domain fields directly ([conventions/go.md](../../docs/conventions/go.md)).
- **Store**: SQLite via `packages/go/gormdb`, at `.cirius-harness/state/harness.sqlite`.
- **Persistence is CQRS (ADR-0013)**: per-aggregate `Reader`/`Writer` interfaces live in the
  **domain** (first: `model.Writer` — Existing(refs)/SaveAll/Count, the lookup keyed by
  `model.Ref`). Commands mutate through
  `command.UnitOfWork` (`DoTx` = one GORM transaction), implemented by `infra/sqlite/unitofwork`
  composing `infra/sqlite/repo` (the GORM Reader/Writer impls). The read side
  (`query.ReadStore` + domain `Reader`s, → `infra/sqlite/readstore`) is **deferred**. Other
  aggregates get a Reader/Writer when a use case needs one.
- **Authorization is Casbin ABAC**, not agent table columns. The agent is the **principal**
  (Casbin subject = agent name). Each policy line carries a `dec` value
  (`allow|ask|deny`), read via `EnforceEx` on the matched rule — so the three-valued
  decision survives Casbin's binary enforce. No match ⇒ `deny`. The `obj` matcher uses
  `keyMatch`, leaving room for path/url-scoped policies later.

## Status

Domain types for `agent`/`project`/`session`/`worktree`/`container`/`model`/`tool`; authz
domain + Casbin authorizer (`infra/casbin`); seed migrations; the `cmd/harness serve` Pi
handshake (`delivery/pilink`, [ADR-0008](../../docs/adr/0008-pi-client-integration-stdio.md));
and **model sync** — the first full pass through the layers: `serve` auto-applies migrations,
the `models`/`models_synced` wire frame (thin handler) drives the first CQRS use case
(`app/command.SyncModels`, behind the generic `decorator.CommandHandler` contract with a slog
logging decorator — [ADR-0012](../../docs/adr/0012-cqrs-application-layer.md)), persisting in one
transaction through `command.UnitOfWork` → `infra/sqlite/unitofwork` + `infra/sqlite/repo`
([ADR-0013](../../docs/adr/0013-idiomatic-go-layout-and-unit-of-work.md)); the model seed was
removed ([ADR-0011](../../docs/adr/0011-client-reported-model-catalog.md)). Deferred: the read
side (`query.ReadStore` + domain `Reader`s, with the first query), per-aggregate Writers for the
other aggregates, policy/Casbin **seeding**, MCP / events adapters, session create/resume +
config merge/validate, and all client **governance** (model handoff, permission gating, tool grants).
