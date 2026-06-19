package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/session"
)

// Sessions is the aggregate repository for the session domain. The plural name
// denotes the collection of Session aggregates.
type Sessions interface {
	// Get returns the session with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (session.Session, error)
	// List returns all sessions.
	List(ctx context.Context) ([]session.Session, error)
	// ListByWorktree returns the sessions that ran in the worktree at path.
	ListByWorktree(ctx context.Context, path string) ([]session.Session, error)
	// Save inserts or updates the session and its members as one aggregate.
	Save(ctx context.Context, s session.Session) error
	// Delete removes the session with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
