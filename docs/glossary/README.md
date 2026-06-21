# Glossary

Canonical vocabulary for the platform. When two specs disagree on what a term means, **this file wins** - and the specs are out of date.

Every term should map to a real type when one exists. If a term has no code representation, that's a hint the abstraction may be missing.

---

## Orchestration runtime

The live counterpart to the declarative agent team — *work actually happening*.

- **Project** — a codebase the harness operates on; identified by a UUID v7 surrogate, with
  `name` as its unique business key; a monorepo is a `kind`. Type: `domain.Project`
  (`services/harness/internal/domain`).
- **Worktree** — an isolated git working copy belonging to a project; identified by a UUID v7
  surrogate, with its absolute `path` as the unique business key; concurrent worktrees are
  the substrate for parallel work. Type: `domain.Worktree`
  (`services/harness/internal/domain`).
- **Container** — an execution environment belonging to a project, sibling to a worktree; a
  session may run in one. Type: `domain.Container`
  (`services/harness/internal/domain`).
- **Environment** — where a session runs: a container, a worktree, or `unset` (not yet
  provisioned). Modeled on the session as `env_type` + a polymorphic `env_id`
  ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)).
- **Session** — one run of the harness, scoped to a **project** and executed in an
  **environment** (container | worktree | unset), with a lifecycle
  (pending → running → completed/failed/cancelled), identified by a UUID v7. Type:
  `domain.Session` (`services/harness/internal/domain`).
- **Membership** — the join recording which agents joined a session and, per
  [ADR-0007](../adr/0007-roles-and-per-session-model-binding.md), the **model** that agent ran
  with (`model_id`, empty for model-less `prayer`). Type: `domain.Member`, persisted in
  `session_agents`.

## Agent team (declarative)

The team definition — *who can work* — independent of any running session.

- **Agent** — one member of the harness team and a pure **role**; identified by a UUID v7
  surrogate, with `name` as its unique business key and the authorization principal (the
  Casbin subject). It holds **no model** — which model plays the role is bound per session
  (`session_agents.model_id`); it is granted **tools** from the catalog via `agent_tools`
  ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)). Type: `domain.Agent`
  (`services/harness/internal/domain`).
- **Model** — a provider/model-id in the first-class catalog (e.g. `anthropic/claude-opus-4-7`,
  stored as provider + `slug`), referenced by id from a session membership. Type: `domain.Model`
  (`services/harness/internal/domain`).
- **Archetype** — an agent's purpose-level operating style: `communicator`,
  `principle-driven`, `utility-runner`, or `none` (model-less). Maps to a model family
  (see [research](../research/model-families.md)). Type: `domain.Archetype`.
- **Tool** — an entry in the capability catalog (read, grep, edit, bash, …), granted to agents
  via `agent_tools`. Type: `domain.Tool` (`services/harness/internal/domain`).
- **Source** — where an agent definition came from: `system` (default) or `user`
  (workspace overlay). Type: `domain.Source`.

## Authorization

Per [ADR-0003](../adr/0003-authorization-casbin-abac.md) — Casbin ABAC, policy in the DB.

- **Principal** — the subject a decision is made about; here, the **agent name**.
- **Action** — an authorizable capability (read, edit, bash, webfetch, websearch). Type:
  `domain.Action`.
- **Decision** — the three-valued outcome: `allow`, `ask`, or `deny`. Type:
  `domain.Decision`.
- **Authorizer** — resolves (principal, resource, action) → Decision. Concrete Casbin
  implementation in `internal/infra/casbin`; its interface is defined by the consuming use case
  when one lands ([ADR-0013](../adr/0013-idiomatic-go-layout-and-unit-of-work.md)).
- **Policy** — one authorization rule, stored as a `casbin_rule` row (adapter-owned),
  carrying its own decision in the `dec` field.
