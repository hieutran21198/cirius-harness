package domain

// WorktreeStatus reports whether a worktree is currently usable.
type WorktreeStatus string

const (
	// WorktreeActive marks a worktree backed by a live, checked-out git worktree.
	WorktreeActive WorktreeStatus = "active"
	// WorktreeStale marks a worktree whose backing directory is gone or detached.
	WorktreeStale WorktreeStatus = "stale"
)

// Valid reports whether s is a known status.
func (s WorktreeStatus) Valid() bool {
	switch s {
	case WorktreeActive, WorktreeStale:
		return true
	default:
		return false
	}
}
