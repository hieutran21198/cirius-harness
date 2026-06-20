# deploy/

How a harness-governed coding client is **deployed and operated**. This is the knowledge
base; the per-client runbooks live in `deploy/<citizen>/README.md`.

## Model

A **citizen** is a deployed AI coding client governed by the harness. **Pi** is the first
citizen (`deploy/pi/`).

The topology is **1:1 — one client ⇄ one child-harness**
([ADR-0009](../docs/adr/0009-deployment-topology-per-client-harness.md)). Entering the client
launches `harness serve` as a child process and connects over stdio
([ADR-0008](../docs/adr/0008-pi-client-integration-stdio.md)); the harness lives and dies with
that client session. There is no shared runtime today.

```
            ┌─────────────── deployment unit (a "citizen") ───────────────┐
            │   client (Pi)  ──stdio NDJSON──▶  child-harness (harness serve)
            └─────────────────────────────────────────────────────────────┘
                                   │  (future)
                                   ▼
                          motherboard service  ◀── many child-harnesses connect up
                          (Module 2 — reserved, not built)
```

The **motherboard** is the future central service that child-harnesses connect *up* to for
the cross-client / concurrency view — Module 2, reserved in
[ADR-0001](../docs/adr/0001-harness-layout.md). The 1:1 pairing below it does not change when
it arrives. Until then, each child-harness is blind to the others.

## Layout

```
deploy/
├── AGENTS.md                       # this file — the deployment model
├── <citizen>/                      # per-client runbook + config (pi is the first)
│   └── pi/  README.md  settings.example.json
└── environment/<env>/              # environment overlays
    ├── local/   README.md  harness.env.example
    └── prod/    README.md          # deferred (documented stub)
```

- `deploy/<citizen>/` — how to run that client governed by the harness, and its client-specific
  config templates. Adding a client adds a folder here; it does not touch the others.
- `deploy/environment/<env>/` — the **deployment environment** overlay (a *target*: local vs
  prod). `local` is fleshed out; `prod` is a stub.

> **Naming:** "deployment environment" here (local/prod target) is **not** the domain
> **Environment** — where a *session* runs (container | worktree | unset,
> [ADR-0007](../docs/adr/0007-roles-and-per-session-model-binding.md)). Different concepts;
> keep them distinct in writing.

## Packaging

Intentionally **config + runbook only** right now — no Docker/Nix images
([ADR-0009](../docs/adr/0009-deployment-topology-per-client-harness.md)). Images are an
additive follow-up once the governance surface is larger. Pi's own isolation options
(Gondolin micro-VM / Docker / OpenShell) are noted in `deploy/environment/prod/README.md`.

## Adding a new citizen

1. Build/locate the harness binary the client will spawn
   (`devenv tasks run harness:build` → `.cirius-harness/bin/harness`).
2. Add the client's harness **adapter** — for Pi this is the stdio extension
   (`.pi/extensions/harness`, ADR-0008); another client brings its own inbound adapter under
   `services/harness/internal/adapter/inbound/<client>` and whatever the client loads to spawn
   `harness serve`.
3. Create `deploy/<citizen>/README.md` (runbook) + any client config template.
4. Point it at an environment overlay (`deploy/environment/<env>/`).

## Secrets

Never commit real credentials. Only `*.example` templates live here; real `.env` files are
gitignored. Provider auth (API keys / client `/login`) is supplied per environment.
