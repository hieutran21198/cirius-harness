package domain

// SessionStatus tracks a session through its lifecycle.
type SessionStatus string

const (
	// SessionPending marks a session created but not yet started.
	SessionPending SessionStatus = "pending"
	// SessionRunning marks a session actively executing.
	SessionRunning SessionStatus = "running"
	// SessionCompleted marks a session that finished successfully.
	SessionCompleted SessionStatus = "completed"
	// SessionFailed marks a session that ended in error.
	SessionFailed SessionStatus = "failed"
	// SessionCancelled marks a session stopped by the user before completion.
	SessionCancelled SessionStatus = "cancelled"
)

// Valid reports whether s is a known status.
func (s SessionStatus) Valid() bool {
	switch s {
	case SessionPending, SessionRunning, SessionCompleted, SessionFailed, SessionCancelled:
		return true
	default:
		return false
	}
}

// Terminal reports whether s is an end state (no further transitions).
func (s SessionStatus) Terminal() bool {
	switch s {
	case SessionCompleted, SessionFailed, SessionCancelled:
		return true
	default:
		return false
	}
}
