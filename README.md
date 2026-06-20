# cirius-harness

A **control plane over an AI coding client** (opencode is the reference). cirius-harness
governs the client — it declares an agent team, binds roles to models, and orchestrates
concurrent work. It does **not** call models directly.

Three goals:

- **Harness AI-coding** — a control plane over an AI coding client, not a provider client.
  _(Module 1 — `services/harness`)_
- **AI-agents orchestration** — a declarative agent team mapped to model families, with
  per-agent permissions, tools, and fallbacks. _(Module 1)_
- **Concurrency works** — view and control concurrent work across harness modules.
  _(Module 2 — reserved, see [ADR-0001](docs/adr/0001-harness-layout.md))_

## Layout

```
├── docs/                # ADRs, specs, conventions, glossary, research, PDRs
├── apps/<name>/         # nx-managed apps — client adapters & UIs (e.g. the Pi extension)
├── packages/
│   ├── go/              # ONE Go module — shared packages (gormdb, migrate, casbinx, slogx)
│   └── ts/libs/         # nx-managed shared TS libraries
├── services/<name>/     # Deployable backend service — own go.mod, hexagonal + DDD
├── deploy/              # Deployment / environment config
├── devenv.{nix,yaml}    # Sole source of truth for dev tooling (Nix)
├── go.work              # Lists every Go module
└── nx.json / package.json / pnpm-workspace.yaml   # JS/TS workspace
```

`services/harness` is the harness engine (Module 1, built now). Module 2 (multi-module
concurrency view/control) is reserved.

## Stack

- **Backend**: Go 1.26.3 or TypeScript on Node 22, managed by nx + pnpm workspaces.
- **Architecture**: hexagonal per service (`internal/{domain,port,application,adapter}`,
  ports/adapters split inbound/outbound — [ADR-0004](docs/adr/0004-ports-and-adapters-topology.md)),
  DDD-style bounded contexts.
- **Persistence**: SQLite via pure-Go GORM + goose migrations; UUID v7 surrogate keys
  ([ADR-0005](docs/adr/0005-surrogate-uuid-v7-keys.md)). Authorization via Casbin
  ([ADR-0003](docs/adr/0003-authorization-casbin-abac.md)).
- **Dev env**: [devenv](https://devenv.sh) (Nix) — the only place tooling and tasks are declared.

## Getting started

Tooling is provided by devenv and loaded automatically by direnv on `cd` into the repo
(`.envrc`). Then bootstrap the workspace:

```bash
devenv tasks run workspace:bootstrap   # pnpm install + go work sync
```

## Commands

`devenv tasks run <name>` is the canonical entrypoint — it fans actions across **every** Go
module and nx-managed TS lib. Never add a Makefile; declare tasks in `devenv.nix`.

| Task                              | Does                                                             |
| --------------------------------- | --------------------------------------------------------------- |
| `workspace:bootstrap`             | `pnpm install` + `go work sync`                                 |
| `workspace:fmt`                   | `gofmt` all Go modules + `nx format:write`                      |
| `workspace:lint`                  | `golangci-lint run` per Go module + `nx run-many -t lint`       |
| `workspace:test`                  | `go test ./...` per Go module + `nx run-many -t test`           |
| `db:migrate` / `db:rollback` / `db:status` | goose `up` / `down` / `status` for the harness DB     |

For tighter loops, work inside a single module (`go build ./...` from the repo root fails —
`go.work` has no module at `.`; always build per-module):

```bash
cd services/harness
go build ./... && go test ./...
golangci-lint run ./...
```

## Documentation

| Topic                          | Location                                          |
| ------------------------------ | ------------------------------------------------- |
| Architectural decisions        | [docs/adr/](docs/adr/)                            |
| Feature / system designs       | [docs/specs/](docs/specs/README.md)              |
| Workspace-wide conventions     | [docs/conventions/](docs/conventions/README.md)  |
| Domain terms                   | [docs/glossary/](docs/glossary/README.md)        |
| What-to-use decisions / evidence | [docs/pdr/](docs/pdr/README.md) · [docs/research/](docs/research/README.md) |

The full knowledge base for contributors (and AI agents) lives in
[AGENTS.md](AGENTS.md) — conventions, anti-patterns, and gotchas.
