# Architecture conventions (idiomatic Go)

How every Go service is structured internally. Decided in
[ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md) (which supersedes the
port/adapter hexagon of [ADR-0004](../adr/0004-ports-and-adapters-topology.md)) and the CQRS app
layer of [ADR-0012](../adr/0012-cqrs-application-layer.md). The per-service layout tree lives in
[services/AGENTS.md](../../services/AGENTS.md).

## The two edges

A service has two kinds of edge. Both are named interfaces, **defined where they are consumed**
(the Go idiom), not in a central port tree:

- **Driving** — what *calls* the application (CLI, MCP, events, the Pi link). The interface it
  calls is declared **in the delivery package** that calls it (e.g. `pilink.Handler`); the
  composition root implements it and delegates to `app.Application`.
- **Driven** — what the application *calls* (a UnitOfWork, a ReadStore, an authorizer). The
  interface is declared **in the app package that consumes it** (e.g. `command.UnitOfWork`) and
  implemented under `infra/`.

### Adapters that span a process boundary

A delivery adapter can have a **client-side half outside the Go service** — in another language
or process. The Pi integration is the canonical example: the Go `pilink` delivery adapter
([ADR-0008](../adr/0008-pi-client-integration-stdio.md)) is *driven* by a TypeScript Pi
extension in `apps/pi-harness-extension/` over stdio. The service still only sees a delivery
adapter; the extension is that adapter's far end, built into the client's load path
([ADR-0010](../adr/0010-ts-build-pipeline-apps-to-pi-extensions.md)). One such app per
**citizen** (the governed client), kept out of `services/` and `packages/`.

## Layout

```
internal/
├── domain/        # aggregates, value objects, validation errors (stdlib only) +
│                  #   per-aggregate Reader/Writer interfaces (they speak only domain types)
├── app/           # use cases (CQRS — ADR-0012) + the app-owned driven ports, defined where used
│   ├── command/   #   write handlers + port.go (UnitOfWork, TransactionalUnitOfWork)
│   ├── query/     #   read handlers + (later) ReadStore
│   └── decorator/ #   generic CommandHandler/QueryHandler + cross-cutting decorators
├── delivery/      # driving adapters; each declares the app-usecase interface it calls
│   └── <name>/    #   e.g. pilink (Pi stdio link) — pilink.Handler is that driving port
└── infra/         # driven adapters implementing the app's driven ports
    └── <engine>/  #   e.g. sqlite/{repo, unitofwork, readstore} — repo holds the Reader/Writer
                   #   impls; unitofwork/readstore compose them into the app ports. Also casbin/.
```

## Rules

- **Dependency direction is inward**: `delivery`/`infra` → `app` → `domain`. `domain` imports
  only the standard library and its own packages; **nothing inner imports `delivery`/`infra`**.
  (Enforced by review until an import-boundary linter is wired — an ADR.)
- **Define an interface where it is consumed, when a use case needs it** — not speculatively.
  Driven ports live in the `app` package that calls them (`command.UnitOfWork`,
  `query.ReadStore`); driving ports live in the `delivery` package that calls them.
- **Per-aggregate `Reader`/`Writer` interfaces live in the domain** (e.g. `model.Writer`) — they
  reference only domain types. Commands mutate through a **UnitOfWork** (`DoTx` = one
  transaction); queries read through a **ReadStore**.

## Anti-patterns

- A central `internal/port/` tree, or an `internal/adapter/{inbound,outbound}` tree — interfaces
  belong with their consumer; adapters live under `delivery/` (driving) and `infra/` (driven).
- A fat application `Service` accumulating methods — one handler per use case (ADR-0012).
- Speculative interfaces with no implementation or caller (the old aggregate repo stubs).
- Any import from `domain`/`app` into `delivery`/`infra`.
