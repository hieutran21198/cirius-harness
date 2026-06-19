# 0001. Default team model assignments

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

The harness ships a **default team** of eight agents (the team you get when your workspace
defines nothing). Each agent needs a concrete primary model and a fallback, chosen for the
job it does and the prompt style it requires. This PDR records those assignments as one
decision; they are encoded in `.cirius-harness/00-system.yaml`.

## Decision

Assign each agent by **archetype → model family → concrete model**, per
[model-families research](../research/model-families.md):

| Agent         | Archetype        | Verb     | Primary model               | Fallback                     |
| ------------- | ---------------- | -------- | --------------------------- | ---------------------------- |
| `council`     | communicator     | route    | `anthropic/claude-opus-4-7` | `openai/gpt-5.4`             |
| `planner`     | communicator     | design   | `anthropic/claude-opus-4-7` | `moonshot/kimi-k2.7`         |
| `implementer` | principle-driven | build    | `openai/gpt-5.5`            | `anthropic/claude-opus-4-8`  |
| `researcher`  | principle-driven | gather   | `openai/gpt-5.4`            | `google/gemini-3-pro`        |
| `explorer`    | utility-runner   | scan     | `minimax/minimax-m3`        | `deepseek/deepseek-v3`       |
| `reviewer`    | communicator     | critique | `anthropic/claude-sonnet-4-6` | `openai/gpt-5.4`           |
| `scribe`      | communicator     | retain   | `anthropic/claude-sonnet-4-6` | `moonshot/kimi-k2.7`       |
| `prayer`      | none             | pray     | — (model-less)              | —                            |

Rationale in brief: communicator/instruction-following roles (routing, planning, critique,
distillation) go to the Claude family — Opus for heavy reasoning, Sonnet for the lighter
critique/retain jobs. Autonomous technical roles (build, gather) go to the GPT family.
Fast/cheap scanning goes to MiniMax (utility-runner). `prayer` is intentionally model-less.

## Evidence

[docs/research/model-families.md](../research/model-families.md) — the Claude-family vs
GPT-family prompt-style and capability profiles, and the archetype mapping this table
follows. Confidence is **medium** (experience-based, not yet benchmark-backed); the
fallbacks hedge that.

## Consequences

- Positive: every default agent has a job-appropriate model and a fallback, so the team
  works out of the box and degrades gracefully on error/ratelimit/budget/quota.
- Negative / risk: assignments are experience-based, not benchmarked; model version pins
  (e.g. `gpt-5.5`, `opus-4-7`) will drift and need revisiting as families update.
- Schema impact: lands in `.cirius-harness/00-system.yaml` and, downstream, the (future)
  seed migration that writes system agents into the `agents` table.

## Alternatives considered

- **One family for everything** — simpler ops. Rejected: the communicator/principle-driven
  split is the whole point; a single family is wrong for half the roles.
- **MiniMax for more than scanning** — cheaper. Rejected: discouraged for reasoning per the
  research; confined to `explorer`.

## References

- [docs/research/model-families.md](../research/model-families.md) — cited evidence.
- `.cirius-harness/00-system.yaml` — where this decision is encoded.
- [ADR-0001](../adr/0001-harness-layout.md) — the research→PDR→schema pipeline.
