# Glossary

Canonical vocabulary for the platform. When two specs disagree on what a term means, **this file wins** - and the specs are out of date.

Every term should map to a real type when one exists. If a term has no code representation, that's a hint the abstraction may be missing.

---

## Orchestration runtime

The live counterpart to the declarative agent team — *work actually happening*.

- **Project** — a codebase the harness operates on, keyed by name; a monorepo is a
  `kind`. Type: `project.Project` (`services/harness/internal/domain/project`).
- **Worktree** — an isolated git working copy belonging to a project, keyed by its
  absolute path; concurrent worktrees are the substrate for parallel work. Type:
  `worktree.Worktree` (`services/harness/internal/domain/worktree`).
- **Session** — one run of the harness inside a worktree, with a lifecycle
  (pending → running → completed/failed/cancelled), identified by a UUID v7. Type:
  `session.Session` (`services/harness/internal/domain/session`).
- **Membership** — the join recording which agents joined a session (and whether
  they are still active). Type: `session.Member`, persisted in `session_agents`.

## Agent team (declarative)

The team definition — *who can work* — independent of any running session.

- **Agent** — one member of the harness team, keyed by name; the authorization principal.
  Type: `agent.Agent` (`services/harness/internal/domain/agent`).
- **Archetype** — an agent's purpose-level operating style: `communicator`,
  `principle-driven`, `utility-runner`, or `none` (model-less). Maps to a model family
  (see [research](../research/model-families.md)). Type: `agent.Archetype`.
- **Tool** — a capability an agent may use (read, grep, edit, bash, …). Type: `agent.Tool`.
- **Source** — where an agent definition came from: `system` (default) or `user`
  (workspace overlay). Type: `agent.Source`.

## Authorization

Per [ADR-0003](../adr/0003-authorization-casbin-abac.md) — Casbin ABAC, policy in the DB.

- **Principal** — the subject a decision is made about; here, the **agent name**.
- **Action** — an authorizable capability (read, edit, bash, webfetch, websearch). Type:
  `authz.Action`.
- **Decision** — the three-valued outcome: `allow`, `ask`, or `deny`. Type:
  `authz.Decision`.
- **Authorizer** — the outbound port that resolves (principal, resource, action) → Decision.
  Type: `outbound.Authorizer` (`internal/port/outbound`); Casbin implementation in
  `internal/adapter/outbound/casbinauthz`.
- **Policy** — one authorization rule, stored as a `casbin_rule` row (adapter-owned),
  carrying its own decision in the `dec` field.
