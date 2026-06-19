# Conventions

Workspace-wide rules. If something here conflicts with a per-service `AGENTS.md`, the per-service rule wins **only** for that service - the workspace rule still applies everywhere else.

## Index

- [architecture.md](architecture.md) - hexagonal layout: inbound/outbound ports & adapters
- [git.md](git.md) - branch + commit + PR rules
- [go.md](go.md) - Go style, error handling, testing
- [api.md](api.md) - CLI command (cmd) / MCP / event contract rules
- [persistence.md](persistence.md) - SQLite + GORM, repositories, schema, migrations

## When to add a convention

You add a convention here when:

1. The rule applies to **more than one service or app**.
2. People keep getting it wrong, AND the right behaviour isn't obvious from the existing code.

Otherwise, write a comment near the code instead.

## When to remove a convention

When it's no longer true. Either the tooling now enforces it (good, remove the prose), or we've decided differently (good, write an ADR and remove the prose). Stale conventions are worse than no conventions.
