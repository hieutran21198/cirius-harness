# `.pi/` — Pi coding client integration

This workspace is governed by **cirius-harness** through the **Pi** coding client
(`@earendil-works/pi-coding-agent`, pi.dev). The harness is a control plane *over* Pi: when
you enter Pi here, a project-local extension launches the harness and connects to it.

Only this README is tracked. Everything else under `.pi/` is generated or runtime state: the
`extensions/harness/` extension is **built** from `apps/pi-harness-extension/` (see below), and
Pi's own sessions, settings, installed packages, and auth are gitignored.

## What's here

| Path | Role |
| --- | --- |
| `extensions/harness/` | **Build output** (gitignored) — the compiled Pi extension that launches `harness serve` on session start and connects over stdio ([ADR-0008](../docs/adr/0008-pi-client-integration-stdio.md)). Source lives in [`apps/pi-harness-extension/`](../apps/pi-harness-extension/); build with `devenv tasks run pi-extension:build`. |

The extension spawns one harness process per Pi session, performs a hello/ready handshake
(newline-delimited JSON over stdio), shows liveness in the footer, and kills the process on
exit. This first slice is **connect-only** — it proves the channel; model handoff,
permission gating, and tool grants come later over the same channel.

## One-time setup

1. **Build the harness binary** (the extension runs it from `.cirius-harness/bin/harness`):

   ```bash
   devenv tasks run harness:build
   ```

2. **Build the Pi extension** (compiles `apps/pi-harness-extension/` into
   `.pi/extensions/harness/index.js`, where Pi loads it):

   ```bash
   devenv tasks run pi-extension:build
   ```

3. **Trust the project** so Pi loads the project-local extension (project-local extensions
   load only after trust is resolved). Either start Pi with `-a`:

   ```bash
   pi -a
   ```

   or run `pi` and use `/trust` once (then restart).

On entry you should see a footer status like `● harness · schema v20260619091805` and a
"harness connected" notification. If the binary is missing, Pi stays fully usable and the
extension notifies you to run the build task.

## Troubleshooting

- **"harness: binary missing"** — run `devenv tasks run harness:build`, then `/reload`.
- **Extension never loads** (not in Pi's startup header) — build it:
  `devenv tasks run pi-extension:build` (Pi loads `.pi/extensions/harness/index.js`).
- **No status appears** — confirm the project is trusted (`/trust`) and the extension is
  loaded (it shows in Pi's startup header).
- **Inspect the channel manually** — `harness serve` speaks NDJSON on stdio:

  ```bash
  printf '{"type":"hello","cwd":"%s"}\n' "$PWD" | .cirius-harness/bin/harness serve
  # → {"type":"ready","schemaVersion":...,"dbPath":...,"pid":...}
  ```
