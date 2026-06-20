# 0009. Deployment topology: one child-harness per client, motherboard later

- **Status**: Accepted
- **Date**: 2026-06-20
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

## Context

The harness governs an AI coding **client** ([ADR-0001](0001-harness-layout.md)). The first
client is **Pi**, wired up in [ADR-0008](0008-pi-client-integration-stdio.md): entering Pi
launches `harness serve` as a child process over stdio. We now need a deployment model —
how these pieces are arranged and operated — before adding more clients or a central service.

Two forces shape it:

- The control loop is inherently **per session**: ADR-0008 already spawns one harness
  process per Pi session. There is no shared runtime state to coordinate yet.
- A central service to **view and control concurrent work across modules** is anticipated but
  deliberately unbuilt (ADR-0001 "Module 2", reserved).

## Decision

A **deployment is a client paired 1:1 with its own child-harness process.** One client ⇄ one
harness. We call a deployed, harness-governed client a **citizen**; **Pi is the first
citizen**.

A central **motherboard** service is **deferred** — it is Module 2. When it exists,
child-harness processes will connect *up* to it for the cross-client/concurrency view; the
1:1 client⇄harness pairing below it stays unchanged.

Packaging is **configuration + runbook** for now (no container/Nix images): `deploy/<citizen>/`
holds a client's runbook and config, `deploy/environment/<env>/` holds environment overlays
(local now; prod is a documented stub).

## Consequences

- **Positive**: dead-simple isolation — each citizen's harness lives and dies with its client
  process, so there is nothing shared to lock, reap, or version across clients. Matches
  ADR-0008's per-session spawn exactly; no daemon to operate.
- **Positive**: adding a second citizen (e.g. opencode) is additive — a new `deploy/<citizen>/`
  plus that client's harness adapter; it does not touch Pi's deployment.
- **Negative**: **no cross-client/cross-session view** until the motherboard lands — each
  child-harness is blind to the others. This is the gap Module 2 fills.
- **Negative**: environment config is **duplicated per citizen/environment** (no central
  config service yet); the overlay folders keep the duplication visible and small.
- **Neutral**: "config + runbook" means deployment artifacts are docs and templates today;
  images (Docker/Nix) are a later, additive packaging decision.

## Alternatives considered

- **One shared harness daemon many clients connect to** — a central process from day one.
  Rejected as premature: it is effectively the motherboard (Module 2), and building it now
  would over-design before the concurrency requirements are understood.
- **Harness embedded in-process inside the client** — no separate process. Rejected: it
  couples the harness's lifecycle and language (Go) to each client's runtime and breaks the
  clean stdio boundary established in ADR-0008.
- **Ship container/Nix images now** — Rejected for this round: the governance surface is
  still thin, so a runbook + config templates deliver the deployment story faster and
  packaging stays an additive follow-up.

## References

- [ADR-0001](0001-harness-layout.md) — layout and the reserved Module 2 (the motherboard).
- [ADR-0008](0008-pi-client-integration-stdio.md) — the per-session child-harness over stdio.
- `deploy/AGENTS.md` — the deployment knowledge base this decision anchors.
