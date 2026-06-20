package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/worktree"
)

// Worktrees is the aggregate repository for the worktree domain. The plural name
// denotes the collection of Worktree aggregates.
type Worktrees interface {
	// Get returns the worktree with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (worktree.Worktree, error)
	// List returns all worktrees.
	List(ctx context.Context) ([]worktree.Worktree, error)
	// ListByProject returns the worktrees owned by the project with the given id.
	ListByProject(ctx context.Context, projectID string) ([]worktree.Worktree, error)
	// Save inserts or updates the worktree, keyed by ID.
	Save(ctx context.Context, w worktree.Worktree) error
	// Delete removes the worktree with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
