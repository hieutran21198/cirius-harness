# 0019. Persist the council orchestration plan

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0017](0017-council-orchestration-model.md) (which deferred parsing/persisting
  the plan) and [ADR-0018](0018-harness-observability-logging-audit-session.md) (recording).

## Context

Council emits an `OrchestrationPlan` (ADR-0017), but the plan was **advisory and unpersisted** —
ADR-0017 explicitly deferred parsing or storing it, since there is no runtime executor yet
(Module 2). To make the plan first-class state a future executor can drive, the harness now
**captures an approved plan and persists it relationally**.

The output flow is also reframed. ADR-0017's prompt told council to emit "JSON first, then a
summary". In practice a human needs a **readable** artifact to review, and edits before it is
real. So:

1. `/council <ask>` → council emits a **human-readable Markdown** plan for review.
2. The human reviews and may iterate; council revises the Markdown.
3. On **human approval**, council emits the **final plan as a single JSON object**.
4. The Pi extension captures that JSON and submits it; the harness persists it.

## Decision

**A council plan, once a human approves it, is captured from council's turn output and persisted
as a relational `domain.Plan` aggregate.**

- **Contract.** The `OrchestrationPlan` / `PlannedTask` types (`internal/domain/plan_contract.go`)
  are enriched to be faithful — `Scope`, `Risk`, `Approval`, `Wave`, `Report` are structured and
  `PlannedTask.Inputs` is a list. They remain the single source: the prompt's JSON schema is
  rendered from them by reflection (now recursively, with a drift-guard test), and the inbound
  frame decodes into them. `Assignee` decoding is lenient (a bare-string `"explorer"` or the
  object `{agent, lens}`).
- **Aggregate.** A new encapsulated `domain.Plan` (ADR-0014) with typed ids, `NewPlan` (maps the
  contract, mints ids, derives status, validates the DAG — unique task refs, no dangling
  `depends_on`/wave refs), `RehydratePlan`, and `Snapshot`. Children: `PlanTask`, `PlanWave`,
  `PlanRisk`, `PlanApproval`.
- **Schema (fully relational).** Six tables: `plans` (intent/goal/status/session + the small
  leaves scope/assumptions/report as JSON), `plan_tasks`, `plan_risks`, `plan_approvals`,
  `plan_waves`, and `plan_wave_tasks` (the wave→task membership join). `plans.session_id` is a
  nullable FK (`ON DELETE SET NULL`); children cascade. Written through a new
  `domain.PlanWriter` on the `command.UnitOfWork`.
- **Use case + wire.** A decorated `SubmitPlan` command (audit + logging like the others) maps
  the contract to the aggregate and saves it in one transaction, idempotent on the plan id. A new
  additive frame `submit_plan` (`agent`, `client`, `plan`) → `plan_recorded` (`planId`,
  `taskCount`) carries it; the handler attaches the plan to the current session.
- **Client capture.** Council's prompt `OUTPUT` section drives the two-stage flow (Markdown review
  → JSON on approval). A `/council` opens an interaction that the Pi extension keeps in the council
  persona across **every** turn (via `before_agent_start`) — the proposal, the human's edits, and
  the "approved" turn all run as council, not just the first. The extension watches `agent_end`:
  the **first** (proposal) turn never submits — so the human always reviews before a plan lands —
  and from a later turn it captures the plan JSON (fenced ```` ```json ```` or any balanced object
  carrying a non-empty `tasks` array) and submits it. The interaction ends on a successful submit
  or session reset. Best-effort: review/iterate turns carry no plan and are skipped; a parse/submit
  failure is surfaced but never disrupts the session.

## Consequences

- **Positive**: an approved plan is durable, queryable, relational state — the task DAG, gates,
  risks, and waves are rows a future executor (Module 2) can drive. The contract stays the single
  source for prompt + decode. The frame is additive; the rest of the wire is unchanged.
- **Negative**: capture parses council's free-form output for a JSON block — robust to the
  review/iterate turns (no JSON → skipped) but dependent on council emitting valid JSON on
  approval. Six new tables widen the schema. Driving the plan ("the loop") remains deferred.
- **Neutral**: one new migration (`create_plans`); `plans.session_id` is null when no session was
  recorded (e.g. no client cwd). The plan's leaf structures (scope/assumptions/report) are JSON,
  not their own tables — they are only ever read back as part of a whole plan.

## Alternatives considered

- **Single JSON document** (one `plans` row holding the whole plan as JSON) — rejected: the DAG
  must be queryable for the executor, not re-parsed.
- **JSON-first output, captured on the first turn** — rejected: the human reviews a Markdown
  artifact and edits before it is persisted; only the approved JSON is captured.
- **An explicit `/council-approve` command as the submit trigger** — not taken: capture keys on
  council actually emitting the JSON (which it does post-approval), so no extra command is needed.
- **Execute the plan now** — deferred: the runtime multi-agent executor is Module 2
  ([ADR-0001](0001-harness-layout.md)).

## References

- [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (governed turn),
  [ADR-0017](0017-council-orchestration-model.md) (the plan contract this persists),
  [ADR-0018](0018-harness-observability-logging-audit-session.md) (session recording the plan
  attaches to), [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md) /
  [ADR-0014](0014-domain-encapsulation-single-package.md) (ports, encapsulation),
  [conventions/api.md](../conventions/api.md) (the new frames).
- `internal/domain/{plan,plan_contract,plan_writer,orchestration}.go`,
  `internal/app/command/submit_plan.go`, `internal/infra/sqlite/repo/plan.go`,
  `internal/delivery/pilink/{pilink,handler}.go` (frames + the harness-side handler),
  `migrations/…_create_plans.sql`, `apps/pi-harness-extension/src/index.ts`.
