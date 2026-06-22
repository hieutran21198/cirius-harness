# 0016. Harness-owned agent persona, run as a governed one-shot turn via Pi

- **Status**: Accepted
- **Date**: 2026-06-21
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0007](0007-roles-and-per-session-model-binding.md) (agents are roles;
  this adds the role's *behaviour* — its persona — as harness-owned **code** (a domain
  constant), still no stored per-agent model)

## Context

We want a `/council` slash command in Pi: the user types `/council <message>` and the
**council** agent weighs the request and produces a strategy plan. This is the first slice
of agent **governance** over the Pi wire (deferred since [ADR-0008](0008-pi-client-integration-stdio.md)).

Two facts shape the design:

- **The harness does not call models** (AGENTS.md mission): it is a control plane over a
  client, not a provider client. So `/council` cannot have the harness produce the plan.
- **A Pi extension cannot run a model turn in its command handler** (verified against
  `@earendil-works/pi-coding-agent@0.79.8`): there is no "submit prompt, await response"
  API. But it *can govern* a turn — `pi.setModel(...)`, a `before_agent_start` hook that
  **replaces the system prompt for that one turn**, and `pi.sendUserMessage(text)` which
  **triggers** the turn. The model's reply lands in the normal transcript.

An agent role (archetype, tools, permissions) is client-agnostic; what it *does* is a
system prompt. Today agents carry identity only ([ADR-0007](0007-roles-and-per-session-model-binding.md));
there is nowhere for the harness to express council's planning behaviour. A persona is not
workspace data the user tunes — it is the harness's own definition of how an agent behaves, so
it belongs with the harness **code**, versioned and reviewed with it, not in the mutable store
or the user-editable config.

## Decision

**An agent's behaviour is a harness-owned `persona` — a structured domain constant; the client
runs it as a governed one-shot turn. The harness says *who the agent is*; Pi *executes* the turn.**

- `persona` is a structured `domain.Persona` value (identity, mission, effort-scaling rule, the
  fixed output sections, and a delegation roster), declared as a **domain constant** and looked
  up by agent name via `domain.PersonaFor`. It is **not** persisted (no DB column) and **not** in
  `.cirius-harness/*.yaml`; it is code, rendered to a system-prompt string at resolve time. The
  structured profile lets a test keep council's delegation roster in sync with the real team.
  Council's persona is the strategy-planning prompt (its content follows the
  [orchestration research](../research/agent-orchestration.md)); other agents have none yet.
  Persona is distinct from the **model** (bound per session, ADR-0007) and **permissions**
  (Casbin, ADR-0003).
- A new read-side query `ResolveAgent(name, client)` confirms the agent exists and is enabled
  (via `domain.AgentReader` / `query.ReadStore`, ADR-0013 — governance: don't serve a persona for
  an unknown/disabled role), then attaches the persona resolved from the domain registry. Exposed
  on a new stdio frame pair: `resolve_agent` → `agent_resolved` (ADR-0008 framing).
- The Pi extension's `/council` handler resolves council, arms a one-shot flag + persona, and
  calls `sendUserMessage(message)`; a `before_agent_start` hook returns `{systemPrompt: persona}`
  for that turn only, then reverts. Inline in the current session.
- **Model governance is out of scope here.** `agent_resolved` carries a `model` field, but the
  query leaves it empty: resolving an agent's client-specific model against the synced catalog
  ([ADR-0015](0015-client-aware-model-catalog.md)) needs the config loader/resolver, which is
  unbuilt. v1 runs `/council` on the active model; the persona instructs "plan only, do not edit"
  as a soft guard until **permission** governance (Casbin agent-policy seeding) also lands.

## Consequences

- **Positive**: the control-plane split is real — the harness owns and serves agent behaviour;
  the client runs it without the harness ever touching a model. Persona-as-code means it is
  versioned and reviewed with the harness, the structured profile is unit-testable (the
  delegation roster is checked against the real team), and there is no migration/schema surface
  for behaviour. The wire frame is forward-compatible: the resolver fills `model` later with no
  protocol change.
- **Negative**: v1 governs persona only — model and permissions are advisory (persona text) until
  their milestones. Changing a persona is a code change + redeploy, not a data edit — acceptable
  and intended (behaviour is harness-owned). A one-shot module flag in the extension is
  order-dependent on `sendUserMessage` triggering exactly the next `before_agent_start`;
  acceptable because the handler arms and fires in the same tick.
- **Neutral**: adds the read side (`query` package, `AgentReader`, `readstore`) the service did
  not have — needed eventually regardless; here it provides the exists/enabled governance check.
  The `agents` table is unchanged (no persona column).

## Alternatives considered

- **Persona as an `Agent` field / DB column** (seeded from `00-system.yaml` via migration) —
  rejected: behaviour is not workspace data the user tunes; storing it as mutable state invites
  drift, adds a migration surface, and is not testable as structure. A domain constant keeps
  behaviour with the code that defines it.
- **Persona hardcoded in the extension** — rejected: agent behaviour would live outside the
  control plane; the harness would govern model/permissions but not what the agent does.
- **Forked/isolated council session** (`fork` + `withSession`) — deferred: cleaner separation but
  more moving parts; inline is simplest and the one-shot revert keeps the working session clean.
- **Resolve the model now too** — deferred: needs the config loader/resolver + its own decision;
  folding it in would drag unbuilt multi-client config work into this slice.

## References

- [ADR-0007](0007-roles-and-per-session-model-binding.md) (agents as roles; refined here),
  [ADR-0008](0008-pi-client-integration-stdio.md) (stdio frames),
  [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md) (read-side ports: `AgentReader`,
  `ReadStore`), [ADR-0015](0015-client-aware-model-catalog.md) (the client-specific model the
  resolver milestone will bind), [ADR-0003](0003-authorization-casbin-abac.md) (permissions,
  the other deferred governance milestone).
- [agent-orchestration research](../research/agent-orchestration.md) (the patterns council's
  persona applies). `docs/conventions/api.md` (the `resolve_agent`/`agent_resolved` frames),
  `docs/specs/harness-data-model.md` (persona is a harness-owned domain constant, not an
  `agents` column).
