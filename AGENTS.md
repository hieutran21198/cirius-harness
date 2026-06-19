# Workspace Knowledge Base

## Mission

cirius-harness is a **control plane** over an AI coding client — it governs the client; it
does not call models directly. Three goals:

- **Harness AI-coding** — a control plane over an AI coding client (opencode is the
  reference), not a provider client itself. _(Module 1 — `services/harness`)_
- **AI-agents orchestration** — a declarative agent team mapped to model families, with
  per-agent permissions, tools, and fallbacks. _(Module 1)_
- **Concurrency works** — view and control concurrent work across multiple harness
  modules. _(Module 2 — reserved, see [ADR-0001](docs/adr/0001-harness-layout.md))_

## Layout (authoritative)

```
├── docs/                   # Documentations: ADRs, specs, conventions, glossary...
├── packages/
│   ├── go/                 # ONE Go module - subdirs are packages, not modules
│   │   └── <name>/         # Shared package (import .../packages/go/<name>)
│   └── ts/libs/<name>/     # nx-managed shared TS libraries
├── services/<name>/        # Deployable backend service. Own go.mod. Hexagonal + DDD.
├── devenv.{nix,yaml}       # Sole source of truth for dev tooling
├── go.work                 # Lists every Go module
├── nx.json                 # nx workspace
├── package.json            # Root JS workspace
└── pnpm-workspace.yaml     # pnpm workspaces
```

## Stack

- **Backend**: Go 1.26.3 or TypeScript on Node 22 - managed by nx + pnpm workspaces, hexagonal architecture per service, DDD-style bounded contexts
- **Dev env**: devenv (Nix) - **never write a Makefile**, declare new tasks in `devenv.nix`
- **Module prefix (placeholder)**: `harness-workspace` - rename once when forking
- **Module split** (per [ADR-0001](docs/adr/0001-harness-layout.md)): Module 1 = `services/harness` (the harness engine, built now); Module 2 = multi-module concurrency view/control (reserved, future).

## Where to look

| Task                                   | Location                                        |
| -------------------------------------- | ----------------------------------------------- |
| Add a deployable backend service       | `services/`                                     |
| Add a shared Go package                | `packages/go/`                                  |
| Add a shared TS library                | `packages/ts/`                                  |
| Add a UI application                   | `apps/`                                          |
| Change deployment / environment config | `deploy/`                                        |
| Record evidence on a model / tool / client | [docs/research/](docs/research/README.md)   |
| Record a what-to-use decision          | [docs/pdr/](docs/pdr/README.md)                 |
| Record an architectural decision       | [docs/adr/](docs/adr/)                          |
| Define a cross-team domain term        | [docs/glossary/](docs/glossary/README.md)       |
| Change a workspace-wide convention     | [docs/conventions/](docs/conventions/README.md) |
| Write a feature / system design        | [docs/specs/](docs/specs/README.md)             |

## Conventions

- **One Go module per service. ONE Go module for all shared packages (`packages/go/`).** Services are independent deploy artifacts; shared internal packages have no external consumers and don't need independent versioning. Escape hatch documented in `packages/go/AGENTS.md`.
- **`devenv tasks run <name>` is the only task entrypoint.** No Makefiles, no ad-hoc shell scripts at root.
- **ADRs are append-only.** A decision is changed by writing a new ADR that supersedes the old one, never by editing history.
- **Hexagonal in services means dependencies point inward.** Layout is `internal/{domain,port,application,adapter}`, with `port` and `adapter` split into `inbound`/`outbound` ([ADR-0004](docs/adr/0004-ports-and-adapters-topology.md)). The direction is `adapter → application → port → domain`; nothing inner imports `adapter`. The layout + linter rule are documented in `services/AGENTS.md`.

## Anti-patterns (this workspace)

- **Cross-service Go imports.** Services NEVER import other services. Communicate through API contracts in `packages/go/<contract>` or via the wire (CLI command / MCP / events).
- **Cross-app TS imports.** Apps NEVER import another app. Share through `packages/ts/libs/<lib>`.
- **Root-level service code.** All deployable code lives under `services/` (Go) or `apps/` (UI). The root only carries workspace plumbing.
- **Adding a Makefile.** devenv tasks exist for exactly this. Search for `tasks =` in `devenv.nix` before adding a script.
- **`replace` directives in published go.mod files.** Local development uses the root `go.work`; never ship `replace` to a release branch.
- **Editing files inside `.devenv*`, `.direnv`, `node_modules`, `.nx/cache`.** Generated. Always.

## Commands

`devenv tasks run <name>` is the canonical entrypoint (defined in `devenv.nix`, loaded by
direnv) — it fans the action across **every** Go module and nx-managed TS lib:

| Task                              | Does                                                        |
| --------------------------------- | ----------------------------------------------------------- |
| `devenv tasks run workspace:bootstrap` | `pnpm install` + `go work sync`                        |
| `devenv tasks run workspace:fmt`  | `gofmt` all Go modules + `nx format:write`                  |
| `devenv tasks run workspace:lint` | `golangci-lint run ./...` per Go module + `nx run-many -t lint` |
| `devenv tasks run workspace:test` | `go test ./...` per Go module + `nx run-many -t test`       |
| `devenv tasks run db:migrate` / `db:rollback` / `db:status` | embedded-goose migrate `up` / `down` / `status` for the harness DB |

For tighter loops, work **inside a single module** (each `services/<svc>` and `packages/go`
has its own `go.mod`):

```bash
cd services/harness
go build ./...                       # NOTE: `go build ./...` from the repo ROOT fails —
                                     #   go.work has no module at "."; always build per-module
go test ./...                        # all tests in this module
go test ./internal/domain/agent/...  # one package
go test -run TestAgentValidate ./internal/domain/agent/   # one test
golangci-lint run ./...              # lint this module (per-service .golangci.yml)
go run ./cmd/migrate <up|down|status|version|create <purpose>>   # migration CLI
```

## Notes / gotchas

- `go.work` lists modules explicitly. If `go build ./...` doesn't see a new module, you forgot `go work use ./path/to/module`.
- nx's `workspaceLayout` is overridden to `packages/ts/libs` - generators that hard-code `libs/` will land in the wrong place. Always pass `--directory=packages/ts/libs/<name>` to `nx generate`.
- `.pre-commit-config.yaml` is gitignored. The intent: each contributor wires hooks locally without forcing a shared list. If you want enforcement, move it out of `.gitignore` and propose via an ADR.

<!-- CODEGRAPH_START -->
## CodeGraph

In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or reading files when you need to understand or locate code:

- **MCP tools** (when available): `codegraph_explore` answers most code questions in one call — the relevant symbols' verbatim source plus the call paths between them. `codegraph_node` returns one symbol's source + callers, or reads a whole file with line numbers. If the tools are listed but deferred, load them by name via tool search.
- **Shell** (always works): `codegraph explore "<symbol names or question>"` and `codegraph node <symbol-or-file>` print the same output.

If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
<!-- CODEGRAPH_END -->
