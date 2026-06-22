# 0021. Drive the council plan (read-back + run state, client-coordinated)

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0017](0017-council-orchestration-model.md) and
  [ADR-0019](0019-persist-council-orchestration-plan.md) (which **deferred** driving the plan —
  "the loop" — to a future executor; this lands the read-back + progress recording that a drive
  needs). **Relates to** [ADR-0001](0001-harness-layout.md) (the Module 2 boundary).

## Context

Council produces and the harness persists an `OrchestrationPlan` — a task DAG with waves,
dependencies, assignees, and gates ([ADR-0017](0017-council-orchestration-model.md),
[ADR-0019](0019-persist-council-orchestration-plan.md)). Those ADRs explicitly left *driving* it to
a future executor. Driving means: run each task as its assigned agent, sequenced by the plan's
waves and `depends_on` (independent tasks concurrently, dependents in order), gate high-risk tasks
on human approval, and record progress.

Two platform facts (verified against the installed Pi SDK) shape the decision:

- **Pi has no in-session sub-agents or concurrent turns** — one agent loop per process; an
  extension can only swap the system prompt of the next single turn. Real parallelism is only
  possible by spawning **multiple headless `pi` processes**. Pi supports this:
  `pi --no-extensions --mode json --system-prompt "<persona>" [--model …] "<task prompt>"` runs a
  governed worker to completion and streams its result as JSON, and the extension can spawn it via
  `pi.exec`. `--system-prompt` injects the agent's persona directly; `--no-extensions` stops the
  child from spawning its own harness.
- **The harness had no read-back and the Plan is write-once** — there was no query to fetch a
  persisted plan, and the `Plan` aggregate has no status mutation. A drive needs both.

## Decision

**A council plan is driven by a coordinator in the Pi extension that spawns one headless `pi`
worker per task; the harness serves the plan and records progress. The harness governs and
records — it does not schedule or run workers.**

- **Read-back.** A `domain.PlanReader` (`FindByID`, `LatestForSession`), a readstore
  implementation that rehydrates the plan from its six tables, `Plans()` on `query.ReadStore`, and
  a `GetPlan` query returning the plan in the **`OrchestrationPlan` contract shape** (the same
  vocabulary `submit_plan` uses) plus its status and a ref→task-id map. Carried on a new
  `get_plan` → `plan` frame (`planId` optional — empty fetches the session's latest plan).
- **Run state — a separate, mutable aggregate.** The `Plan` stays an immutable spec; a new
  `domain.PlanRun` (with child `TaskRun`s and a `TaskStatus` enum) records execution. Status moves
  are guarded by legal transitions in the domain (`{planned,approved}→driving→done|cancelled`;
  tasks `pending→running→done|failed|skipped`, `failed→running` retry; idempotent self-moves). A
  `PlanRunWriter` **UPSERTs** (the one writer that updates, not inserts-once). New tables
  `plan_runs` and `plan_task_runs`. A decorated `ReportRun` command loads-or-seeds the run from the
  plan's refs and applies the move in one transaction; carried on a new `report_run` →
  `run_reported` frame.
- **Coordinator (Pi extension).** A `/drive [planId]` command fetches the plan (`get_plan`), then
  walks its waves in order — tasks within a wave run concurrently (`Promise.all`), each awaiting
  its `depends_on` — spawning a headless `pi` worker per task with the assignee's resolved persona,
  threading a dependency's output into its dependents' prompts. Blocking-gated tasks
  (gate `blocking`/`escalating`, high risk, or a sensitive keyword) pause for human approval before
  spawning; edit-capable tasks are serialized within a wave (a shared cwd). Progress is reported via
  `report_run` (best-effort — a report failure never aborts the drive). A `--drive-dry-run` flag
  echoes prompts instead of spawning, to exercise the loop without models.

## Consequences

- **Positive**: the deferred "drive the loop" now works end to end — a persisted plan executes with
  per-agent personas, wave/dependency sequencing, real parallelism (multi-process), human gates,
  and durable run state — with **no change** to the `submit_plan` path or the `Plan` aggregate
  (spec/run separation means an approved plan is never rewritten). Frames are additive. The
  coordinator is unit-testable (injected `exec`/`request`/`confirm`).
- **Negative**: concurrent edit-capable workers in one cwd can still race — mitigated by serializing
  edit tasks within a wave; **git-worktree-per-task** is the robust follow-up. The persona's `model`
  is passed through when present, else the worker uses its default (model governance is still
  ADR-0016's milestone). Capturing a worker's full stdout buffers it in memory; threaded context is
  truncated. The archetype/run-status mapping now lives in two places (domain + the wire) kept in
  sync by tests.
- **Neutral**: one new migration (`create_plan_runs`); one live run per plan in this slice
  (`UNIQUE(plan_id)`). The coordinator relies on the `pi` binary being on `PATH`.

## Module boundary

A **client-coordinated** drive — the harness only serves the plan (`get_plan`) and records what the
client reports (`report_run`), while Pi spawns and runs the workers — is **still Module 1**
(governs + records, as AGENTS.md frames the harness). **Module 2** ([ADR-0001](0001-harness-layout.md))
begins only when the **harness itself** schedules the DAG, provisions environments, launches the
workers, and waits on their outputs — i.e. owns execution and the cross-session/cross-client view.
This ADR deliberately stops short of that; it is the persistence + read-back substrate a future
Module-2 scheduler sits on.

## Alternatives considered

- **Mutate the `Plan` aggregate to carry status** — rejected: breaks its write-once design, lets a
  drive rewrite an approved spec, and has no precedent for status setters. The separate `PlanRun`
  keeps the spec immutable.
- **A status column on `plan_tasks`** — rejected for the same reason; run state is execution, not
  spec.
- **Sequential single-loop driver** (no parallelism) — considered and is the safe fallback, but the
  user chose real parallelism; multi-process headless `pi` delivers it.
- **A harness-owned Go executor that spawns the workers** — that is Module 2; deferred. It inverts
  today's topology (Pi spawns the harness) and is a larger change.

## References

- [ADR-0017](0017-council-orchestration-model.md) (the plan this drives),
  [ADR-0019](0019-persist-council-orchestration-plan.md) (persisting the plan — refined here),
  [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) /
  [ADR-0020](0020-specialist-agent-personas.md) (the personas each worker runs as),
  [ADR-0001](0001-harness-layout.md) (Module 2), [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md)
  / [ADR-0014](0014-domain-encapsulation-single-package.md) (ports, encapsulation),
  [conventions/api.md](../conventions/api.md) (the new frames).
- `internal/domain/{plan_reader,plan_run,plan_run_writer,plan_run_reader}.go`,
  `internal/app/query/get_plan.go`, `internal/app/command/report_run.go`,
  `internal/infra/sqlite/repo/{plan_reader,plan_run}.go`,
  `internal/delivery/pilink/{pilink,handler}.go`, `migrations/…_create_plan_runs.sql`,
  `apps/pi-harness-extension/src/coordination/*` + `index.ts`.
