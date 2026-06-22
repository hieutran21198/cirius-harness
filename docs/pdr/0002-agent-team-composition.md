# 0002. Keep a lean 8-agent team; route categories with lenses

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

Council's orchestration brain ([ADR-0017](../adr/0017-council-orchestration-model.md)) reasons over
a rich **category taxonomy** (explore, research, architect, plan, implement, test, review, security,
performance, docs, migration, devops, integrate). That taxonomy is broader than the **team** that
exists. The question: do we mint a specialist agent per category, or route the taxonomy onto a small
team? The constraint is real — over-splitting costs tokens, latency, and accuracy; under-splitting
overloads an agent's context and mismatches capability to task.

## Decision

**Keep the 8 default agents** (`prayer, council, planner, implementer, researcher, explorer,
reviewer, scribe`). Council routes each category to the best-fit agent, optionally summoning it in a
**lens** (focus-mode): reviewer→security/performance/docs-review/plan-gap; planner→architect/
domain-design/integration; implementer→tester/db-specialist/devops/migration. **Add a new agent
only when all four hold**: a distinct skill domain *and* a distinct tool/permission profile *and*
benefit from an isolated context window *and* the work is frequent. Deferred candidates: `tester`
(if TDD becomes a core workflow), `devops` (if deploy/infra work becomes frequent).

## Evidence

[agent-team-composition.md](../research/agent-team-composition.md) — coordination overhead and
saturation, the "bag of agents" 17× error trap, 39–70% sequential handoff degradation (Google),
4–220× token cost (UIUC), and "as little context as required." Lenses are encoded in council's
typed capability roster ([agent-orchestration.md](../research/agent-orchestration.md)).

## Consequences

- Positive: lean team avoids handoff loss, token blow-up, and fragmentation; council's capability
  roster + routing rules give full taxonomy coverage without new agents; clear, evidence-backed
  criteria gate future growth.
- Negative / risk: lenses are advisory prompt guidance, not enforced roles — a lens does not change
  the agent's real permissions; if a deferred category (testing, devops) becomes frequent, revisit.
- Schema impact: **none now** — no change to `.cirius-harness/00-system.yaml` or the seed. A future
  `tester`/`devops` addition would add agent entries + model assignments (a new PDR) + Casbin policy.

## Alternatives considered

- **One agent per category** (security, perf, db, devops, tester, architect, integration-planner) —
  rejected: over-split; the evidence shows token/latency/accuracy costs outweigh specialization for
  a team this size.
- **Collapse to fewer than 8** — rejected: the existing roles already map to distinct archetypes,
  permissions, and model families (PDR-0001); merging would overload context and mismatch capability.

## References

- [agent-team-composition.md](../research/agent-team-composition.md) — cited evidence.
- [ADR-0017](../adr/0017-council-orchestration-model.md) — council's orchestration model that routes
  the taxonomy. [PDR-0001](0001-default-team-model-assignments.md) — the per-agent models.
