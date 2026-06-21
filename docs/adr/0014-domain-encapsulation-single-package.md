# 0014. Single `domain` package + aggregate encapsulation (New/Rehydrate + grouped views)

- **Status**: Accepted
- **Date**: 2026-06-21
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -
- **Refines**: [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md) (the shape of the
  `internal/domain` layer; ADR-0013's `domain/app/delivery/infra` split and UnitOfWork stand)

## Context

ADR-0013 placed the domain under `internal/domain` with per-aggregate Reader/Writer
interfaces. In practice each aggregate lived in its own sub-package
(`internal/domain/{model,agent,session,…}`). Two problems surfaced as the model grew:

1. **Cross-aggregate ceremony.** A session references an agent, a model, and a project. With
   one package per aggregate, any logic spanning them needs cross-imports — and Go's
   no-import-cycles rule makes mutual references between bounded contexts awkward.
2. **Leaky aggregates.** Aggregates exposed public fields (`m.ID`, `m.Enabled`, …), so the
   app and infra layers read and could mutate domain state directly. Construction defaults
   (`Enabled = true`) had started leaking into the app, and there was no distinction between
   *creating* an aggregate and *reconstituting* one from storage.

## Decision

**One `internal/domain` package; aggregates encapsulate their state.**

- **Collapse the per-bounded-context sub-packages into a single `package domain`.**
  Encapsulation moves from the *package* boundary to the *type* boundary — the boundary that
  matters is "app/infra/delivery must not see raw state," which unexported fields enforce,
  while domain code collaborates across aggregates freely. Name collisions are resolved by
  prefixing (`NewModel`/`NewAgent/…`; `ContainerStatus`/`WorktreeStatus`/`SessionStatus`;
  `ToolName`; the `model.Writer` interface becomes `domain.ModelWriter`). Persisted string
  *values* are unchanged — only Go identifiers move.
- **Aggregates hold unexported fields and expose meaning through methods.** No public state.
- **Two constructors per aggregate.** `NewXxx(...)` is fresh creation in the application —
  it applies creation defaults and validates. `RehydrateXxx(...)` is reconstitution from
  storage in the repository — it takes every persisted field as-is (no defaults) and
  validates structural integrity.
- **State leaves the domain only through a domain-owned grouped view with a clear purpose.**
  The persistence view is a `Snapshot()` returning a flat memento (e.g. `ModelSnapshot`); the
  repository maps it to a row, and `RehydrateXxx` mirrors its fields back. Any other purpose
  (a UI projection, an API DTO) gets its own named view when that need arises — the memento is
  not a general-purpose getter.
- **Value objects and enums stay public-representation immutable values.** `Ref` (the
  `(provider, slug)` natural key) and the typed-string enums (`Kind`, `Archetype`, `EnvType`,
  `Action`, `Decision`, …) are equal to their value and carry no mutable identity, so exposing
  their representation is not leaking state. `Ref` remains comparable (a map key) and trivially
  constructible at the wire edge.
- **Grouped views are built when a consumer needs them, not speculatively.** Only `Model` is
  wired through a repository today, so only `Model` has a `Snapshot` view. The other seven
  aggregates get `NewXxx`/`RehydrateXxx` now and a view when a use case consumes them.
- **Fresh aggregates mint their own identity inside `NewXxx`** (via the shared unexported
  `newID()` → `uuid.Must(uuid.NewV7())`). The id *format* is a domain policy, so the application
  constructs with business attributes only (`domain.NewModel(provider, slug)`) and never imports
  `uuid`; it reads the new id back from the aggregate (`Snapshot()`) when it needs it.
  `RehydrateXxx` takes the stored id. This refines
  [ADR-0005](0005-surrogate-uuid-v7-keys.md)'s "generated in the application/adapter" to "minted
  by the domain at construction": the id still exists on the in-memory aggregate before the write
  and is never DB-generated, so ADR-0005's core stands and its body is left intact. The domain
  consequently depends on `github.com/google/uuid` (a pure identity library, not infrastructure).
- **Aggregate identities are typed `~string`s, not bare `string`.** Each aggregate declares a
  named id type (`ModelID`, `AgentID`, `ProjectID`, `SessionID`, `MemberID`, `WorktreeID`,
  `ContainerID`, `ToolID`); fields, constructor parameters, and cross-aggregate references use
  it, so passing one aggregate's id where another's is expected is a compile error — the same
  rule the typed-string enums already follow, extended to identity, for ~zero runtime cost
  (`newID[T ~string]()` mints the right type). The grouped view carries the typed id; the only
  flattening to `string` is the infra row mapper's `string(snap.ID)` cast. The lone exception is
  a **polymorphic** reference (`Session.envID`, a `WorktreeID` *or* `ContainerID` selected by
  `envType`), which stays `string` because a single field cannot carry both id types — the
  no-FK, validate-in-the-domain case from [conventions/persistence.md](../conventions/persistence.md).

## Consequences

- **Positive**: cross-aggregate logic needs no import ceremony; aggregates can't be mutated or
  half-constructed from outside the domain; the create-vs-reconstitute distinction is explicit;
  the repository depends on a stable grouped view, not field layout.
- **Negative**: a second revision of the domain-layer shape (ADR-0013 → 0014); prefixed
  identifiers are more verbose (`SessionPending` vs `StatusPending`); domain construction tests
  must be white-box (`package domain`) to read unexported fields, a deliberate exception to the
  "external package tests for the public API" convention.
- **Neutral**: behavior, schema, and wire protocol are unchanged — the change is mechanical
  (rename + field lowercasing + view indirection). ADR-0013's layout and UnitOfWork stand.

## Alternatives considered

- **Keep per-aggregate packages** — rejected: the cross-aggregate import ceremony is exactly
  what the user wants gone, and inter-context references risk import cycles.
- **Encapsulate value objects too (`NewRef` + accessors)** — rejected: a value object is its
  value; hiding `Ref`'s fields adds accessor ceremony at every construction site for no
  invariant gain.
- **Whole-state getters instead of a grouped memento** — rejected: per-field getters re-expose
  state piecemeal, the opposite of the goal. One purpose-named view is the seam.
- **A `Snapshot` view on all eight aggregates now** — deferred: speculative export surface on
  aggregates with no consumer (same "build it when a use case needs it" rule as ADR-0013).

## References

- [ADR-0013](0013-idiomatic-go-layout-and-unit-of-work.md) (refined here),
  [ADR-0005](0005-surrogate-uuid-v7-keys.md) (id minted in the app, passed into `NewXxx`),
  [ADR-0011](0011-client-reported-model-catalog.md) (the model-sync use case).
- [conventions/go.md](../conventions/go.md) (Domain model section),
  [conventions/persistence.md](../conventions/persistence.md).
