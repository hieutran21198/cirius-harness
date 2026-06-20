# 0012. CQRS application layer (commands/queries + decorators)

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: the per-service `port/inbound` (named use-case interfaces) of [ADR-0004](0004-ports-and-adapters-topology.md)
- **Superseded by**: -

## Context

[ADR-0004](0004-ports-and-adapters-topology.md) placed driving (inbound) ports as **named
use-case interfaces** in `internal/port/inbound`, implemented by `internal/application`. The
first use case (model sync) realized that literally: one `inbound.ModelSyncer` interface and a
flat `application.Service` accumulating a method per use case.

Reviewed against the Three Dots Labs reference
([wild-workouts-go-ddd-example](https://github.com/ThreeDotsLabs/wild-workouts-go-ddd-example),
book [*Go with the Domain*](https://threedots.tech/go-with-the-domain/)), the flat service is
the shape that reference deliberately **avoids**: as use cases accumulate, a single `Service`
grows into a god-object with tangled dependencies, and a hand-named interface per use case is
boilerplate that a generic contract removes. wild-workouts instead models each use case as its
**own handler** behind a generic `CommandHandler`/`QueryHandler`, grouped in an
`Application{Commands, Queries}`, with cross-cutting concerns added by **decorators**.

## Decision

**The application layer is CQRS: one handler per use case, grouped in `Application`.**

```
internal/application/
├── app.go        # Application{ Commands; Queries } + New(ports…) wiring
├── command/      # write-side use cases — one file per command
├── query/        # read-side use cases (none yet)
└── decorator/    # generic CommandHandler/QueryHandler + cross-cutting decorators
```

- A use case is a **command** (writes) or **query** (reads). Each is: a plain command/query
  struct (the intent), a concrete unexported handler over the outbound ports it needs, an
  exported `XHandler` type aliased to the generic contract, and a `NewXHandler` constructor
  that applies the decorators. `Handle` is **pure business logic**.
- The **use-case contract is generic** — `decorator.CommandHandler[C, R]` /
  `QueryHandler[Q, R]` — so one decorator serves every handler. This **replaces named inbound
  port interfaces**: there is no `internal/port/inbound`. Driving adapters depend on the
  concrete `application.Application` and call `app.Commands.X.Handle(ctx, cmd)`.
- **Commands return a result** (`Handle(ctx, C) (R, error)`), unlike canonical CQRS where
  commands return only `error`. Harness commands acknowledge an outcome over the wire (the
  model-sync `{added, total}` ack), so a result-returning contract is the pragmatic default.
- **Cross-cutting concerns are decorators**, not inline code, applied in `NewXHandler` via
  `ApplyCommandDecorators`. Today the only decorator is **logging** (stdlib `log/slog` to
  stderr — matching how the harness already logs). **No metrics decorator** until a metrics
  backend exists.
- The `decorator` package lives **in-service** (mirroring wild-workouts' `internal/common/
  decorator`). When a second service needs the same contracts, promote it to
  `packages/go/cqrs`.

`internal/port/outbound` is **unchanged** — repositories and the authorizer remain named
driven ports. Only the *inbound* edge changes: the use-case contract is the generic handler,
not a per-use-case interface.

## Consequences

- **Positive**: each use case is a small, independently testable unit with exactly the
  dependencies it needs (no god-service); cross-cutting concerns are uniform and added in one
  place; the generic contract removes per-use-case interface boilerplate. The structure is the
  well-trodden wild-workouts shape, so it scales as the harness grows (sessions, agents, config
  merge, governance).
- **Negative**: more scaffolding than a flat service for a one-use-case layer; generics +
  decorators are indirection a reader must learn once. The inbound edge no longer has a named
  interface to grep for — you read `Application` to see the use cases.
- **Neutral**: contradicts ADR-0004's `port/inbound` location only; ADR-0004's inbound/outbound
  *adapter* topology, the inward dependency direction, and all outbound ports stand. ADR-0004
  is left intact (append-only); this ADR supersedes that one part.

## Alternatives considered

- **Keep the flat `application.Service`** — rejected: the fat-service anti-pattern the reference
  exists to avoid; couples unrelated use cases through one struct's dependency set.
- **Keep named inbound port interfaces (`ModelSyncer`) over the handlers** — rejected: the
  generic `CommandHandler[C, R]` already *is* the use-case interface; a second hand-named one
  per use case is redundant layering.
- **Canonical error-only commands + a separate query for the ack** — rejected: two application
  calls to serve one wire frame whose response (`added`) is inherently a write-time fact.
- **Logging + metrics decorators up front** (full wild-workouts) — rejected: metrics would be a
  no-op client with no backend; add the decorator when one exists (the seam is already there).

## References

- [ADR-0004](0004-ports-and-adapters-topology.md) (ports & adapters topology, partly superseded
  here), [ADR-0011](0011-client-reported-model-catalog.md) (the model-sync use case this
  restructures).
- [conventions/architecture.md](../conventions/architecture.md) — the workspace layout rule.
- Three Dots Labs: [wild-workouts-go-ddd-example](https://github.com/ThreeDotsLabs/wild-workouts-go-ddd-example),
  [*Go with the Domain*](https://threedots.tech/go-with-the-domain/) (CQRS + decorator chapters).
