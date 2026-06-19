# 0003. Authorization via Casbin ABAC

- **Status**: Accepted
- **Date**: 2026-06-19
- **Deciders**: hieu
- **Supersedes**: -
- **Superseded by**: -

> **Refined by [ADR-0004](0004-ports-and-adapters-topology.md):** the `Authorizer` port now
> lives in `internal/port/outbound` and its Casbin impl in
> `internal/adapter/outbound/casbinauthz`. The ABAC model and decision semantics are unchanged.

## Context

Each agent has per-capability permissions (read, edit, bash, webfetch, websearch) that are
not simply on/off: the harness needs a third state, **ask** (permit only after asking the
user), alongside **allow** and **deny** — as already expressed in
`.cirius-harness/00-system.yaml`. The first instinct is permission columns on the `agents`
table, but that makes permissions un-queryable as a set, hard-codes the five capabilities,
and offers no path to resource/path scoping (e.g. `scribe` may edit the knowledge store but
not source).

Authorization is therefore a **separate bounded concern** from the agent aggregate, and
needs a model that supports a three-valued decision and future path scoping.

## Decision

We will model authorization with **Casbin (ABAC)**, persisting policy in the database via
the official **gorm-adapter** sharing the same `*gorm.DB` as the rest of the service.
Permissions are **not** columns on `agents`.

- The **agent is the principal**: the Casbin subject is the agent's name.
- Policy is stored in the **`casbin_rule`** table, created/managed by the gorm-adapter
  (`casbinx.NewEnforcer(db, modelText)` wires the enforcer over the shared connection).
- The three-valued decision **allow | ask | deny** is carried by a per-policy **`dec`**
  field and read via **`EnforceEx`** on the matched rule — the binary `Enforce` effect is
  bypassed. **No matching rule ⇒ deny** (default-deny).
- Casbin model:

  ```
  [request_definition] r = sub, obj, act
  [policy_definition]   p = sub, obj, act, dec
  [policy_effect]       e = some(where (p.eft == allow))
  [matchers]            m = r.sub == p.sub && keyMatch(r.obj, p.obj) && r.act == p.act
  ```

- The domain exposes an `authz.Authorizer` **port** (`Decide(ctx, principal, resource,
  action) → Decision`) with `Decision` ∈ {allow, ask, deny}; the Casbin implementation
  lives in `internal/adapters/casbinauthz`.

## Consequences

- **Positive**: permissions are queryable, auditable DB rows (the audit/observability
  pillar), independent of the agent table's shape.
- **Positive**: `keyMatch` on the object makes **path/resource scoping** a future policy
  change, not a schema change (unblocks `scribe`'s knowledge-store-only edit rights).
- **Negative**: the gorm-adapter pulls extra SQL drivers as indirect deps (noted in
  [ADR-0002](0002-persistence-and-migrations.md)).
- **Neutral**: `casbin_rule` is owned by the adapter (AutoMigrate), so it is intentionally
  not part of the goose migrations.

## Alternatives considered

- **Five permission columns on `agents`** — simplest. Rejected: not queryable as a policy
  set, hard-codes the capabilities, and has no path to resource scoping.
- **Casbin RBAC (roles)** — role→permission mapping. Rejected: the need is
  resource/path matching per principal, which ABAC + `keyMatch` fits directly.
- **OPA / Rego** — a full policy engine. Rejected: heavier runtime and language for what a
  Casbin model in-process already covers.

## References

- [ADR-0002](0002-persistence-and-migrations.md) — the shared `*gorm.DB` the adapter reuses.
- `packages/go/casbinx` — enforcer bootstrap; `services/harness/internal/adapters/casbinauthz` — the model + `Decide`.
- `docs/glossary/README.md` — Decision, Action, Authorizer, Principal, Policy.
- Versions: `casbin/casbin/v3` v3.8.1, `casbin/gorm-adapter/v3` v3.41.0.
