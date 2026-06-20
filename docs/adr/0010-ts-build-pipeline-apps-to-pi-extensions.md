# 0010. TypeScript build pipeline: `apps/` → `.pi/extensions/`

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

[ADR-0008](0008-pi-client-integration-stdio.md) shipped the Pi extension as a single
hand-edited file committed at `.pi/extensions/harness/index.ts`, tracked via a `.gitignore`
exception. That makes `.pi/` — a Pi **runtime/config** directory — also the *source of truth*
for code, which is backwards: `.pi/` otherwise holds gitignored session/auth/package state.

We want the extension to be a first-class project under `apps/`, consistent with the
workspace's nx + pnpm TS tooling — tooling `devenv.nix` already assumes (`workspace:fmt`/
`lint`/`test` call `nx format:write` and `nx run-many`) but which was never actually stood up
(root `node_modules` had only `typescript`).

Constraints that shape the pipeline, verified against the installed Pi 0.79.6 loader:

- **Pi dictates the load path.** Pi only discovers extensions under `.pi/extensions/<name>/`,
  resolving `index.ts` then `index.js` (and following symlinks). Source placed anywhere else
  must still *arrive* at `.pi/extensions/`.
- **Pi loads via jiti and injects an allowlist** of runtime modules
  (`@earendil-works/pi-coding-agent`, `…/pi-agent-core`, `…/pi-tui`, `…/pi-ai`, `typebox`).
  Those must stay **external** in any bundle; arbitrary npm deps are not injected.
- The extension is **not** a shared library (imported by nothing) and **not** a standalone
  app (no entrypoint — Pi hosts it). It is the **TypeScript half of the `pilink` inbound
  adapter** ([ADR-0004](0004-ports-and-adapters-topology.md)); its Go half is
  `services/harness/internal/adapter/inbound/pilink`.

## Decision

The extension **source lives in `apps/pi-harness-extension/`** (an nx application) and is
**built into `.pi/extensions/harness/` as generated, gitignored output.**

- Build target: `@nx/esbuild:esbuild` — `platform: node`, `format: esm`, `bundle: true`,
  `outputPath: .pi/extensions/harness` (emits `index.js`), with Pi's injected packages listed
  as `external`.
- Entrypoint: `devenv tasks run pi-extension:build` (mirrors `harness:build`), the single
  task that runs `nx build pi-harness-extension`.
- `.pi/extensions/` is **no longer tracked** — only `.pi/README.md` is. This refines
  ADR-0008's note that `.pi/extensions/` was tracked.
- Lint/format for `apps/` TS uses **Biome**, provided by `devenv.nix` (its npm binary is
  dynamically linked and will not run on NixOS; esbuild, a static Go binary, stays an npm dep).

esbuild is chosen over a plain copy so that an off-allowlist dependency later "just works"
with no pipeline change, even though today's shim needs only Node builtins + a type-only Pi
import.

## Consequences

- **Positive**: `.pi/` returns to being purely generated/runtime; code lives where the
  workspace conventions expect it, with `nx` build caching, typecheck, and lint. Completes the
  TS toolchain the devenv tasks already referenced.
- **Negative**: a build step now precedes `pi -a` (the runbooks call it out); standing up nx +
  esbuild + Biome adds toolchain surface for one small file. NixOS requires `direnv reload`
  after pulling so the devenv-provided Biome is on `PATH` for `nx lint`.
- **Neutral**: `apps/` is the least-wrong nx bucket for a host-loaded adapter; if a second TS
  citizen appears the pattern generalizes (one app per citizen, each built into its client's
  load path).

## Alternatives considered

- **Keep source in `.pi/extensions/` (status quo)** — zero build, but conflates runtime dir
  with source of truth and sits outside the TS toolchain. Rejected once we wanted nx to manage
  it.
- **`packages/ts/libs/pi-harness-extension`** — nx's `libs/` bucket is for shared, imported
  code; this is imported by nothing. Wrong category.
- **Committed symlink `.pi/extensions/harness` → `apps/…/src`** — zero build and never stale,
  but symlinks are fragile on Windows and a committed symlink inside `.pi/` is surprising;
  also forecloses bundling off-allowlist deps later.
- **Plain copy instead of esbuild** — sufficient today (Pi loads `.ts`/`.js` as-is), but
  would need replacing the moment a third-party dep appears. esbuild costs nothing extra now.

## References

- [ADR-0008](0008-pi-client-integration-stdio.md) (the connection this builds on),
  [ADR-0004](0004-ports-and-adapters-topology.md) (inbound adapter placement),
  [ADR-0001](0001-harness-layout.md) (layout: `apps/`, nx `workspaceLayout`).
- Pi 0.79.6 loader (installed with the binary): `dist/core/package-manager.js`
  (`resolveExtensionEntries`: `index.ts` → `index.js`, `pi.extensions` manifest field),
  `dist/core/extensions/loader.js` (jiti + injected module allowlist).
