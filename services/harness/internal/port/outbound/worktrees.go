package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/worktree"
)

// Worktrees is the aggregate repository for the worktree domain. The plural name
// denotes the collection of Worktree aggregates.
type Worktrees interface {
	// Get returns the worktree with the given path, or ErrNotFound.
	Get(ctx context.Context, path string) (worktree.Worktree, error)
	// List returns all worktrees.
	List(ctx context.Context) ([]worktree.Worktree, error)
	// ListByProject returns the worktrees owned by the named project.
	ListByProject(ctx context.Context, project string) ([]worktree.Worktree, error)
	// Save inserts or updates the worktree, keyed by Path.
	Save(ctx context.Context, w worktree.Worktree) error
	// Delete removes the worktree with the given path, or returns ErrNotFound.
	Delete(ctx context.Context, path string) error
}
