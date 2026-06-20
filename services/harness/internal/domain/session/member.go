package session

import "fmt"

// Member records one agent's participation in a session — the agent↔session join
// (session_agents). It carries the model the agent ran with this session, so the
// run is reproducible regardless of later edits to the agent or model catalog.
type Member struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// AgentID is the id of the participating agent.
	AgentID string
	// ModelID is the id of the model the agent ran with; empty for a model-less
	// agent (prayer).
	ModelID string
}

// Validate checks the member's invariants.
func (m Member) Validate() error {
	if m.AgentID == "" {
		return fmt.Errorf("%w: member agent id is required", ErrInvalidSession)
	}
	return nil
}
