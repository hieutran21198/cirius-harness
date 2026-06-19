# services/

Deployable backend services. **Each service is its own Go module** (own `go.mod`,
own deploy artifact). Services never import one another — communicate through API
contracts in `packages/go/<contract>` or over the wire (CLI command / MCP / events).

## Hexagonal + DDD (the dependency rule)

Each service follows hexagonal architecture with **dependencies pointing inward**
([ADR-0004](../docs/adr/0004-ports-and-adapters-topology.md)):

```
internal/
├── domain/        # aggregates, value objects, validation errors. Pure — stdlib only.
├── port/          # interfaces only
│   ├── inbound/    # driving ports — use-case interfaces
│   └── outbound/   # driven ports — repositories, Authorizer
├── application/   # use cases orchestrating the domain (load config, resolve agent, route, …).
└── adapter/       # infrastructure implementations (singular)
    ├── inbound/    # driving adapters — CLI, MCP, events
    └── outbound/   # driven adapters — DB stores, clients, authz enforcers
```

- `internal/adapter/*` **may import** `internal/application`, `internal/port`, and `internal/domain`.
- `internal/domain`, `internal/port`, and `internal/application` **must never import** `internal/adapter`.
- The domain depends on nothing outside the standard library; `port` imports only `domain`.
  Repository and authorizer **interfaces** live in `internal/port/outbound`; their
  **implementations** live in `internal/adapter/outbound`.
- **Define a port when the application requires the dependency**, not speculatively.

These rules and the layout are the workspace convention in
[docs/conventions/architecture.md](../docs/conventions/architecture.md), and the intent
behind the lint expectation in the root `AGENTS.md`. Until a formal import-boundary linter
is wired (an ADR), it is enforced by review.

## Conventions

- Per-service `.golangci.yml` (copy from a sibling). See `docs/conventions/go.md`.
- DB files live at `.cirius-harness/state/{service-name}.sqlite` (gitignored).
- No `replace` directives in committed `go.mod`; the root `go.work` links local modules.
