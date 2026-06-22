# 0017. Council orchestration model & machine-readable plan contract

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (council's persona is a
  domain constant; this gives that persona an internal structure and an output contract)
- **Refined by**: [ADR-0019](0019-persist-council-orchestration-plan.md) (the deferred plan
  persistence below now lands; the output is reframed to Markdown-for-review then JSON-on-approval)

## Context

Council must be more than a "make a plan" prompt. It is the team's **orchestrator**: classify a
request's intent, weigh it across dimensions, decompose it into categorized tasks, route each to the
best-fit agent by capability, sequence them into dependency-ordered waves, and govern the flow with
quality gates and human approval. Three forces shape how we encode this:

- **The harness governs, never executes** (AGENTS.md). There is **no runtime multi-agent executor**
  yet, and council runs as a **single governed Pi turn** ([ADR-0016](0016-harness-owned-agent-persona-governed-turn.md)).
  So council **plans** the orchestration; a human reviews the plan; a future engine (Module 2,
  [ADR-0001](0001-harness-layout.md)) drives it.
- **The persona is harness-owned code** ([ADR-0016](0016-harness-owned-agent-persona-governed-turn.md)).
  Council's behaviour should be **structured, typed, and testable**, not a prose blob.
- **The team is lean** ([PDR-0002](../pdr/0002-agent-team-composition.md)). Council's roster must
  route a rich category taxonomy onto a few agents (with lenses), not assume one agent per category.

## Decision

**Council's behaviour is a typed orchestration model (`domain.CouncilProfile`) rendered to its
system prompt, and council emits a machine-readable `OrchestrationPlan` that a human reviews before
it is driven.**

- The framework is fully structured in Go (`internal/domain/orchestration.go`): `Intent`,
  `TaskDimension` (the 7 lenses), `Category` (the taxonomy), `AgentCapability` (the team roster with
  cost/speed/reliability/risk/tools/lenses), `RoutingRule`, `PipelineStage` (classify → discover →
  decompose → assign → sequence → execute → collect → cross-check → approve → integrate → validate →
  report), `QualityGate` (the four-gate HITL model + a Definition-of-Done checklist), and the
  assignment factors `TaskType + Risk + Scope + RequiredSkill + Dependencies + OutputType`.
  `CouncilProfile.SystemPrompt()` renders these into the mechanics-heavy prompt (on-archetype for
  Claude). The capability roster is authored from `.cirius-harness/00-system.yaml`,
  [model-families](../research/model-families.md), and [PDR-0001](../pdr/0001-default-team-model-assignments.md);
  a test keeps every referenced agent a real team role.
- Council's **output** is a typed `OrchestrationPlan` / `PlannedTask` contract
  (`internal/domain/plan_contract.go`); the prompt's required output format is rendered **from those
  types by reflection**, so the contract has a single source. Council emits the plan as JSON first
  (machine-readable), then a short human summary; **high-risk tasks block on human approval**.
- `Persona` becomes an interface (`Agent()`, `SystemPrompt()`); `CouncilProfile` implements it.
  The `resolve_agent`/`agent_resolved` wire ([ADR-0008](0008-pi-client-integration-stdio.md)) and the
  Pi extension are **unchanged** — persona is still a string on the wire, only richer.

## Consequences

- **Positive**: council's orchestration logic is typed, unit-tested (sync guard on the roster;
  schema↔prompt drift guard), and versioned with the harness. The machine-readable plan is
  forward-compatible with a future executor; the wire/extension did not change.
- **Negative**: the rendered prompt is large (intended — Claude rewards it). The plan is **advisory**
  until the runtime executor exists; v1 council cannot itself activate agents or wait on outputs.
  Lenses are prompt guidance, not enforced permissions (PDR-0002).
- **Neutral**: no schema/DB change; no new wire frame.

## Alternatives considered

- **Prose-only persona** — rejected: the user requires a structured profile; prose is untestable and
  drifts from the real team.
- **Emit the plan as Go-parsed/executed now** — deferred: needs a runtime multi-agent engine
  (Module 2); v1 is plan-only with human review.
- **One agent per category** — rejected ([PDR-0002](../pdr/0002-agent-team-composition.md)): over-
  splitting costs tokens, latency, and accuracy.

## References

- [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (persona as a domain constant;
  refined here), [PDR-0002](../pdr/0002-agent-team-composition.md) (lean team + lenses),
  [agent-orchestration.md](../research/agent-orchestration.md) and
  [agent-team-composition.md](../research/agent-team-composition.md) (the evidence).
- `internal/domain/{orchestration,plan_contract,persona}.go` — the implementation.
