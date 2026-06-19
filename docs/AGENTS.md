# docs/

Harness documentation - decisions, specs, conventions, vocabulary. Anything that explains **why** or **how** lives here.

## Layout

```
docs/
├── research/           # Evidence corpus: AI models (good/bad), tools, clients.
├── pdr/                # Provider Decision Records (what to use). Append-only.
├── adr/                # Architecture Decision Records. Append-only.
├── specs/              # Feature / system specs.
├── conventions/        # Workspace-wide coding + process conventions.
└── glossary/           # Shared vocabulary.
```

## When to write what

| The change is...                                        | Document it as...                      |
| ------------------------------------------------------- | -------------------------------------- |
| Evidence about a model / tool / client (good/bad)       | A finding in [research/](research/)    |
| A what-to-use decision (provider / model / tool)        | A PDR in [pdr/](pdr/)                   |
| An architectural decision with tradeoffs                | New ADR in [adr/](adr/)                |
| A feature design touching 2+ modules                    | A spec in [specs/](specs/)             |
| A naming / process rule the whole workspace follows     | A page in [conventions/](conventions/) |
| A term the whole company uses that needs disambiguation | An entry in [glossary/](glossary/)     |

## Conventions

- **ADRs are append-only.** To reverse a decision, write a new ADR that supersedes the old; never edit the original.
- **Specs and ADRs reference modules by path, not by description.**
- **Glossary is canonical.** When two specs disagree on what "XXX" means, the glossary wins. Update the glossary first.
- **No PII, no secrets, no live credentials.** Docs are public-by-default for everyone with repo access.

## Anti-patterns

- **Documenting service internals here.**
- **Editing past ADRs.** They are historical. Annotate via supersedes.
- **`final-v2-FINAL.md` filenames.** ADRs use the `NNNN-kebab-title.md` form. Specs use `NNNN-kebab-title.md` too if there's an ordering, otherwise just `kebab-title.md`.
- **Drift between glossary and code.** A glossary term should map to an actual type. If they diverge, fix the docs OR the code in the same PR.
