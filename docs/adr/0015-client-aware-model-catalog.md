# 0015. The model catalog is client-aware: client joins the natural key

- **Status**: Accepted
- **Date**: 2026-06-21
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0011](0011-client-reported-model-catalog.md) (the catalog is still
  client-reported and cumulative; only its key changes — it is no longer client-agnostic)

## Context

ADR-0011 made the catalog a **client-agnostic union** keyed on `(provider, slug)`, assuming
`(provider, slug)` is a namespace shared across clients. Testing against two clients showed it
is not — the same underlying model is reported under different names per client:

- Pi → `openai-codex/gpt-xxx`
- opencode → `openai/gpt`

The naming is a property of the **client's own model registry**, not a universal provider
taxonomy. Under the `(provider, slug)` key, two clients' models become unrelated rows with no
record of which client reported each, and any case where two clients *did* reuse a
`(provider, slug)` for different models would silently clobber on upsert.

The `models` frame already carries `client` ([ADR-0008](0008-pi-client-integration-stdio.md)),
but the harness discarded it (a stderr log only).

## Decision

**The client is part of a catalog entry's natural key: identity is `(client, provider, slug)`,
`UNIQUE(client, provider, slug)`.**

- A typed `domain.ClientKind` (`pi`, `opencode`, …) is a first-class field on `Model` and on the
  `Ref` natural-key value object. The `models` frame's `client` is now **required** and
  validated at the delivery edge — an unknown/missing client is an `error` frame.
- The catalog stays **client-reported and cumulative** (ADR-0011 otherwise stands): reported
  refs absent from the catalog insert with a minted id; nothing is deleted. The union is now
  **per-client**, not client-agnostic.
- Existing rows migrate by rebuilding the `models` table (SQLite can't drop the old `UNIQUE`),
  **preserving `id`** so the `session_agents.model_id → models(id)` FK stays valid; pre-existing
  rows are attributed to `pi` (the only client that has ever written).

Cross-client **unification** — recognising that Pi's `openai-codex/gpt-5` and opencode's
`openai/gpt-5` are the *same* model for handoff/fallback — is **out of scope**. It needs a
canonical-model + per-client-alias mapping and a source of truth for canonical names; a future
ADR will decide it if/when cross-client governance is built.

## Consequences

- **Positive**: each client's models are distinct, attributed rows; no cross-client clobber; the
  catalog is correct when one DB sees multiple clients (both clients in one project, or the
  future Module 2 motherboard). Provenance is recorded for that later unification work.
- **Negative**: the same real model appears as N rows across N clients with no link between them
  — deferred to the unification ADR. `ClientKind`'s known set must grow as clients are added.
- **Neutral**: under ADR-0009 a child-harness normally sees one client, so a single catalog rarely
  mixes clients today; this is the forward-compatible key regardless. Wire/schema change only —
  the Pi extension already sends `client: "pi"`.

## Alternatives considered

- **Client as a non-key provenance attribute** (keep `(provider, slug)` unique) — rejected: it
  labels rows but still lets two clients sharing a `(provider, slug)` collide; identity, not just
  provenance, is what differs.
- **Canonical model + per-client aliases** — deferred (not rejected): the right model for
  cross-client handoff, but far larger and needs a canonical-naming source of truth not required
  to fix the collision now.
- **Normalise providers instead of keying on client** — rejected: the divergence is a property of
  the client's registry, not a fixable provider-name skew; the client is the correct disambiguator.

## References

- [ADR-0011](0011-client-reported-model-catalog.md) (refined here),
  [ADR-0009](0009-deployment-topology-per-client-harness.md) (one child-harness per citizen),
  [ADR-0008](0008-pi-client-integration-stdio.md) (the `models` frame),
  [ADR-0005](0005-surrogate-uuid-v7-keys.md) (surrogate id; the natural key is separate).
- `docs/conventions/persistence.md` (catalog key), `docs/conventions/api.md` (the `models` frame),
  `docs/specs/harness-data-model.md` (the `models` table).
