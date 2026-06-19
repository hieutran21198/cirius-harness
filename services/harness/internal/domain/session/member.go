package session

import (
	"fmt"
	"time"
)

// Member records one agent's participation in a session — the live join between
// the agent and session domains. Agent is the agent's natural key (its name).
type Member struct {
	// Agent is the name of the participating agent (its natural key).
	Agent string
	// Role is an optional per-session designation (e.g. "lead"); may be empty.
	Role string
	// Active reports whether the agent is still part of the session.
	Active bool
	// JoinedAt is when the agent joined the session.
	JoinedAt time.Time
}

// Validate checks the member's invariants.
func (m Member) Validate() error {
	if m.Agent == "" {
		return fmt.Errorf("%w: member agent is required", ErrInvalidSession)
	}
	return nil
}
