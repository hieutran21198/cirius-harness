# environment/prod — deferred

Production deployment is **not defined yet**
([ADR-0009](../../../docs/adr/0009-deployment-topology-per-client-harness.md): packaging is
config + runbook, local first). This stub records what a real `prod` must address so it is not
forgotten.

> "Deployment environment" (this folder) ≠ the domain **Environment** (where a *session*
> runs). See [`../../AGENTS.md`](../../AGENTS.md).

## What prod must address when it becomes real

- **Secret sourcing** — provider credentials from a secret manager / injected env, never
  committed and never baked into an artifact. No `harness.env` in the repo.
- **Model catalog** — a restricted, explicitly-enabled set of providers/models (not "whatever
  the user is authenticated for").
- **Persistence** — a durable harness DB location with backups, separate from a repo working
  tree; define retention.
- **Isolation hardening** — run the client in a sandbox rather than on a bare host. Pi offers
  Gondolin micro-VM, plain Docker, and OpenShell (see Pi's `docs/containerization.md`); this
  is also where **container/Nix packaging** (deferred today) would land.
- **Motherboard** — once the central service exists (Module 2,
  [ADR-0001](../../../docs/adr/0001-harness-layout.md)), define how each child-harness connects
  up to it for the cross-client view.

Until then, use [`../local`](../local/README.md).
