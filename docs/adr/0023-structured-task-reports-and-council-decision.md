# 0023. Structured task reports and a council decision stage

- **Status**: Accepted
- **Date**: 2026-06-23
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0021](0021-drive-the-council-plan.md) (which drove the plan but recorded only a
  free-text per-task summary) and [ADR-0020](0020-specialist-agent-personas.md) (the worker
  personas now also carry a report contract). **Relates to**
  [ADR-0017](0017-council-orchestration-model.md) / [ADR-0019](0019-persist-council-orchestration-plan.md)
  (the plan/decision symmetry).

## Context

When a council plan is driven ([ADR-0021](0021-drive-the-council-plan.md)), each task runs as a
headless `pi` worker and the coordinator reports back only a free-text `summary` (first line, ≤500
chars) recorded on the `TaskRun`. The worker's full output is never persisted, and there is **no
schema** on what a task returns. There is also **no post-execution council stage**: council only
plans (pre-execution). So any synthesis of "what the drive produced" would have to re-read the
unstructured Pi conversation — memory as the source of truth, with no form.

Council already emits a **structured, validated plan** on the way in (`OrchestrationPlan` →
`domain.NewPlan`, with the contract reflection-rendered into council's prompt via
`planContractSpec()` and guarded by a drift test). The return side had no equivalent.

## Decision

**Every driven task returns a schema'd report envelope the worker self-reports; the harness
validates and stores it (with the raw output kept for audit); and council runs a post-execution
decision stage that consumes the normalized envelopes — not the conversation — and emits a
structured decision the harness validates and stores.**

- **One report envelope for every agent.** A new `domain.TaskReportEnvelope` (status, summary,
  dod_met, confidence, and optional `outputs`/`findings`/`verification`/`follow_ups`/
  `open_questions`) is the single normalized shape council consumes; archetype-specific richness
  goes in the optional slices. `reportContractSpec()` renders it by reflection into **every
  specialist's** `AgentProfile.SystemPrompt()` (reusing the plan contract's `writeShape`), guarded
  by a drift test — so a worker is always instructed to close its turn with the envelope as a
  fenced ```json block.
- **Worker self-reports; raw is audit.** The coordinator extracts the envelope from the worker's
  stdout (`extractReport`) and, when present, attaches it plus the **full raw output** to the
  `report_run` task frame. A missing or malformed envelope is **normalized to a minimal valid one**
  (`normalizeEnvelope`) so the drive never breaks on validation. The raw output is stored but is
  **not** surfaced to council — it is for audit/debug.
- **Storage — a new aggregate + table.** `domain.TaskReport` (validated envelope + raw, keyed to
  the run and the plan-local task ref) UPSERTs on `(plan_run_id, task_ref)` — a retried task
  overwrites its report. The `ReportRun` command stores it in the **same transaction** as the
  status move, so progress and report never diverge. New table `task_reports`.
- **Council decision stage.** After the drive completes, the extension fetches the run's normalized
  envelopes (`get_reports`), composes a decision-stage message, and runs **one council turn** under
  the council persona (which now carries a `POST-EXECUTION DECISION` section + a
  `decisionContractSpec()`). Council emits a `domain.CouncilDecision` (overall verdict, per-task
  verdict, dod_met, next actions); the extension captures it and submits it (`submit_decision`).
  The harness validates it into `domain.PlanDecision` and stores it append-only. New table
  `council_decisions`. Skipped on a dry run (no real output to judge).

New frames (additive): `report_run` gains optional `report`/`raw`/`agent` on its task; new
`get_reports` → `reports` and `submit_decision` → `decision_recorded`.

## Consequences

- **Positive**: every agent's result has one validated shape; council consumes the normalized
  reports as the source of truth instead of re-reading the conversation; raw output is preserved
  for audit. The plan/decision symmetry mirrors the existing contract-first pattern (reflection
  spec + drift test), so prompt and types cannot drift. Frames are additive; the report rides the
  existing `report_run` round-trip (no extra trip per task).
- **Negative**: a worker that ignores the contract yields a thin synthesized envelope (status +
  summary only), so the decision is only as rich as the workers' compliance. The decision turn
  costs one extra council turn (a real model call) per drive. Raw output is stored in the DB
  (`task_reports.raw`) — large for big outputs; a file/artifact store is a future option.
- **Neutral**: one new migration (`create_task_reports`, two tables). `council_decisions` is
  append-only (each iteration records its own; latest by `created_at` is current). The decision
  stage runs in-session as council, reusing the `before_agent_start` persona swap.

## Module boundary

Still **Module 1** ([ADR-0001](0001-harness-layout.md)): the harness governs and records (validates
and stores the reports and the decision); the Pi extension still coordinates execution and runs the
in-session council turns. Nothing here makes the harness schedule or run workers.

## Alternatives considered

- **A dedicated normalizer step** (free prose → envelope via a separate LLM call) — rejected for
  now: an extra model call per task; worker self-report matches the existing plan-contract pattern.
  The `normalizeEnvelope` fallback covers non-compliant workers.
- **Per-archetype envelope schemas** — rejected: council would have to branch per shape; a single
  envelope with optional slices keeps consumption uniform while preserving richness.
- **Extend `plan_task_runs` with report columns** — rejected: couples the durable artifact to the
  run-progress row; a dedicated `task_reports` table separates "report/artifact" from "progress".
- **Reuse the existing `report_run` summary only** (no schema) — rejected: that is the status quo
  the change exists to fix.

## References

- [ADR-0021](0021-drive-the-council-plan.md) (the drive this extends),
  [ADR-0019](0019-persist-council-orchestration-plan.md) /
  [ADR-0017](0017-council-orchestration-model.md) (the plan contract this mirrors),
  [ADR-0020](0020-specialist-agent-personas.md) (the worker personas now carrying the report
  contract), [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md) /
  [ADR-0014](0014-domain-encapsulation-single-package.md) (ports, encapsulation),
  [conventions/api.md](../conventions/api.md) (the new frames).
- `internal/domain/{report_contract,decision_contract,task_report,task_report_writer,task_report_reader,plan_decision,plan_decision_writer}.go`,
  `internal/app/command/{report_run,submit_decision}.go`, `internal/app/query/get_reports.go`,
  `internal/infra/sqlite/repo/{task_report,plan_decision}.go`,
  `internal/delivery/pilink/{pilink,handler}.go`, `migrations/…_create_task_reports.sql`,
  `apps/pi-harness-extension/src/coordination/{parse,engine,types}.ts` + `index.ts`.
