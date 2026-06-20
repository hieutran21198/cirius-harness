# 0011. Models are client-reported, synced into a cumulative catalog

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

The `models` catalog was **hardcoded**: migration `20260619091805_seed_system_agents.sql`
seeds nine fixed `(provider, slug)` rows. But which models actually exist — and which are
usable — is a property of the **client**: Pi knows its configured/authenticated models
(`ctx.modelRegistry.getAvailable()`), and a different client (opencode, …) will know a
different set. A catalog the harness ships can only ever be wrong.

The end goal is that at session start the harness learns the client's models, then merges
configuration, resolves each agent to an *available* model, validates, and persists a
per-session config. This ADR records that **target flow** and decides the **first step**: stop
seeding models; learn them from the client.

This is also the first real **DB write-path through the hexagon** — before it, only repository
interfaces and the Casbin adapter existed.

## Decision

**Models are client-reported and synced into a global, cumulative catalog.**

- At `session_start` the client (its harness extension) sends its enabled models over the
  pilink stdio channel ([ADR-0008](0008-pi-client-integration-stdio.md)) as a `models` frame;
  the harness replies `models_synced{added, total}`.
- The catalog is **global and cumulative**: reported `(provider, slug)` refs not already
  present are inserted with a freshly minted UUID v7 (`enabled = 1`); **nothing is deleted**.
  It is the client-agnostic union across all clients and sessions. `(provider, slug)` is the
  upsert key (matching the table's `UNIQUE` constraint).
- The hardcoded model seed is **removed** by a new migration (append-only — the seed migration
  is left intact; the new one reverses just its model rows).
- `harness serve` **applies migrations on start**, so the per-session child is self-sufficient
  (no out-of-band `db:migrate` needed before sync works).

### Target flow (recorded; implemented incrementally)

1. **Sync** the client's models into the catalog. ← *this slice*
2. **Merge** config: system config moves to embedded
   `services/harness/assets/00-system-config.yaml`; an optional user overlay
   `.cirius-harness/config.yaml` deep-merges on top.
3. **Resolve** each agent to one *available* model (primary → ordered `fallbacks`).
4. **Validate**: if an agent's primary and all fallbacks are unavailable, **skip that agent
   and warn** in the client; the session still starts.
5. **Persist** the session config to the runtime tables — a `sessions` row keyed by a
   **client-agnostic** `(client_kind, client_session_id)` (Pi's `sessionManager.getSessionId()`),
   plus `session_agents` model bindings. **Resume** looks the session up by that key; if absent,
   it behaves like a new session.

## Consequences

- **Positive**: the catalog reflects reality (what the client can actually run); the harness
  ships no model list to drift. The cumulative union is dead-simple and naturally multi-client.
  Establishes the first GORM store + write-path, the spine the later slices extend.
- **Negative**: stale entries linger (a model a client stops offering stays `enabled`); a
  later reconciliation/expiry may be wanted. Auto-migrate-on-serve means a malformed migration
  fails the session rather than a separate step — acceptable for a per-session child.
- **Neutral**: this refines the spec's "the seed migration is done" note and the
  seeded-catalog assumption behind [ADR-0006](0006-model-catalog-and-agent-profiles.md) /
  [ADR-0007](0007-roles-and-per-session-model-binding.md); agents-as-roles and per-session
  model binding are unchanged.

## Alternatives considered

- **Mirror the current client** (sync sets `enabled=1` for reported, `0` for the rest) —
  rejected: one client's view would clobber the catalog each connect, breaking the multi-client
  goal.
- **Per-session availability snapshot** (a `session_models` table) — deferred: useful for
  reproducibility, but not needed for Slice 1; the cumulative catalog + a later snapshot can
  coexist.
- **Keep seeding a default catalog** — rejected: the harness cannot know any client's real,
  authed model set; a shipped list is guaranteed wrong for someone.

## References

- [ADR-0008](0008-pi-client-integration-stdio.md) (the stdio channel this rides),
  [ADR-0007](0007-roles-and-per-session-model-binding.md) (agents as roles, per-session model
  binding), [ADR-0010](0010-ts-build-pipeline-apps-to-pi-extensions.md) (the extension app).
- `docs/conventions/api.md` (the `models` / `models_synced` frames),
  `docs/specs/harness-data-model.md` (catalog vs runtime tables).
- Pi 0.79.6: `ctx.modelRegistry.getAvailable()` (enabled/authed models),
  `ctx.sessionManager.getSessionId()` + `session_start.reason` (used by the later resume slice).
