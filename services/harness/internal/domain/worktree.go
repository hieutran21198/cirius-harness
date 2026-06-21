package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidWorktree is returned by Validate for a structurally invalid worktree.
var ErrInvalidWorktree = errors.New("worktree: invalid")

// WorktreeID is a Worktree's surrogate identity (a UUID v7). A named string type so a
// worktree id can't be silently passed where another aggregate's id is expected.
type WorktreeID string

// Worktree is the aggregate root describing one isolated git working copy
// belonging to a project; concurrent worktrees are the substrate for running work
// in parallel. Sessions run inside a worktree.
type Worktree struct {
	id        WorktreeID
	path      string
	projectID ProjectID
	branch    string
	status    WorktreeStatus
}

// NewWorktree assembles a fresh worktree from its attributes, minting its own
// identity (UUID v7), active by default, and validates it.
func NewWorktree(path string, projectID ProjectID, branch string) (Worktree, error) {
	w := Worktree{id: newID[WorktreeID](), path: path, projectID: projectID, branch: branch, status: WorktreeActive}
	return w, w.Validate()
}

// RehydrateWorktree reconstitutes a Worktree from its persisted state (no creation
// defaults) and validates structural integrity.
func RehydrateWorktree(id WorktreeID, path string, projectID ProjectID, branch string, status WorktreeStatus) (Worktree, error) {
	w := Worktree{id: id, path: path, projectID: projectID, branch: branch, status: status}
	return w, w.Validate()
}

// Validate checks the worktree's invariants.
func (w Worktree) Validate() error {
	if w.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidWorktree)
	}
	if w.path == "" {
		return fmt.Errorf("%w: path is required", ErrInvalidWorktree)
	}
	if w.projectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidWorktree)
	}
	if w.branch == "" {
		return fmt.Errorf("%w: branch is required", ErrInvalidWorktree)
	}
	if !w.status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidWorktree, w.status)
	}
	return nil
}
