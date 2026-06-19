# Specs

Feature and system design documents.

## When to write a spec

- The change spans **two or more modules**.
- The change reshapes a public contract (CLI command, MCP, events).
- The change has non-trivial migration / rollout concerns.

A spec is **not**:

- An ADR (one decision with tradeoffs) → write an ADR instead.
- A ticket (one task with acceptance criteria) → file a ticket instead.
- Per-service implementation notes → live in `services/<name>/AGENTS.md`.

## Naming

```
docs/specs/NNNN-kebab-title.md
```

Use a sequence number when ordering matters; drop it for evergreen reference specs.

## Shape (suggested)

```
# Title

- Status: Draft | Reviewed | Implemented | Archived
- Owner: <author>
- Reviewers: <names>
- Related ADRs: ADR-NNNN, ADR-MMMM

## Problem
## Goals / non-goals
## Design
## Rollout / migration
## Open questions
## References
```

Keep specs short. If a section sprawls, that section probably wants its own spec or ADR.
