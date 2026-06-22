package domain

import "context"

// SessionWriter persists sessions and their members (session_agents). It is a
// domain-owned driven port obtained from a UnitOfWork and implemented by the infra
// adapter (ADR-0013).
type SessionWriter interface {
	// Save inserts the session row. It is idempotent on the session id (a re-sent
	// hello for the same session is a no-op), so it does not clobber an existing run.
	Save(ctx context.Context, s Session) error
	// AddMember records one agent's participation in the session (a session_agents
	// row). It is idempotent on (session, agent): recording the same agent twice in a
	// session is a no-op.
	AddMember(ctx context.Context, sessionID SessionID, m Member) error
}

// ProjectWriter persists projects. It is a domain-owned driven port obtained from a
// UnitOfWork and implemented by the infra adapter (ADR-0013).
type ProjectWriter interface {
	// EnsureByRoot returns the id of the project at rootPath, creating it (with name)
	// if absent. rootPath is the project's unique business key.
	EnsureByRoot(ctx context.Context, rootPath, name string) (ProjectID, error)
}
