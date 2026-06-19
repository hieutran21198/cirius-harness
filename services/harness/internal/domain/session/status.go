package session

// Status tracks a session through its lifecycle.
type Status string

const (
	// StatusPending marks a session created but not yet started.
	StatusPending Status = "pending"
	// StatusRunning marks a session actively executing.
	StatusRunning Status = "running"
	// StatusCompleted marks a session that finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed marks a session that ended in error.
	StatusFailed Status = "failed"
	// StatusCancelled marks a session stopped by the user before completion.
	StatusCancelled Status = "cancelled"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// Terminal reports whether s is an end state (no further transitions).
func (s Status) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
