// Package worktree is the worktree bounded context: the Worktree aggregate and
// the Worktrees repository port. A worktree is an isolated git working copy
// belonging to a project (see internal/domain/project); concurrent worktrees are
// the substrate for running work in parallel. Sessions run inside a worktree.
package worktree

import (
	"errors"
	"fmt"
)

// ErrInvalidWorktree is returned by Validate for a structurally invalid worktree.
var ErrInvalidWorktree = errors.New("worktree: invalid")

// Worktree is the aggregate root describing one isolated git working copy.
type Worktree struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// Path is the absolute filesystem path of the worktree, its unique business key.
	Path string
	// ProjectID is the id of the owning project.
	ProjectID string
	// Branch is the git branch checked out in this worktree.
	Branch string
	// Status reports whether the worktree is active or stale.
	Status Status
}

// Validate checks the worktree's invariants.
func (w Worktree) Validate() error {
	if w.Path == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidWorktree)
	}
	if w.ProjectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidWorktree)
	}
	if w.Branch == "" {
		return fmt.Errorf("%w: branch is required", ErrInvalidWorktree)
	}
	if !w.Status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidWorktree, w.Status)
	}
	return nil
}
