// Package domain is the harness's pure domain model: every aggregate root
// (Model, Agent, Project, Session, Worktree, Container, Tool), the value objects
// and typed-string enums they are built from, and the per-aggregate driven ports
// (e.g. ModelWriter). It is one package on purpose — cross-aggregate logic (a
// session referencing an agent, a model, and a project) collaborates without
// import ceremony (ADR-0014, superseding the per-bounded-context packages of
// ADR-0013).
//
// Encapsulation is at the type boundary, not the package boundary: aggregates
// hold unexported fields and expose meaning through methods. State is created
// with NewXxx (fresh, in the application) or RehydrateXxx (reconstituted from
// storage, in the repository), and leaves the domain only through domain-owned
// grouped views with a clear purpose (e.g. Model.Snapshot for persistence). The id
// format is a domain identity policy too: each NewXxx mints its own id via newID()
// (see id.go); only RehydrateXxx takes a stored id. Identities are typed per aggregate
// (ModelID, AgentID, …) rather than bare strings, so one aggregate's id can't be passed
// where another's is expected. The package carries no GORM tags
// and no infrastructure imports — it depends only on the stdlib plus
// github.com/google/uuid for identity generation.
package domain
