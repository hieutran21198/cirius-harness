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

- **Construct aggregates through a package-level `New(...) (T, error)`.** It assembles the
  struct, applies creation defaults (e.g. a new `model` is `Enabled`, a new `session` is
  `pending`/`EnvUnset`), and returns `t, t.Validate()`. Callers never build an aggregate by
  setting fields one by one — that lets invariants leak into the caller.
- **`Validate()` enforces every invariant, including a non-empty surrogate `ID`.** A
  default-constructed aggregate must fail `Validate()`.
- **The id and clock come from the application/adapter, passed *into* `New`.** Mint the
  UUID v7 (and stamp timestamps) in the use case so the caller knows the id before the write
  ([persistence.md](persistence.md)); the domain never generates them. The app supplies them
  as `New` arguments rather than assigning fields after the fact.

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
