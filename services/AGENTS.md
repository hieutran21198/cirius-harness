# services/

Deployable backend services. **Each service is its own Go module** (own `go.mod`,
own deploy artifact). Services never import one another — communicate through API
contracts in `packages/go/<contract>` or over the wire (CLI command / MCP / events).

## Layers + DDD (the dependency rule)

Each service uses idiomatic-Go layers with **dependencies pointing inward**
([ADR-0013](../docs/adr/0013-idiomatic-go-layout-and-unit-of-work.md), superseding the
port/adapter hexagon of ADR-0004):

```
internal/
├── domain/        # aggregates, value objects, validation errors (stdlib only) +
│                  #   per-aggregate Reader/Writer interfaces (speak only domain types).
├── app/           # CQRS use cases + the app-owned driven ports, defined where consumed.
│   ├── command/   #   write handlers + port.go (UnitOfWork, TransactionalUnitOfWork).
│   ├── query/     #   read handlers + (later) ReadStore.
│   └── decorator/ #   generic CommandHandler/QueryHandler + cross-cutting decorators.
├── delivery/      # driving adapters; each declares the app-usecase interface it calls.
└── infra/         # driven adapters implementing the app's driven ports (DB, authz, clients).
```

- `internal/delivery/*` and `internal/infra/*` **may import** `internal/app` and `internal/domain`.
- `internal/domain` and `internal/app` **must never import** `internal/delivery` or `internal/infra`.
- The domain depends on nothing outside the standard library (and its own packages).
- **Define an interface where it is consumed, when a use case needs it** — driven ports in the
  `app` package that calls them (`command.UnitOfWork`); driving ports in the `delivery` package
  that calls them (`pilink.Handler`). No central `port/` tree, no `adapter/` tree.

These rules and the layout are the workspace convention in
[docs/conventions/architecture.md](../docs/conventions/architecture.md), and the intent
behind the lint expectation in the root `AGENTS.md`. Until a formal import-boundary linter
is wired (an ADR), it is enforced by review.

## Conventions

- Per-service `.golangci.yml` (copy from a sibling). See `docs/conventions/go.md`.
- DB files live at `.cirius-harness/state/{service-name}.sqlite` (gitignored).
- No `replace` directives in committed `go.mod`; the root `go.work` links local modules.
