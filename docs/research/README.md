# Research

The **evidence corpus** for the harness: what we've learned about AI **models**
(what each is good/bad at), **tools**, and **clients**. This is the raw material that
**feeds decisions** — it is not itself a decision.

```
research → pdr (decision) → .cirius-harness schema → ai-agent domain + migration → runtime
```

See [ADR-0001](../adr/0001-harness-layout.md) for the full pipeline.

## What belongs here

- Model capability profiles — strengths, weaknesses, prompt-style fit (e.g. the
  Claude-family vs GPT-family reasoning already noted in
  [`.cirius-harness/00-system.yaml`](../../.cirius-harness/00-system.yaml)).
- Tool evaluations — what a tool does well, its limits, when to reach for it.
- Client evaluations — capabilities and control surface of clients (opencode, …).

## Index

- [model-families.md](model-families.md) — Claude-family vs GPT-family prompt-style fit.
- [agent-orchestration.md](agent-orchestration.md) — orchestrator-worker / plan-and-execute,
  capability routing, task DAGs, four-gate HITL; the evidence behind council's orchestration model.
- [agent-team-composition.md](agent-team-composition.md) — how many agents (lean wins); the evidence
  behind keeping 8 agents + lenses (PDR-0002).

## What does NOT belong here

- **Decisions.** "We will use model X for agent Y" is a [PDR](../pdr/README.md), and it
  cites the research that justifies it.
- **Architecture decisions.** Those are [ADRs](../adr/README.md).

## Conventions

- A research doc states **findings + evidence**, and ends with a **recommendation** the
  PDR can act on. Evidence over opinion: link benchmarks, transcripts, or runs.
- Research is **living** — update it as models/tools/clients change. Note the date.
- Naming: `docs/research/kebab-title.md` (e.g. `claude-vs-gpt-prompt-fit.md`). Drop the
  sequence number; research is evergreen reference, not ordered.
- Use [template.md](template.md).
