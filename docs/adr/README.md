# Architecture Decision Records

ADRs capture **decisions** - not designs, not specs, not status updates. An ADR explains why we chose X over Y, with enough context that a future contributor (or future-you) can revisit it.

## Template

[template.md](template.md) is a hybrid of two canonical ADR formats:

- **Structure** (`Context` / `Decision` / `Consequences` / `Alternatives considered` / `References`) follows Michael Nygard's [original ADR proposal](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) (Cognitect, 2011).
- **Metadata header** (`Status`, `Date`, `Deciders`, `Supersedes`, `Superseded by`) follows the [MADR](https://github.com/adr/madr) (Markdown Any Decision Records) convention.

For a curated catalog of every ADR template variant and the reasoning behind each, see [joelparkerhenderson/architecture-decision-record](https://github.com/joelparkerhenderson/architecture-decision-record).

## Naming

```
docs/adr/NNNN-kebab-case-title.md
```

- `NNNN`: four-digit sequence, padded with zeros (`0001`, `0002`, ...).
- Title is short, decisive, and reads as a result, not a question. "use-go-workspaces" beats "should-we-use-go-workspaces".

## Lifecycle

ADRs are **append-only**. To change a decision:

1. Write a new ADR with a higher number.
2. Set `Status: Accepted` on the new one.
3. Set `Status: Superseded by ADR-NNNN` on the old one. Leave the body intact.

`Status` transitions:

- `Proposed` - under discussion in a PR
- `Accepted` - merged, in effect
- `Superseded by ADR-NNNN` - replaced
- `Deprecated` - no longer in effect, no replacement

## Writing a new ADR

```bash
NEXT=$(printf "%04d" $(( $(ls docs/adr | grep -E '^[0-9]{4}' | wc -l) + 1 )))
TITLE="my-decision-title"
cp docs/adr/template.md "docs/adr/${NEXT}-${TITLE}.md"
$EDITOR "docs/adr/${NEXT}-${TITLE}.md"
```

## Index

- [0001-harness-layout](0001-harness-layout.md) - top-level harness repo layout
- [0002-persistence-and-migrations](0002-persistence-and-migrations.md) - SQLite + GORM (pure-Go) + embedded goose
- [0003-authorization-casbin-abac](0003-authorization-casbin-abac.md) - Casbin ABAC, policy in the DB
- [0004-ports-and-adapters-topology](0004-ports-and-adapters-topology.md) - inbound/outbound port & adapter layout
- [0005-surrogate-uuid-v7-keys](0005-surrogate-uuid-v7-keys.md) - UUID v7 surrogate PK on every aggregate
- [0006-model-catalog-and-agent-profiles](0006-model-catalog-and-agent-profiles.md) - first-class models + immutable agent profiles (session-pinned) — superseded by 0007
- [0007-roles-and-per-session-model-binding](0007-roles-and-per-session-model-binding.md) - agents as roles, per-session model binding, tool catalog, polymorphic session environment
- [0008-pi-client-integration-stdio](0008-pi-client-integration-stdio.md) - Pi extension launches `harness serve` as a per-session stdio (NDJSON) child
- [0009-deployment-topology-per-client-harness](0009-deployment-topology-per-client-harness.md) - one child-harness per client (citizen); central motherboard deferred to Module 2
