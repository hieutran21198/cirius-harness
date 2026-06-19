# Git conventions

## Branches

- `main` is the trunk. It always builds, lints, and tests cleanly.
- Feature branches: `<short-prefix>/<scope>/<short-description>`. Examples:
  - `feat/billing/add-invoice-export`
  - `fix/auth/cookie-expiry`
  - `docs/conventions/init`
- Long-lived branches other than `main` need an ADR.

## Commits

We use Conventional Commits. The shape that matters:

```
<type>(<scope>): <imperative subject>

<body, wrapped at ~72 cols, explaining the WHY>

<footers: BREAKING CHANGE, Closes #123, Co-authored-by, ...>
```

- `type`: one of `feat | fix | refactor | docs | test | chore | build | ci | perf | revert`.
- `scope`: the directory or subsystem affected (`billing`, `tenancy`, `deploy`, `nx`, ...).
- `subject`: imperative ("add", not "added" or "adds"). No trailing period.

Atomic commits beat dump-everything commits. A commit that touches `services/billing/`, `services/orders/`, and `packages/go/contracts/` belongs as **three** commits.

## Pull requests

- One reviewer minimum. Two when the change touches more than one bounded context.
- PR description points at the ADR (if any), the spec (if any), the ticket (if any). No essay needed - links are enough.
- Squash merge for feature branches; preserve commit chain for multi-step refactors only when the chain is genuinely informative.

## Anti-patterns

- "fix stuff" / "wip" / "asdf" as commit messages on `main`.
- Mixing unrelated changes in one commit because they're in the same file.
- Force-pushing to shared branches.
- Editing history of merged commits.
