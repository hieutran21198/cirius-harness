# Agent orchestration & strategy-planning patterns — designing `council`

- **Type**: tool
- **Date**: 2026-06-21
- **Status**: Current

## Summary

The harness's `council` agent is an **orchestrator/lead**: it weighs a request and produces a
**strategy plan** that delegates execution to specialists. The strongest published guidance
(Anthropic's orchestrator-worker pattern and multi-agent research system, plus the broader
planner/supervisor literature) converges on a small set of rules: **plan before acting**,
**scale effort to complexity**, and **delegate with specificity**. Council is Claude-family, which
*rewards* the mechanics-heavy, checklisted prompt these rules imply
([model-families](model-families.md)).

## Findings

**Orchestrator-worker — when subtasks can't be predicted.** A central LLM breaks a task into
subtasks, delegates to workers, and synthesizes. *Use when* you "can't predict the subtasks
needed" — the canonical example is a multi-file code change, i.e. exactly council's job. Distinct
from fixed parallelization because the subtasks are decided from the input.

**Plan-and-execute — separate planning from execution.** "The architect draws the full blueprint
upfront, hands it to builders." Council *plans only* and never edits — the implementer executes.
This is the clean PLAN-mode / ACT-mode split; council is pure PLAN mode.

**Lead-agent flow** (Anthropic multi-agent research system): *analyze → think/plan → delegate with
specificity → synthesize → decide if more is needed.* Two lessons dominate:

- **Scale effort to complexity (the #1 lesson).** "Agents struggle to judge appropriate effort for
  different tasks, so we embedded scaling rules in the prompts" — e.g. 1 specialist for simple,
  2–4 for comparisons, more for complex. Without this, agents over- or under-invest.
- **Vague delegation is the #1 failure.** Early "research the X" instructions made subagents
  "perform the exact same searches." The fix: give each delegated step an **objective, expected
  output, tools/sources, and clear boundaries**; avoid overlap; prefer parallel where independent.

**Strategy-plan anatomy.** A good agent-produced plan carries: goal restatement · assumptions /
drivers · risks & unknowns · decomposition · sequencing (mark parallelizable) · delegation ·
success criteria / checkpoints. Ask for underlying assumptions, not just conclusions.

**Simplicity-first.** "Find the simplest solution possible, and only increas[e] complexity when
needed." Structure council's prompt only where it demonstrably improves the plan.

### Orchestration mechanics (the council-brain layer)

- **Capability-based / router-driven routing.** A router classifies the task and selects the
  best-fit expert by skill, cost, latency, and reliability. *Use when* the team is heterogeneous
  and tasks vary — council's exact job. Drives council's typed **capability roster** and
  **routing rules**.
- **Task Dependency Graph (DAG) → parallel "waves".** Decompose the request into a
  dependency-ordered graph; independent tasks run concurrently in waves. *Use when* a request has
  separable parts — split by module/package/repo boundary.
- **Four-gate human-in-the-loop**: advisory · validating · blocking · escalating. *Use when*
  actions vary in blast radius — low-risk runs free; auth/payment/security/config/production or
  irreversible work **blocks** on human approval; novel patterns **escalate**.
- **Cost-aware / step-wise model selection.** Strong model for the steps that matter, cheap model
  for mechanical steps; confidence-based escalation. *Use when* a plan mixes deep and trivial work
  — prefer the cheapest capable agent per task.
- **Definition of Done = a blocking verifier** against the spec (build/test/lint/rule/human
  approved). *Use* as each task's exit criteria.

- Good (for council): orchestrator-worker + plan-and-execute, with effort-scaling and specific
  delegation baked into the persona.
- Bad: a thin "make a plan" prompt — it under-specifies effort and delegation, the two documented
  failure modes.
- Fit: the **communicator** archetype / Claude family — mechanics-heavy, checklisted prompts.

## Evidence

External best-practice + first-party guidance; **confidence: medium** (industry guidance, not a
harness-specific benchmark). Sources:

- Anthropic — [Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)
  (routing · orchestrator-workers · evaluator-optimizer · simplicity-first).
- Anthropic — [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
  (lead-agent flow, effort-scaling rules, delegation specificity).
- Plan-and-execute / supervisor patterns (LangGraph supervisor & plan-and-execute, CrewAI,
  AutoGen group-chat) — the planner-decides / executor-acts split and hierarchical delegation.
- [MasRouter](https://arxiv.org/pdf/2502.11133) and router-driven task-classification + expert-
  selection frameworks — capability-based routing.
- Four-gate human-in-the-loop (digitalapplied governance framework; Cloudflare Agents HITL docs);
  cost-aware routing ([arXiv 2508.12491](https://arxiv.org/pdf/2508.12491)); task DAG / waves
  (coordinator → dependency-ordered DAG → parallel waves).

## Recommendation

Encode council as a **structured persona** (a `domain.Persona`, [ADR-0016](../adr/0016-harness-owned-agent-persona-governed-turn.md))
that renders a mechanics-heavy prompt with: a plan-only/read-only mission, an **effort-scaling**
rule, a **fixed output contract** (Goal · Assumptions · Risks & unknowns · Strategy · Delegation ·
Success checks), and a **delegation roster** tied to the real team where each summoned step states
its objective/output/boundaries. Keep it the single source of council's behaviour; revisit as the
team or models change. Confidence **medium**.

## References

- [ADR-0016](../adr/0016-harness-owned-agent-persona-governed-turn.md) — the persona that applies
  this research. [model-families](model-families.md) — why council is Claude-family.
  [PDR-0001](../pdr/0001-default-team-model-assignments.md) — the team and their model assignments.
