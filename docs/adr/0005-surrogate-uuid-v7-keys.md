# 0005. Surrogate UUID v7 primary keys for all aggregates

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: ADR-0002 identity decision (natural keys)
- **Superseded by**: -

> **Extended by [ADR-0006](0006-model-catalog-and-agent-profiles.md):** adds the `models` and
> `agent_profiles` aggregates, which follow the same UUID v7 surrogate-identity rules.
>
> **Refined by [ADR-0007](0007-roles-and-per-session-model-binding.md):** a join table that
> carries an attribute (`session_agents.model_id`) may take a surrogate `id`; pure junctions
> (`agent_tools`) keep a composite PK.

## Context

[ADR-0002](0002-persistence-and-migrations.md) chose a **natural key where stable** — agent
`name`, project `name`, worktree `path` — and a UUID v7 surrogate only for `sessions`, which
has no natural key. In practice natural keys are renamable identity: rename an agent or move
a worktree and every referencing row (`agent_tools`, `session_agents`, `worktrees.project`,
…) has to change with it, and the identity model differs table-by-table. We want one uniform,
rename-safe identity across every aggregate.

## Decision

Every aggregate table has a **UUID v7 surrogate primary key**: `id TEXT PRIMARY KEY NOT NULL`
(the `NOT NULL` also closes SQLite's quirk that a bare `TEXT PRIMARY KEY` permits NULLs).
This is the **full-surrogate** form — the UUID is both the PK **and** the reference target:

- Natural keys (`name`, `path`, `root_path`) are demoted to `UNIQUE NOT NULL` attributes —
  still the human/business key (authoring, lookup, authz), but not the identity.
- Foreign keys and cross-aggregate references use the **UUID**: `agent_tools.agent_id`,
  `agent_fallbacks.agent_id`, `worktrees.project_id`, `sessions.worktree_id`,
  `session_agents.{session_id,agent_id}`; domain aggregates carry an `ID`, and references
  become `worktree.ProjectID`, `session.WorktreeID`, `session.Member.AgentID`.
- IDs are generated in the **application/adapter** via `uuid.NewV7()`, never by the DB
  (SQLite generates nothing). Seeds supply **fixed UUID v7 literals**.

**Authorization is unchanged**: the Casbin principal stays the agent **name** (a UNIQUE
attribute), so [ADR-0003](0003-authorization-casbin-abac.md) and `casbinauthz` are untouched.

## Consequences

- Positive: relationships are rename-safe; identity is uniform across all aggregates; UUID
  v7 is time-ordered so it indexes well as a PK.
- Negative: rows are no longer human-readable (a join is needed to see that an `agent_id` is
  "council"); every repository port, seed, and use case must carry ids; there is no DB-side
  id generation, so the application must mint the id before every insert.
- Neutral: natural keys still exist as UNIQUE columns, so name/path lookups and the authz
  principal keep working.

## Alternatives considered

- **Natural keys** (the superseded ADR-0002 decision) — readable rows, no generation needed.
  Rejected: renames rewrite referencing rows; identity is non-uniform.
- **Storage-only surrogate** — UUID PK, but FKs/refs stay by the natural key. Rejected: the
  point of the change is to decouple references from renamable keys; a half-measure keeps the
  coupling.

## References

- [ADR-0002](0002-persistence-and-migrations.md) — the identity decision this refines.
- [ADR-0004](0004-ports-and-adapters-topology.md) — where the ports being re-keyed live.
- [conventions/persistence.md](../conventions/persistence.md),
  [specs/harness-data-model.md](../specs/harness-data-model.md).
