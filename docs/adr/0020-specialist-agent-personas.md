# 0020. Specialist agent personas (harness-owned, archetype-aware)

- **Status**: Accepted
- **Date**: 2026-06-22
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (persona is a
  harness-owned domain constant resolved by name; v1 shipped only council's — this extends the
  registry to every working specialist and ties prompt *style* to the agent's archetype) and
  [ADR-0017](0017-council-orchestration-model.md) (council's roster names these specialists; now
  each has a behaviour to run as).

## Context

[ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) established that an agent's
behaviour is a harness-owned `domain.Persona` — code, not data — resolved by name via
`domain.PersonaFor` and rendered to the system prompt the client runs a turn under. v1 shipped
exactly one persona, **council**, and stated plainly that "other agents have none yet": `/council`
was the only resolve path, and council is plan-only, so the specialists it routes to (planner,
implementer, researcher, explorer, reviewer, scribe) had identity rows
([ADR-0007](0007-roles-and-per-session-model-binding.md)) but no behaviour. When council delegates
a task, or a future runtime executor (Module 2) drives a planned task, the harness has nothing to
hand the client that says *how that agent works* — the turn would run as the client's default
agent, not in-role.

Two facts shape the design:

- **Style is a function of archetype, not of agent.** `.cirius-harness/00-system.yaml`, the seed
  migration, and [docs/research/model-families.md](../research/model-families.md) classify each
  agent by archetype — `communicator` (Claude family: mechanics-heavy, checklisted prompts reward
  more rules), `principle-driven` (GPT family: concise principles + decision criteria; more rules
  = more contradiction surface = drift), `utility-runner` (cheap/fast scan, not deep reasoning).
  A persona's prompt should be written in the style its model family rewards, and the archetype is
  already seeded per agent.
- **No wire/query/handler change is required.** `query.ResolveAgent` already calls `PersonaFor`
  generically and returns the rendered persona (empty when none); the `pilink` `resolve_agent`
  frame handler is generic over agent name. Registering personas makes them resolve over the
  existing `resolve_agent` → `agent_resolved` frame with zero protocol or handler change.

## Decision

**Each working specialist gets a harness-owned persona, registered in the same
`domain.PersonaFor` registry as council, rendered by a shared archetype-aware `AgentProfile`.**

- **Shared type.** `domain.AgentProfile` (fields: agent, archetype, identity, mission, principles,
  output, boundaries, effort) implements `domain.Persona`. Its `SystemPrompt()` renders the same
  fixed anatomy in the style the `archetype` dictates: `communicator` profiles emit council-style
  uppercase section headers and numbered steps (reusing council's `section()` helper);
  `principle-driven` and `utility-runner` profiles emit quiet labels and bullets — concise, less
  scaffolding. The archetype thus binds prompt **style** to the model family the agent runs on,
  in one place.
- **Six instances.** planner, implementer, researcher, explorer, reviewer, scribe are added to the
  `personas` map. Council keeps its bespoke `CouncilProfile` (its orchestration framework is
  genuinely richer — intents, dimensions, a routing roster, a plan contract — and is governed by
  ADR-0017); both types implement `Persona`. `prayer` stays unregistered (model-less, archetype
  `none`).
- **Style matches the seed.** Each profile's `archetype` field must equal the archetype the agent
  is seeded with (single source: the seed migration / `00-system.yaml`). A unit test asserts this,
  so a profile cannot silently disagree with the model family it runs on.
- **Boundaries are behavioural, not permissions.** Each profile states its guardrails as intent
  ("you never edit source", "stop and surface decisions for human approval"), never as a permission
  grant. Casbin ([ADR-0003](0003-authorization-casbin-abac.md)) is the enforcer; the persona is a
  soft guard, as in ADR-0016.
- **No wire, query, or handler change; domain-only.** The personas resolve over the existing
  `resolve_agent` path and are immediately available to council's delegation and the future
  executor. **No client command is added in this change** — a human-facing invocation path (e.g. a
  generic `/agent <name> <ask>` in the Pi extension) is deferred to a follow-up.

## Consequences

- **Positive**: every working agent now runs in-role wherever it is resolved — council's
  delegation and the future executor get a real behaviour to hand the client, with no protocol
  change. Persona-as-code keeps behaviour versioned and unit-tested; the archetype-aware renderer
  keeps prompt style aligned to the model family without per-agent bespoke types (one shared type,
  one renderer). The drift guards (archetype-matches-seed, every-field-renders) make the personas
  self-checking.
- **Negative**: persona prose can drift from the real Casbin/config permissions (e.g. a persona
  saying "read-only" while config grants more). This is the same soft-guard caveat as ADR-0016,
  mitigated by phrasing boundaries as behavioural intent rather than permission verbs. A persona is
  a code change + redeploy per agent (intended). The archetype mapping now lives in two places
  (the seed and the profile) that must agree — the consistency test is the guard.
- **Neutral**: the registry grows from 1 to 7 entries. The specialist personas are not yet
  exercisable by a human (no command) until the deferred client wiring lands; they are reachable
  over the wire today and by the Module 2 executor later. `scribe`'s "knowledge store only" scope
  is behavioural until path-scoped permissions land (a `00-system.yaml` note).

## Alternatives considered

- **Bespoke profile type per agent** (six mini-`CouncilProfile`s) — rejected: the six specialists
  share one anatomy (identity/mission/principles/output/boundaries/effort); one `AgentProfile` with
  archetype-aware rendering makes the archetype→style mapping a single testable fact instead of six
  copies that drift. Council keeps a bespoke type only because its shape is genuinely richer.
- **Prose-only personas** (a hand-written string per agent, no struct) — rejected: loses the
  field-level drift guard (the reflection test) and the archetype-consistency check; the structured
  profile is what makes the persona testable.
- **Defer until the executor exists** (Module 2) — rejected: the personas resolve over the existing
  wire with zero new code paths; deferring would leave council able to *name* a delegate it cannot
  describe.
- **Add the client command now** (generic `/agent` or per-agent commands) — deferred, not taken in
  this change: it is an additive Pi-extension change that would generalize council's one-shot
  `before_agent_start` state and gate plan-capture behind a council-only flag. Kept separate so this
  change stays domain-only and does not touch the council plan-capture flow.

## References

- [ADR-0016](0016-harness-owned-agent-persona-governed-turn.md) (refined here — persona as a
  harness-owned domain constant; the `resolve_agent`/`agent_resolved` frames),
  [ADR-0017](0017-council-orchestration-model.md) (council's roster names these specialists),
  [ADR-0007](0007-roles-and-per-session-model-binding.md) (agents as roles),
  [ADR-0003](0003-authorization-casbin-abac.md) (permissions — the authoritative boundary the
  persona prose only soft-guards),
  [ADR-0015](0015-client-aware-model-catalog.md) (the client-specific model the resolver will bind).
- [model-families research](../research/model-families.md) (archetype → model family → prompt
  style), [agent-orchestration research](../research/agent-orchestration.md) (per-agent behaviour,
  effort scaling, delegation specificity), [PDR-0002](../pdr/0002-agent-team-composition.md) (the
  lean team these personas govern).
- `internal/domain/{agent_profile,persona}.go`, `internal/domain/agent_profile_test.go`,
  `docs/glossary/README.md`.
