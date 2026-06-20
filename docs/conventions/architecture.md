# Architecture conventions (hexagonal)

How every Go service is structured internally. Decided in
[ADR-0004](../adr/0004-ports-and-adapters-topology.md); the per-service layout tree lives in
[services/AGENTS.md](../../services/AGENTS.md).

## The two edges

A hexagon has two kinds of edge. Name them:

- **Inbound (driving)** — what *calls* the application: CLI, MCP, events. The interface it
  calls is an **inbound port**; the thing that calls it is an **inbound adapter**.
- **Outbound (driven)** — what the application *calls*: repositories, the authorizer,
  external clients. The interface is an **outbound port**; the implementation is an
  **outbound adapter**.

### Adapters that span a process boundary

An inbound adapter can have a **client-side half outside the Go service** — in another
language or process. The Pi integration is the canonical example: the Go `pilink` inbound
adapter ([ADR-0008](../adr/0008-pi-client-integration-stdio.md)) is *driven* by a TypeScript
Pi extension that lives in `apps/pi-harness-extension/` and talks to it over stdio. The
service still only sees an inbound adapter; the extension is that adapter's far end, built
into the client's load path ([ADR-0010](../adr/0010-ts-build-pipeline-apps-to-pi-extensions.md)).
One such app per **citizen** (the governed client), kept out of `services/` and `packages/`.

## Layout

```
internal/
├── domain/        # aggregates, value objects, validation errors. Pure — stdlib only.
├── port/          # interfaces only
│   ├── inbound/    # driving ports (use-case interfaces)
│   └── outbound/   # driven ports (repositories, Authorizer)
├── application/   # use cases orchestrating the domain
└── adapter/       # implementations (singular)
    ├── inbound/    # driving adapters (CLI, MCP, events)
    └── outbound/   # driven adapters (DB stores, clients, authz enforcers)
```

`port` and `adapter` are **singular**; the plurality lives in `inbound`/`outbound`.

## Rules

- **Dependency direction is inward**: `adapter → application → port → domain`. `port`
  imports only `domain`; `domain` imports only the standard library; **nothing inner imports
  `adapter`**. (Enforced by review until an import-boundary linter is wired — an ADR.)
- **Define a port when the application requires the dependency** — not speculatively. An
  outbound port earns its place when a use case (or an existing adapter, like the Casbin
  authorizer) actually needs it.
- **The domain holds no interfaces to infrastructure.** Repository and authorizer ports live
  in `port/outbound`, never in `internal/domain`.

## Anti-patterns

- A plural `internal/adapters/` directory, or `internal/ports/` — use the singular
  `adapter`/`port` with `inbound`/`outbound` subpackages.
- Repository or authorizer interfaces declared inside `internal/domain`.
- Any import from `domain`/`port`/`application` into `adapter`.
