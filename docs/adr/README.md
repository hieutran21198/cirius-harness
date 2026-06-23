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
- [0010-ts-build-pipeline-apps-to-pi-extensions](0010-ts-build-pipeline-apps-to-pi-extensions.md) - Pi extension source in `apps/`, esbuild-built into `.pi/extensions/` via a devenv task
- [0011-client-reported-model-catalog](0011-client-reported-model-catalog.md) - models are client-reported (synced at session start) into a global cumulative catalog; seed removed
- [0012-cqrs-application-layer](0012-cqrs-application-layer.md) - CQRS application layer (command/query handlers + decorators); replaces the named `port/inbound` interfaces of ADR-0004
- [0013-idiomatic-go-layout-and-unit-of-work](0013-idiomatic-go-layout-and-unit-of-work.md) - flatten to `domain/app/delivery/infra` (app-owned driven ports, consumer-defined interfaces); UnitOfWork for commands; supersedes ADR-0004
- [0014-domain-encapsulation-single-package](0014-domain-encapsulation-single-package.md) - one `domain` package; aggregates encapsulate state behind New/Rehydrate + grouped views; refines ADR-0013
- [0015-client-aware-model-catalog](0015-client-aware-model-catalog.md) - the model catalog key gains `client` (`(client, provider, slug)`); model names are client-specific; refines ADR-0011
- [0016-harness-owned-agent-persona-governed-turn](0016-harness-owned-agent-persona-governed-turn.md) - an agent's `persona` is a harness-owned domain constant (code, not data); `/council` resolves it (`resolve_agent` frame) and Pi runs a one-shot governed turn; refines ADR-0007
- [0017-council-orchestration-model](0017-council-orchestration-model.md) - council's behaviour is a typed orchestration model rendered to its prompt; it emits a machine-readable `OrchestrationPlan` (human-reviewed, executed later); refines ADR-0016
- [0018-harness-observability-logging-audit-session](0018-harness-observability-logging-audit-session.md) - configurable structured logging to per-session files (level from config), a persisted audit log (events) via a command decorator, and session recording
- [0019-persist-council-orchestration-plan](0019-persist-council-orchestration-plan.md) - capture an approved council plan (Markdown review → JSON on approval) and persist it relationally (`plans` + 5 child tables) via a `submit_plan` frame; refines ADR-0017
- [0020-specialist-agent-personas](0020-specialist-agent-personas.md) - every working specialist (planner, implementer, researcher, explorer, reviewer, scribe) gets a harness-owned persona via a shared archetype-aware `AgentProfile`; no wire/query change; client command deferred; refines ADR-0016
- [0021-drive-the-council-plan](0021-drive-the-council-plan.md) - drive a persisted plan: a Pi-extension coordinator spawns one headless `pi` worker per task (by wave + depends_on, gated); harness adds plan read-back (`get_plan`) + a separate mutable `PlanRun` (`report_run`); still Module 1; refines ADR-0017/0019
- [0022-harness-logs-to-per-session-file](0022-harness-logs-to-per-session-file.md) - harness logs go to the per-session file only (no stderr tee); console is the fallback only when the file is disabled (`HARNESS_LOG_FILE="-"`) — stops log records mixing into the client's TUI; refines ADR-0018
- [0023-structured-task-reports-and-council-decision](0023-structured-task-reports-and-council-decision.md) - every driven task self-reports a schema'd `TaskReportEnvelope` (validated + stored as a `task_report`, raw output kept for audit); council runs a post-execution decision stage that consumes the normalized envelopes and emits a `CouncilDecision` (`get_reports`/`submit_decision`); still Module 1; refines ADR-0020/0021
