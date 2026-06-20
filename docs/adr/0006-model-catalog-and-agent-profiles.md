# 0006. First-class model catalog + immutable agent profiles

- **Status**: Superseded by ADR-0007
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: ADR-0002 (agent-carries-model); refines ADR-0005 (identity)
- **Superseded by**: ADR-0007

> **Superseded by [ADR-0007](0007-roles-and-per-session-model-binding.md):** the profile
> cluster is removed — an agent is a pure role, the model is bound per session on
> `session_agents.model_id`, and tools become a catalog. Body kept intact (append-only).

## Context

The model lived on the agent row (`agents.model`), and a session recorded only which agent
joined (`session_agents.agent_id`) — never which model it ran with. So a session read its
agent's model **live**: editing an agent's model retroactively rewrote every session that
ever used it, including finished ones, and the system could not answer "what did `council`
actually run with last Tuesday?". Separately, a model was a free-text string
(`"anthropic/claude-opus-4-7"`) duplicated across `agents.model` and every
`agent_fallbacks.model` row, with no place to attach model-level facts.

We want session runs to be **reproducible** (immune to later edits of the agent definition)
and the model to be a **first-class, normalized** entity.

## Decision

Restructure identity into a catalog, a pure agent, and an immutable binding.

- **`models` is a first-class catalog**: `(id PK, provider, model_id, enabled,
  UNIQUE(provider, model_id))`. The provider/model-id pair is named once and referenced by id.
- **The agent is pure identity/role**: `agents` keeps `name, responsibility, archetype,
  description, source, enabled` and **drops `model`**; tools and fallbacks no longer hang off
  the agent.
- **An `agent_profile` is an immutable, versioned binding** of an agent to its runtime
  config: a primary `model_id`, ordered fallback models (`profile_fallbacks`), and the tool
  set (`profile_tools`). `agent_profiles` carries `is_current` and `created_at`; a **partial
  unique index `(agent_id) WHERE is_current = 1`** enforces exactly one current profile per
  agent. **A profile is never edited in place** — changing an agent's model *inserts a new
  profile* (`is_current = 1`) and clears the old one's flag.
- **A session pins the profile it ran with**: `session_agents.profile_id` references
  `agent_profiles(id)` (nullable — model-less `prayer` has no profile). New sessions attach
  the agent's current profile; existing sessions keep their recorded profile, so history is
  frozen.

IDs are UUID v7 minted in the application/adapter, never the DB (per
[ADR-0005](0005-surrogate-uuid-v7-keys.md)); the seed supplies fixed UUID v7 literals for
models and profiles.

**Authorization is unchanged**: the Casbin principal stays the agent **name**, so
[ADR-0003](0003-authorization-casbin-abac.md) and `casbinauthz` are untouched.

## Consequences

- Positive: session runs are reproducible — editing an agent's model cannot rewrite past
  runs; the model is normalized and gains a home for model-level metadata; model swaps are
  auditable as a profile history; model-less `prayer` falls out naturally (an agent with no
  profile).
- Negative: reading an agent's effective config now needs joins across
  `agent_profiles` / `profile_tools` / `profile_fallbacks` / `models`; every model change
  mints a profile row; `session_agents.profile_id` is nullable.
- Neutral: the agent ↔ model link moves from one column to a small graph; the seed grows a
  `models` catalog and one current profile per model-having agent.

## Alternatives considered

- **Live reference** (the status quo) — session reads the agent's model directly. Rejected:
  editing an agent rewrites the history of finished sessions.
- **Editable shared profile** — one mutable profile per agent. Rejected: a session pointing
  at it would still see later edits — the same bug, one indirection deeper.
- **Per-message version pointer** — an immutable `agent_versions` row per edit referenced by
  each message. Heavier (version-minting on every edit, mid-session granularity). Deferred:
  `is_current` + per-session `profile_id` covers the between-sessions case we need now.

## References

- [ADR-0002](0002-persistence-and-migrations.md) — agent-carries-model, refined here.
- [ADR-0003](0003-authorization-casbin-abac.md) — authz principal unchanged.
- [ADR-0005](0005-surrogate-uuid-v7-keys.md) — UUID v7 identity rules these new aggregates follow.
- [conventions/persistence.md](../conventions/persistence.md),
  [specs/harness-data-model.md](../specs/harness-data-model.md),
  `.cirius-harness/00-system.yaml` (the authoring source the seed normalizes).
