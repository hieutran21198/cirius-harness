# 0007. Agents as roles, per-session model binding, polymorphic session environment

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: ADR-0006
- **Superseded by**: -
- **Refined by**: [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (an agent gains a
  harness-owned `persona` — its behaviour/system prompt; still no stored per-agent model)

> Refines [ADR-0005](0005-surrogate-uuid-v7-keys.md): a join table that carries an attribute
> (`session_agents.model_id`) may use a surrogate `id`; pure junctions (`agent_tools`) keep a
> composite PK.

## Context

[ADR-0006](0006-model-catalog-and-agent-profiles.md) bound an agent to its model through an
immutable, versioned `agent_profiles` aggregate (model + tools + fallbacks), pinned by
sessions for reproducibility. On review the **table structure itself** was judged wrong: the
profile cluster (`agent_profiles`, `profile_tools`, `profile_fallbacks`, the `is_current`
partial index) was heavy, and reading an agent's effective config meant a four-way join. We
want a simpler structure, and one that also makes room for the orchestration runtime (a
session runs somewhere — a container or a git worktree).

## Decision

Restructure around **roles bound to models at session time**, and a **polymorphic session
environment**.

- **No profiles.** Delete `agent_profiles`, `profile_tools`, `profile_fallbacks`, `is_current`.
- **An agent is a pure role.** `agents` holds identity/role only (name, archetype,
  responsibility, description, source, enabled). **The model is removed from the agent.**
- **The model is bound per session.** `session_agents.model_id` (nullable; references the
  `models` catalog) records which model played the role in that run. Which model an agent uses
  is a runtime decision, so a session is inherently reproducible — there is no shared agent
  field to edit retroactively.
- **Tools are a catalog.** A `tools` table is the capability vocabulary
  (read, grep, glob, list, edit, bash, webfetch, websearch); agents are granted tools through
  the `agent_tools` junction (composite PK `(agent_id, tool_id)`).
- **Sessions belong to a project and run in a polymorphic environment.** `sessions.project_id`
  scopes the run; `sessions.env_type ∈ {container, worktree, unset}` with `sessions.env_id`
  holding the environment's id (or null). `env_id` has **no foreign key** — it is validated in
  the domain. **`containers`** is a new execution environment, sibling to `worktrees`.
- **`models.model_id` is renamed `models.slug`** (the provider's model name) so that the name
  `model_id` everywhere else unambiguously means a foreign key into `models.id`.
- **Fallbacks are deferred** — not modeled this round; they would reattach to `session_agents`
  later if the orchestrator needs them.
- **`casbin_rule` stays adapter-owned** (ADR-0003) — not created or seeded by goose.

IDs are UUID v7 minted app-side (ADR-0005); the seed supplies fixed literals for models,
tools, and agents.

## Consequences

- Positive: the agent model is far simpler (no versioned profile, no 4-way join); a session's
  model history is inherently frozen on the membership; execution environments are pluggable
  (container or worktree) behind one polymorphic reference; the model catalog stays normalized.
- Negative: there is **no stored per-agent default model** — the per-agent model lines in
  `.cirius-harness/00-system.yaml` are not seeded into any column, so model assignment is
  entirely a runtime concern; fallbacks are unmodeled; `env_id` carries no referential
  integrity (the domain must guarantee it points at a real container/worktree).
- Neutral: tools move from a per-profile set to an agent-level catalog grant; `session_agents`
  gains a surrogate `id` because it now carries data.

## Alternatives considered

- **Keep ADR-0006 profiles** — rejected: the structure was the thing judged wrong.
- **Default model on the agent** (`agents.default_model_id`) — rejected for now: the user
  chose to make model purely a session-time binding; a default can be added later.
- **Two nullable FK columns** (`worktree_id`, `container_id`) + a CHECK instead of
  `env_type`/`env_id` — rejected to match the polymorphic shape specified, accepting the loss
  of FK integrity on the environment reference.

## References

- [ADR-0006](0006-model-catalog-and-agent-profiles.md) — superseded by this decision.
- [ADR-0002](0002-persistence-and-migrations.md), [ADR-0003](0003-authorization-casbin-abac.md)
  (casbin_rule ownership), [ADR-0005](0005-surrogate-uuid-v7-keys.md) (identity).
- [conventions/persistence.md](../conventions/persistence.md),
  [specs/harness-data-model.md](../specs/harness-data-model.md),
  `.cirius-harness/00-system.yaml`.
