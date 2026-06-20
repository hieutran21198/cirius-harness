# deploy/pi — Pi, the first citizen

Runbook for operating **Pi** (`@earendil-works/pi-coding-agent`, pi.dev) governed by the
harness. Pi is the first **citizen** — a coding client paired 1:1 with its own child-harness
([ADR-0009](../../docs/adr/0009-deployment-topology-per-client-harness.md)).

For the deployment model overview see [`../AGENTS.md`](../AGENTS.md). For the *in-repo dev
wiring* of the tracked extension see [`/.pi/README.md`](../../.pi/README.md) — this file is the
**operate** view and does not duplicate it.

## What runs

```
pi  ──spawns on session_start──▶  harness serve   (child process, stdio NDJSON)
        │  ◀──── ready (schema vN) ───────┘
        └─ status line: ● harness · schema vN
```

When you enter Pi in a trusted project, the extension `.pi/extensions/harness` (built from
`apps/pi-harness-extension/`) launches `.cirius-harness/bin/harness serve`, performs a
hello/ready handshake over stdio
([ADR-0008](../../docs/adr/0008-pi-client-integration-stdio.md)), and shows liveness in the
footer. The harness process exits with the Pi session. This first slice is **connect-only** —
no model/permission/tool governance yet.

## Prerequisites

- Pi installed and on `PATH` (`pi --version`).
- An environment overlay selected — for development that is
  [`../environment/local`](../environment/local/README.md).
- The harness binary and the Pi extension built:

  ```bash
  devenv tasks run harness:build        # → .cirius-harness/bin/harness
  devenv tasks run pi-extension:build   # → .pi/extensions/harness/index.js
  ```

## Run

```bash
# from the project root
pi -a        # -a trusts the project so the project-local extension loads
```

Or run `pi` and `/trust` once, then restart. On entry you should see
`● harness · schema v…` in the footer and a "harness connected" notification.

Per-session controls: `/reload` re-runs the handshake; `/new` and `/resume` restart the
child-harness; `/quit` tears it down (no orphan process).

## Config

- **Pi global settings** — `~/.pi/agent/settings.json` (provider, model, thinking level).
  See [`settings.example.json`](settings.example.json) for a default-team-aligned starting
  point. Provider **auth** is environment-specific (see the environment overlay), never
  committed.
- **Harness DB** — `harness serve` opens `.cirius-harness/state/harness.sqlite` (its cwd is
  the project root). The DB path is currently a CLI argument, not an env var.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `harness: binary missing` | `devenv tasks run harness:build`, then `/reload`. |
| No harness status appears | Confirm the project is trusted (`/trust`); the extension shows in Pi's startup header. |
| Verify the channel by hand | `printf '{"type":"hello","cwd":"%s"}\n' "$PWD" \| .cirius-harness/bin/harness serve` → expect one `{"type":"ready",...}` line. |
| Stale harness process | None should outlive Pi; check with `pgrep -x harness`. |
