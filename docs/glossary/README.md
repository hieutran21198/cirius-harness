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
  `session_agents`. Recorded when an agent is resolved in a session
  ([ADR-0018](../adr/0018-harness-observability-logging-audit-session.md)).
- **Audit log** — the append-only record of what the harness did: one **Event** per command
  (kind, actor, ok/error status, message), written through a command audit decorator. Persisted
  in the `events` table; distinct from the **ephemeral stderr/file logs** and from scribe's
  distilled lessons. Type: `domain.Event`
  ([ADR-0018](../adr/0018-harness-observability-logging-audit-session.md)).
- **Actor** (audit) — who caused a command (the reporting client; `""` if unknown). Carried on
  the context (`internal/app/appctx`) by the delivery layer and recorded on each Event. Distinct
  from the authorization **Principal** (the agent name).
- **Plan** (persisted) — a council orchestration plan **after a human approved it**, captured from
  council's turn output and stored as a relational aggregate: the request analysis (intent, goal,
  scope, assumptions, risks), the task DAG (**plan task**s grouped into **plan wave**s via a
  membership join), the human-approval gates, and a closing report. Attached to the session it was
  produced in; identified by a UUID v7. A future executor (Module 2) drives it; the harness only
  records it ([ADR-0019](../adr/0019-persist-council-orchestration-plan.md)). Type: `domain.Plan`
  (root), `domain.PlanTask` / `domain.PlanWave` / `domain.PlanRisk` / `domain.PlanApproval`
  (children), persisted in `plans` / `plan_tasks` / `plan_waves` / `plan_wave_tasks` / `plan_risks`
  / `plan_approvals`. Distinct from the **Orchestration plan** contract (the wire/prompt shape it
  is built from).
- **Plan run** — the execution state over an approved **Plan** when it is driven: the drive's
  status (driving → done/cancelled) and the per-task progress the client-coordinated drive reports.
  The Plan stays an immutable spec; the run records what happened, so an approved plan is never
  rewritten. One live run per plan; identified by a UUID v7. Type: `domain.PlanRun` (root) with
  child `domain.TaskRun`, persisted in `plan_runs` / `plan_task_runs`
  ([ADR-0021](../adr/0021-drive-the-council-plan.md)).
- **Task run** — one task's progress within a **plan run**: its status (pending → running →
  done/failed/skipped) and a short result summary. Type: `domain.TaskRun`, persisted in
  `plan_task_runs`.
- **Drive** — client-coordinated execution of a plan: the harness serves the plan (`get_plan`) and
  records progress (`report_run`); a coordinator in the Pi client spawns one headless `pi` worker
  per task, sequenced by the plan's waves and dependencies, gating high-risk tasks on human
  approval. The harness governs and records, it does not schedule or run workers — so a drive is
  still Module 1; a harness-owned scheduler would be Module 2
  ([ADR-0021](../adr/0021-drive-the-council-plan.md), [ADR-0001](../adr/0001-harness-layout.md)).

## Agent team (declarative)

The team definition — *who can work* — independent of any running session.

- **Agent** — one member of the harness team and a pure **role**; identified by a UUID v7
  surrogate, with `name` as its unique business key and the authorization principal (the
  Casbin subject). It holds **no model** — which model plays the role is bound per session
  (`session_agents.model_id`); it is granted **tools** from the catalog via `agent_tools`
  ([ADR-0007](../adr/0007-roles-and-per-session-model-binding.md)). Type: `domain.Agent`
  (`services/harness/internal/domain`).
- **Persona** — an agent's harness-owned behaviour, modelled as a **structured profile** (identity,
  mission, principles, fixed output sections, boundaries, effort-scaling rule) that renders to the
  **system prompt** the control plane hands the client to run a turn as that agent (e.g. council
  weighing a request into a strategy plan; the implementer executing a task in-role). It is
  harness-owned **code** — a `domain.Persona` value resolved by name via `domain.PersonaFor`,
  **not** stored in the DB or workspace config. Every **working** agent has one; only the
  model-less `prayer` (archetype `none`) has none. The prompt's *style* follows the agent's
  **archetype** (communicator → checklisted; principle-driven → concise principles; utility-runner
  → terse). Distinct from the **model** (bound per session) and **permissions** (Casbin). Resolved
  over the `resolve_agent` frame and run as a one-shot governed turn
  ([ADR-0016](../adr/0016-harness-owned-agent-persona-governed-turn.md),
  [ADR-0020](../adr/0020-specialist-agent-personas.md)). Type: `domain.Persona` (an interface;
  `services/harness/internal/domain`).
- **Council profile** — council's persona: a typed **orchestration model** (intents, the 7 task
  dimensions, the category taxonomy, the agent-capability roster, routing rules, the flow pipeline,
  quality gates, and the assignment factors) rendered to council's system prompt
  ([ADR-0017](../adr/0017-council-orchestration-model.md)). Type: `domain.CouncilProfile`.
- **Agent profile** — the shared persona for the working specialists (planner, implementer,
  researcher, explorer, reviewer, scribe): identity, mission, principles, fixed output sections,
  boundaries, and an effort-scaling rule, rendered to a system prompt in the style its **archetype**
  dictates (communicator → checklisted; principle-driven → concise principles; utility-runner →
  terse). Implements `domain.Persona`; the specialist counterpart to council's richer **Council
  profile** ([ADR-0020](../adr/0020-specialist-agent-personas.md)). Type: `domain.AgentProfile`.
- **Category** — a kind of work a task falls into (explore, research, architect, plan, implement,
  test, review, security, performance, docs, migration, devops, integrate). Richer than the team;
  council routes several categories onto one agent ([PDR-0002](../pdr/0002-agent-team-composition.md)).
  Type: `domain.Category`.
- **Lens** — a focus-mode an agent is summoned in so the team stays lean rather than minting a new
  agent (e.g. `reviewer` in a *security* lens, `implementer` in a *tester* lens). Advisory guidance,
  not a permission change ([PDR-0002](../pdr/0002-agent-team-composition.md)).
- **Agent capability** — council's model of one team agent: what it handles, its tools, cost/speed,
  reliability, risk tolerance, permissions, and lenses; the input to capability-based routing. Type:
  `domain.AgentCapability`.
- **Quality gate** — one rung of the four-gate human-in-the-loop model (advisory → validating →
  blocking → escalating): how much oversight a task needs before it proceeds; high-risk work blocks
  on human approval. Type: `domain.QualityGate`.
- **Orchestration plan** — council's output **contract**: an intent, a dimensioned analysis, and a
  task DAG (per-task assignee+lens, expected output, dependencies, wave, definition-of-done, gate,
  risk). Council presents it as Markdown for human review, then — on approval — emits it as JSON
  matching this contract ([ADR-0017](../adr/0017-council-orchestration-model.md),
  [ADR-0019](../adr/0019-persist-council-orchestration-plan.md)). It is the single source for both
  the prompt schema (rendered by reflection) and the inbound decode. Type:
  `domain.OrchestrationPlan` / `domain.PlannedTask`.
- **Model** — a provider/model-id in the first-class catalog (e.g. `anthropic/claude-opus-4-7`,
  stored as `client` + provider + `slug`), referenced by id from a session membership. Model
  names are **client-specific**, so the natural key is `(client, provider, slug)`
  ([ADR-0015](../adr/0015-client-aware-model-catalog.md)). Type: `domain.Model`
  (`services/harness/internal/domain`).
- **Client** (a.k.a. **citizen**) — the AI coding client the harness governs, paired 1:1 with its
  own child-harness ([ADR-0009](../adr/0009-deployment-topology-per-client-harness.md)); e.g.
  `pi`, `opencode`. It reports its enabled models, named in its own registry's vocabulary. Type:
  `domain.ClientKind`.
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
