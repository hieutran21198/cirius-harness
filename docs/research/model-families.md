# Model families — Claude-family vs GPT-family, and archetype fit

- **Type**: model
- **Date**: 2026-06-19
- **Status**: Current

## Summary

AI models don't just differ in "smart vs dumb" — they respond to **different prompt
styles**. Two families dominate our usage: the **Claude family** rewards mechanics-driven,
instruction-heavy prompts; the **GPT family** rewards principle-driven, autonomous ones.
This split maps directly onto the [archetypes](../../.cirius-harness/README.md)
(`communicator`, `principle-driven`, `utility-runner`) the harness assigns agents.

## Findings

**Claude family — communicative, instruction-following.**
Responds to *mechanics-driven* prompts: detailed checklists, templates, step-by-step
procedures, nested workflows. More rules ⇒ more compliance; it will follow a 1,100-line
prompt step by step. Strong at multi-step instruction-following, maintaining flow across
many tool calls, nuanced delegation/orchestration, and well-structured communicative
output. Weaker on deep, purely-technical problem solving than the GPT family.

- Ranking within family: **Claude (Fable > Opus > Sonnet) > Kimi (k2.7 > k2.6) > GLM (5.2 > 5.1) > Big Pickle (GLM 4.6)**.
- **Fit**: the **communicator** archetype — `council`, `planner`, `reviewer`, `scribe`.

**GPT family — principle-driven, autonomous.**
Responds to *principle-driven* prompts: concise principles, XML structure, explicit
decision criteria. More rules ⇒ more contradiction surface ⇒ more drift. Works best when
you state the goal and let it find the mechanics. Strong at deep, autonomous technical work
and information-seeking.

- Ranking within family: **GPT (5.5 > 5.4) > Deepseek > MiniMax**.
- **Fit**: the **principle-driven** archetype — `implementer`, `researcher`.
- **MiniMax** is strongly discouraged for reasoning: cheap and fast, use for **explorer /
  quick tasks only** (m3 > m2.7 > m2.5) — i.e. the **utility-runner** archetype.

## Evidence

This is **experience-based prompt-engineering observation**, not a formal benchmark suite:
accumulated from running these families against the harness's own (large, mechanics-heavy)
prompts and tasks, and recorded in the `.cirius-harness/00-system.yaml` header comments.
**Confidence: medium.** Gap: no reproducible benchmark/transcript corpus is linked yet;
adding one (task suites + scored runs per family) would raise confidence and let us
re-rank objectively as models change.

## Recommendation

Assign by **archetype → family → concrete model**, with a same- or cross-family fallback:

- **communicator** → Claude family (Opus for heavy reasoning, Sonnet for lighter
  distillation/critique); Kimi as in-family fallback.
- **principle-driven** → GPT family (GPT-5.5 for build, 5.4 for research); Claude Opus or
  Gemini (long-context) as fallback.
- **utility-runner** → MiniMax for fast/cheap scanning only; Deepseek as fallback.

Acted on in [PDR-0001](../pdr/0001-default-team-model-assignments.md).

## References

- `.cirius-harness/00-system.yaml` — the header comments this doc formalizes.
- `.cirius-harness/README.md` — the archetype definitions.
- [PDR-0001](../pdr/0001-default-team-model-assignments.md) — the decision citing this research.
