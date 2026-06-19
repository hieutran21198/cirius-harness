# Go conventions

Applies to every Go module in the repo: services, shared packages, and tools.

## Style

- `gofmt` + `goimports` are non-negotiable; CI rejects anything else.
- `golangci-lint` runs per module. The shared baseline lives in each module's `.golangci.yml` (start by copying from a sibling).
- Package names are short, lower-case, no underscores. Directory and package name match.
- Public identifiers carry a doc comment that starts with the identifier name. `// OrgID identifies ...` not `// Identifies the ...`.

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
