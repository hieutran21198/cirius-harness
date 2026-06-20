# environment/local

The **local** deployment target: a developer running a citizen on their own machine. This is
the only fully-defined environment today ([ADR-0009](../../../docs/adr/0009-deployment-topology-per-client-harness.md));
`../prod` is a stub.

> "Deployment environment" (this folder) ≠ the domain **Environment** (where a *session*
> runs). See [`../../AGENTS.md`](../../AGENTS.md).

## Shape of "local"

| Concern | Local |
| --- | --- |
| Where it runs | On the host, no isolation. |
| Harness binary | `.cirius-harness/bin/harness` (`devenv tasks run harness:build`). |
| Harness DB | The repo's `.cirius-harness/state/harness.sqlite` (gitignored). |
| Provider auth | The developer's shell env vars and/or the client's `/login`. |
| Model catalog | Full — whatever the developer is authenticated for. |
| Logging | Verbose; harness logs to stderr. |

## Setup

1. Build the harness: `devenv tasks run harness:build`.
2. Provide provider auth — either authenticate inside the client (Pi: `/login`) or export the
   provider keys your client needs. Copy the template and fill it in **locally** (it is
   gitignored):

   ```bash
   cp deploy/environment/local/harness.env.example deploy/environment/local/harness.env
   # edit harness.env, then source it in your shell as needed
   ```

3. Run the citizen — for Pi see [`../../pi/README.md`](../../pi/README.md).

## Notes on the env template

`harness.env.example` is **honest about what is wired today**:

- The **provider API-key variables** (e.g. `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) are the
  real, live config surface — they are read by Pi / your client, not by the harness.
- The `HARNESS_*` entries are **reserved / forward-looking**. The harness does not read env
  vars yet (the `serve` DB path is a CLI argument). They document the intended config contract
  so the file does not lie about behavior that exists.
