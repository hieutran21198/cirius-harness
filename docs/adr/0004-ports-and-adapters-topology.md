# 0004. Ports & adapters topology (inbound/outbound)

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: ADR-0001 layout (in part); the port-location language of ADR-0002 (§repository pattern) and ADR-0003 (§authorizer location)
- **Superseded by**: -

## Context

[ADR-0001](0001-harness-layout.md) set a coarse hexagonal layout: a single plural
`internal/adapters/` directory, with repository **ports defined inside the domain**
(`agent.Agents`, `authz.Authorizer`, …). That arrangement names *adapters* but not the two
kinds of hexagon edge — **driving (inbound)**, what calls the application, versus **driven
(outbound)**, what the application calls. As the service grows past one adapter
(`casbinauthz`) toward CLI/MCP/event entrypoints and GORM stores, the missing distinction
makes "where does this interface/implementation go?" ambiguous, and it leaves driven ports
co-located with aggregates that have no business depending on persistence shapes.

## Decision

We will give each service an explicit inbound/outbound topology:

```
internal/
├── domain/        # aggregates, value objects, validation errors. Pure — stdlib only.
├── port/          # interfaces (the hexagon's edges)
│   ├── inbound/    # driving ports — application/use-case interfaces
│   └── outbound/   # driven ports — repositories, Authorizer
├── application/   # use cases orchestrating the domain (future)
└── adapter/       # infrastructure implementations (singular)
    ├── inbound/    # driving adapters — CLI, MCP, events
    └── outbound/   # driven adapters — DB stores, clients, authz enforcers
```

- **`port`** and **`adapter`** are both **singular**, each split into **`inbound`** and
  **`outbound`**.
- **Outbound** holds driven ports (the aggregate repositories — `Agents`, `Projects`,
  `Sessions`, `Worktrees` — and `Authorizer`) and their infra adapters (the Casbin enforcer
  today; GORM stores next).
- **Inbound** holds driving ports (application/use-case interfaces) and their adapters
  (CLI, MCP, events); populated once the application layer lands.
- The **domain** keeps only aggregates, value objects, and validation errors. Repository and
  authorizer interfaces move out of `internal/domain` into `internal/port/outbound`.
- **A port is provided when the application requires the dependency**, not preemptively.
- Dependency direction is inward: `adapter → application → port → domain`. `port` imports
  only `domain`; `domain` imports only the standard library; nothing inner imports `adapter`.

## Consequences

- Positive: each new interface/implementation has one obvious home; the driving/driven edge
  is named in the tree; the domain no longer carries persistence-facing interfaces.
- Negative: deeper nesting, and repository ports are no longer co-located with the aggregate
  they read/write — you follow an import to `port/outbound` to see the contract.
- Neutral: `inbound` packages exist as documented placeholders until the application layer
  defines its first use case; the relocation is a pure move (signatures unchanged).

## Alternatives considered

- **Ports in the domain + a flat plural `adapters/`** (ADR-0001 / DDD-classic) — Rejected:
  names adapters but not the inbound/outbound edge, and couples the domain to driven ports.
- **Application layer owns the ports, no inbound/outbound split** — Rejected: still leaves
  "driving vs driven" unnamed, which is exactly the distinction that guides placement.

## References

- [ADR-0001](0001-harness-layout.md), [ADR-0002](0002-persistence-and-migrations.md),
  [ADR-0003](0003-authorization-casbin-abac.md) — the layout and port-location statements
  this ADR refines.
- [services/AGENTS.md](../../services/AGENTS.md) — the authoritative per-service layout.
- [conventions/architecture.md](../conventions/architecture.md) — the workspace-wide rule.
