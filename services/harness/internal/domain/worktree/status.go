package worktree

// Status reports whether a worktree is currently usable.
type Status string

const (
	// StatusActive marks a worktree backed by a live, checked-out git worktree.
	StatusActive Status = "active"
	// StatusStale marks a worktree whose backing directory is gone or detached.
	StatusStale Status = "stale"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusStale:
		return true
	default:
		return false
	}
}
