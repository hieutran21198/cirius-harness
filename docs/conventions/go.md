# Go conventions

Applies to every Go module in the repo: services, shared packages, and tools.

## Style

- `gofmt` + `goimports` are non-negotiable; CI rejects anything else.
- `golangci-lint` (**v2**) runs per module. The curated baseline lives in each module's
  `.golangci.yml`: `default: standard` plus high-signal linters that reinforce these
  conventions — `errorlint` (`%w` + `errors.Is/As`), `revive` (exported doc-comments,
  error/context naming), `gocritic`, `unconvert`, `unparam`, `bodyclose`, `misspell`,
  `nakedret`, `wastedassign`, `copyloopvar`, plus `govet`'s `shadow` analyzer (catches a
  variable — typically `err` — shadowing one from an enclosing scope) — with `gofmt`/`goimports` as formatters
  (`goimports.local-prefixes: harness-workspace`). `revive`'s `unused-parameter` is excluded
  on purpose: `ctx` stays the named first parameter even when unused (see Concurrency). The
  two module configs are kept **identical** — when adding a module, copy a sibling's verbatim.
- Package names are short, lower-case, no underscores. Directory and package name match.
- Public identifiers carry a doc comment that starts with the identifier name. `// OrgID identifies ...` not `// Identifies the ...`.

## Domain model

The domain is **one package** (`internal/domain`), not a package per aggregate
([ADR-0014](../adr/0014-domain-encapsulation-single-package.md)): cross-aggregate logic
collaborates without import ceremony. Because everything shares a package, identifiers are
**prefixed** to stay unambiguous — `NewModel`/`NewAgent`, `ContainerStatus`/`SessionStatus`
(+ `ContainerPending`/`SessionPending` constants), `ToolName`, `domain.ModelWriter`.

- **Aggregates expose no public state.** Fields are unexported; domain *meaning* is reached
  through intention-revealing methods, never by reading or setting a field from another layer.
  Value objects and typed-string enums (`Ref`, `Kind`, `Archetype`, `EnvType`, `Action`,
  `Decision`) are the exception — they are equal to their value and stay public-representation.
- **Two constructors per aggregate.** `NewXxx(...)` is **fresh creation** (used in the app): it
  takes the business attributes only, **mints its own identity** (a UUID v7, via the shared
  unexported `newID()`), applies creation defaults (a new `Model` is enabled, a new `Session` is
  `pending`/`EnvUnset`), and returns `t, t.Validate()`. `RehydrateXxx(...)` is **reconstitution
  from storage** (used in the repo): it takes every persisted field as-is — including the stored
  `id` — no defaults, and validates. Callers never build an aggregate field by field.
- **`Validate()` enforces every invariant, including a non-empty surrogate `ID`.** A
  default-constructed aggregate must fail `Validate()`. (Since `NewXxx` mints the id, an empty
  id is only reachable on the `RehydrateXxx` path — a corrupt row.)
- **Identities are typed per aggregate, not bare `string`.** Each aggregate declares a named
  `~string` id (`ModelID`, `AgentID`, `ProjectID`, …); fields, constructor parameters, and
  cross-aggregate references use it (`Member{ agentID AgentID; modelID ModelID }`), so passing
  one aggregate's id where another's is expected is a compile error — the same "no poor-man's
  primitives" rule the typed-string enums follow, for ~zero runtime cost. `newID[T ~string]()`
  mints the right type (`id: newID[ModelID]()`). The one exception is a **polymorphic**
  reference (`Session.envID`, a `WorktreeID` *or* `ContainerID` chosen by `envType`), which
  stays `string` because a single field can't carry both — integrity is in `Validate()`.
- **Fresh aggregates own their identity; the clock comes from the application/adapter.** The id
  *format* (UUID v7) is a domain policy, so `NewXxx` mints it internally — the app supplies only
  business attributes (`domain.NewModel(provider, slug)`) and never imports `uuid`. The id is
  still minted in-process before the write, so the caller can read it back from the aggregate
  (`Snapshot()`), and it is never DB-generated ([persistence.md](persistence.md), ADR-0005).
  Timestamps are still stamped in the use case and passed into `NewXxx`.
- **State leaves the domain only through a domain-owned grouped view with a clear purpose.**
  The persistence view is a `Snapshot()` returning a flat memento (e.g. `ModelSnapshot`) that the
  repo maps to a row and `RehydrateXxx` mirrors back — never per-field getters. A different
  purpose (UI/API projection) gets its own named view, added when a consumer needs it.
- **Domain construction tests are white-box (`package domain`)** so they can assert on
  unexported fields and creation defaults — a deliberate exception to the external-test rule
  under Testing below.

## Errors

- Sentinel errors at package level: `var ErrNotFound = errors.New("foo: not found")`.
- Wrap with context: `return fmt.Errorf("repo.LoadInvoice: %w", err)`.
- Test with `errors.Is` / `errors.As`, never string comparison.
- Panics belong in init code and truly impossible branches. Domain code returns errors.

## Concurrency

- `context.Context` is the first parameter of any function that may block, IO, or call further into the stack. Pass it - never store it on a struct.
- Goroutines have a clear owner that knows how to cancel them. No fire-and-forget without `errgroup` or equivalent.
- Channel direction is restricted at function signatures (`chan<- T`, `<-chan T`) wherever it helps the reader.

## Testing

- `_test.go` next to the code under test. External package tests (`package foo_test`) when exercising the public API.
- Table-driven tests when input/output varies; nested `t.Run` so failures point at the case.
- `t.Parallel()` at the top of every test that doesn't share state.
- No network / disk / DB in unit tests. Integration tests use build tags and run in a separate target.

## Anti-patterns

- `interface{}` / `any` without a doc-justified reason.
- Empty structs as poor-man's enums - use typed string constants in a small package.
- Returning `(T, bool)` where the bool means "ok" - return `(T, error)` with a sentinel.
- `init()` that does I/O, opens connections, or mutates globals.
- `replace` directives committed to release branches.
