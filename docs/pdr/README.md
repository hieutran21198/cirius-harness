# Provider Decision Records

A **PDR** records a **decision about what the harness uses** — which provider, model,
tool, or client, for which job — justified by [research](../research/README.md). PDRs
are to provider/model/tool choices what [ADRs](../adr/README.md) are to architecture.

```
research (evidence) → PDR (decision) → .cirius-harness schema → ai-agent domain + migration → runtime
```

See [ADR-0001](../adr/0001-harness-layout.md) for the full pipeline.

## PDR vs ADR vs research

| Doc          | Captures                                  | Example                                            |
| ------------ | ----------------------------------------- | -------------------------------------------------- |
| **research** | evidence / findings                       | "GPT-5.5 is strong at autonomous technical work"   |
| **PDR**      | a what-to-use decision (with tradeoffs)   | "`implementer` uses GPT-5.5, falls back to Opus"   |
| **ADR**      | an architecture decision                  | "harness layout + research→code pipeline"          |

A PDR's decision lands in the schema ([`.cirius-harness/00-system.yaml`](../../.cirius-harness/00-system.yaml))
and, downstream, in the seed migration for the `ai-agent` domain.

## Lifecycle (append-only, mirrors ADR)

PDRs are **append-only**. To change a decision, write a new PDR with a higher number and
mark the old one superseded. `Status`: `Proposed` → `Accepted` → `Superseded by PDR-NNNN`
/ `Deprecated`.

## Naming

```
docs/pdr/NNNN-kebab-title.md
```

`NNNN` is a four-digit zero-padded sequence. Title reads as a result
("use-gpt-for-implementer"), not a question.

## Writing one

Use [template.md](template.md). Every PDR **must cite the research** that justifies it —
a PDR without evidence is just an opinion.
