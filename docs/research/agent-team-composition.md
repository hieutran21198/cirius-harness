# Agent team composition — how many agents, and when to add one

- **Type**: tool
- **Date**: 2026-06-22
- **Status**: Current

## Summary

More agents is **not** better. Across controlled studies and practitioner reports, expanding a
multi-agent team past what the workload needs adds coordination overhead, token cost, latency, and
error cascades. The harness should keep a **lean team** and let council route a rich category
taxonomy onto it — adding a new agent only when a role is genuinely distinct *and* frequent.

## Findings

- **Coordination overhead & diminishing returns.** Scaling agent counts shows saturation effects;
  "Towards a Science of Scaling Agent Systems" derives quantitative scaling limits across
  topologies. Beyond the workload's needs, extra agents stop helping.
- **The "bag of agents" trap.** Naively adding agents can compound errors (a ~17× error inflation
  is reported) unless the coordination pattern matches the workload.
- **Handoff loss.** Google-cited results show **39–70% degradation** on sequential multi-agent
  tasks from inter-agent handoffs — the loss can exceed the specialization gain.
- **Token cost.** A UIUC study found multi-agent systems consume **4–220×** more tokens than a
  single-agent equivalent.
- **Context engineering.** "As little context as possible, but as much as required" — too many
  narrow agents fragment context and *increase* hallucination, not decrease it.
- **When specialists do pay**: a distinct expertise that benefits from an **isolated context
  window** and a genuinely different tool/permission profile. Subagents shine for research/scan
  side-quests that return a compact result to the lead, keeping the main context clean.

- Good: a small team of broad roles + council routing categories (with lenses) onto them.
- Bad: one agent per category (security, perf, db, devops, tester, architect…) — over-split.
- Fit: keep the 8 default roles; map the taxonomy onto them.

## Evidence

Controlled studies + practitioner reports; **confidence: medium-high** (consistent across
independent sources). Sources: Towards a Science of Scaling Agent Systems (Yubin Kim et al.) ·
["bag of agents" 17× trap](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/)
· [Multi-Agent Teams Hold Experts Back](https://arxiv.org/html/2602.01011v1) · Google handoff
degradation & UIUC token-cost figures (via Augment Code multi-agent failure-modes guide) ·
Anthropic [multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
(subagents = isolated context windows).

## Recommendation

**Keep the 8 default agents** (`prayer, council, planner, implementer, researcher, explorer,
reviewer, scribe`). Council knows the richer category taxonomy and routes each category to the
best-fit agent, optionally in a **lens** (reviewer→security/performance; planner→architect/
integration; implementer→tester/db/devops/migration). **Add a new agent only when all hold**: a
distinct skill domain *and* a distinct tool/permission profile *and* benefit from an isolated
context, *and* the work is frequent. Top deferred candidates: `tester` (if TDD becomes core),
`devops` (if deploy/infra work becomes frequent). Confidence **medium-high**. Acted on in
[PDR-0002](../pdr/0002-agent-team-composition.md).

## References

- [PDR-0002](../pdr/0002-agent-team-composition.md) — the team-composition decision.
- [agent-orchestration.md](agent-orchestration.md) — how council routes the taxonomy.
- [model-families.md](model-families.md), [PDR-0001](../pdr/0001-default-team-model-assignments.md)
  — the per-agent model assignments.
