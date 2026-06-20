# `.cirius-harness/` — workspace config

This directory models how cirius-harness reads a user's workspace. The schema here is the
**AI-agents orchestration** goal made concrete — the declarative agent team the control
plane runs. See root [`AGENTS.md`](../AGENTS.md) for the full mission (harness AI-coding,
orchestration, concurrency).

| File             | Role                                        | Owns                                               |
| ---------------- | ------------------------------------------- | -------------------------------------------------- |
| `00-system.yaml` | **Default schema** (shipped by the harness) | the default agent team + global `fallbacks` policy |

`00-system.yaml` is the team you get **out of the box** when your workspace defines
nothing. A user override layer that merges on top of it is **deferred** (see below).

## Archetypes

Each agent has an **archetype** — its purpose-level operating style — backed by the model
families that fit it (the `00-system.yaml` header comments carry the full reasoning):

- **communicator** — instruction-following, mechanics-driven (detailed checklists, nested
  workflows). Model families: Claude ▸ Kimi ▸ GLM.
- **principle-driven** — autonomous, goal-first (state the goal, it finds the mechanics).
  Model families: GPT ▸ Deepseek.
- **utility-runner** — speed over intelligence; cheap, high-volume work. Model family:
  MiniMax.

## The default team

One verb per agent. Claude bookends (design + critique); GPT does the autonomous
middle (build + research); MiniMax scans cheap. Only `implementer` may edit.

| Agent         | Verb     | Archetype        | model → fallback                           |
| ------------- | -------- | ---------------- | ------------------------------------------ |
| `council`     | route    | communicator     | `claude-opus-4-7` → `gpt-5.4`              |
| `planner`     | design   | communicator     | `claude-opus-4-7` → `kimi-k2.7`            |
| `implementer` | build    | principle-driven | `gpt-5.5` → `claude-opus-4-8`              |
| `researcher`  | gather   | principle-driven | `gpt-5.4` → `gemini-3-pro`                 |
| `explorer`    | scan     | utility-runner   | `minimax-m3` → `deepseek-v3`               |
| `reviewer`    | critique | communicator     | `claude-sonnet-4-6` → `gpt-5.4`            |
| `scribe`      | retain   | communicator     | `claude-sonnet-4-6` → `kimi-k2.7`          |
| `prayer`      | pray     | none 🙏          | _(model-less — burns incense, not tokens)_ |

**Flow:** council routes → planner designs → implementer builds →
researcher / explorer feed them → reviewer critiques → scribe retains.
_(…and `prayer` burns incense over the whole thing — a dev with incense does the job
better than nothing.)_

`scribe` is the team's **memory**: it receives technical debt (from `reviewer` /
`implementer`) and end-of-task summaries (from `council`), distills them, and persists
them as knowledge that later runs pull from. It is distinct from the **audit trail**
(raw events = _what happened_); `scribe` owns the distilled lessons (= _what we
learned_). It is the only non-`implementer` agent that writes — and only to the
knowledge store.

## Agent shape

Each agent is a block with four keys:

```yaml
agents:
  reviewer:
    model: "anthropic/claude-sonnet-4-6" # primary model (provider/model-id)
    permissions: # least-privilege; allow | ask | deny
      read: allow
      edit: deny
      bash: ask
      webfetch: deny
      websearch: deny
    tools: [read, grep, glob, list] # tools the agent may use
    fallbacks: ["openai/gpt-5.4"] # ordered; tried after the primary
```

Fallback is a **harness-level** concept: the harness picks a single concrete model
(advancing through `fallbacks` per the global `fallbacks.on` triggers — `error`,
`ratelimit`, `budget`, `quota`) and hands that one model to the client.

## Deferred (not in this schema yet)

- **Routing / weighting** — `council`'s job of classifying a task and assigning it to
  an agent (it routes; it does not author plans — that's `planner`).
- **`variant` / effort tiers** (e.g. claude `high` / `xhigh` / `max`) — chosen by the
  weighting logic later.
- **User override layer** — a workspace file that merges on top of `00-system.yaml`,
  plus a shared `defaults` block agents inherit instead of repeating permissions/tools.
- **Path-scoped permissions** — `edit: allow` is currently all-or-nothing. `scribe`
  needs write access to the knowledge store _only_ (never source), which the coarse
  `allow | ask | deny` model can't yet express.
