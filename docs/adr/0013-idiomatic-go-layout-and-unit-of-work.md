# 0013. Idiomatic-Go layout (domain/app/delivery/infra) + Unit of Work

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: [ADR-0004](0004-ports-and-adapters-topology.md) (the whole inbound/outbound
  port & adapter topology); refines [ADR-0012](0012-cqrs-application-layer.md) (what handlers depend on)
- **Superseded by**: -

## Context

[ADR-0004](0004-ports-and-adapters-topology.md) gave the service a `port/{inbound,outbound}` +
`adapter/{inbound,outbound}` hexagon. In practice that is **too much ceremony** for Go: a central
`port/` tree of interfaces, away from where they are used, plus a parallel `adapter/` tree. Go's
idiom is the opposite — **define an interface where it is consumed**, keep it small, and let
implementations satisfy it structurally.

[ADR-0012](0012-cqrs-application-layer.md) moved the app layer to CQRS but still injected
per-aggregate repository interfaces (`outbound.Models`, all `Get/List/Save/Delete`) into handlers.
The user wants the CQRS split carried to the persistence edge — commands mutate through a
transactional **Unit of Work**, queries read through a **ReadStore** — and the package layout
flattened to idiomatic Go.

## Decision

**Layout is `internal/{domain, app, delivery, infra}`** — no `port/`, no `adapter/`.

```
internal/
├── domain/        aggregates, value objects, validation errors (stdlib only) +
│                  per-aggregate Reader/Writer interfaces (e.g. model.Writer)
├── app/           use cases (CQRS) + the app-owned driven ports, defined where consumed
│   ├── command/   write handlers + port.go (UnitOfWork, TransactionalUnitOfWork)
│   ├── query/     read handlers + (later) ReadStore        [empty until the first query]
│   └── decorator/ generic CommandHandler/QueryHandler + logging
├── delivery/      driving adapters; each defines the app-usecase interface it calls
│   └── pilink/    Pi stdio link — pilink.Handler is that driving port
└── infra/         driven adapters implementing the app's driven ports
    ├── sqlite/    repo/ (GORM Reader/Writer impls) + unitofwork/ (command.UnitOfWork)
    │              + readstore/ (query.ReadStore — added with the first query)
    └── casbin/    Casbin authorizer (concrete)
```

- **The app owns its driven ports, defined in the consuming package.** `command.UnitOfWork`
  lives in `app/command`; `query.ReadStore` will live in `app/query`. No standalone port tree.
- **Driving ports are consumer-defined interfaces in the delivery layer.** `pilink.Handler`
  (in `delivery/pilink`) declares what the transport calls on the app; the composition root
  implements it and delegates to `app.Application`.
- **Per-aggregate `Reader`/`Writer` interfaces live in the domain package** (e.g. `model.Writer`),
  because they speak only domain types. This reverses ADR-0004's "no interfaces in domain" and
  its `Get/List/Save/Delete` repo shape.
- **Commands mutate through a `UnitOfWork`.** `TransactionalUnitOfWork` exposes the domain
  `Writer`s; `UnitOfWork` adds `DoTx(ctx, func(ctx, TransactionalUnitOfWork) error)` — one GORM
  transaction, committing on nil and rolling back on error. SyncModels runs entirely in one `DoTx`.
- **The infra adapter is layered**: `infra/sqlite/repo` holds the GORM Reader/Writer
  implementations (bound to a `*gorm.DB`); `infra/sqlite/unitofwork` and (later)
  `infra/sqlite/readstore` compose those repos to implement the app's `command.UnitOfWork` /
  `query.ReadStore`. `DoTx` builds the repo writers over the open transaction.
- **The read side (`ReadStore` + domain `Reader`s) is deferred** until the first query. Until
  then the sync ack's `total` is counted in-transaction via `model.Writer.Count`.
- **Define a port when a use case consumes it.** The 6 unused aggregate repo stubs
  (agents/projects/sessions/worktrees/containers/tools), the `Authorizer` interface, and
  `ErrNotFound` were **removed** — reintroduced (as domain Reader/Writer, or the consumer's
  interface) when a use case needs them. `infra/casbin` exposes a concrete `Decide` meanwhile.

## Consequences

- **Positive**: less ceremony; interfaces sit next to their consumer (idiomatic Go), small and
  purpose-built; mutations are transactional by construction; no speculative interfaces. The
  delivery/infra split names the two adapter kinds without a parallel port tree.
- **Negative**: a third revision of the persistence/topology rule (ADR-0004 → 0012 → 0013);
  contributors must read `app/command/port.go` (not a central `port/`) to find the driven ports;
  domain-located Reader/Writer means the domain package now carries interfaces (mitigated: they
  reference only domain types).
- **Neutral**: behavior is unchanged; the move was mechanical. ADR-0012's CQRS
  `Application{Commands,Queries}` + handler-per-use-case + decorators stand — only what the
  handlers depend on changed (app-owned ports, not a `port/outbound`).

## Alternatives considered

- **Keep ADR-0004's port/adapter hexagon** — rejected: the central port tree is the ceremony
  the user wants gone.
- **A single `app/port` package for all driven ports** — rejected in favor of consumer-defined
  interfaces in `app/command` / `app/query` (least ceremony, most idiomatic).
- **Build ReadStore now** — deferred: no query consumes it yet; building it would be speculative
  (the rule above). It lands with the first read use case.
- **Per-aggregate `UpdateXxx` closure repositories** (wild-workouts trainer style) instead of a
  UnitOfWork — rejected: the user prefers an explicit UoW that scales to multi-aggregate writes
  (e.g. session + session_agents).

## References

- [ADR-0004](0004-ports-and-adapters-topology.md) (superseded), [ADR-0012](0012-cqrs-application-layer.md)
  (CQRS app layer, refined here), [ADR-0011](0011-client-reported-model-catalog.md) (the model-sync
  use case), [ADR-0008](0008-pi-client-integration-stdio.md) (the Pi stdio link in delivery/pilink).
- [conventions/architecture.md](../conventions/architecture.md) — the workspace layout rule.
