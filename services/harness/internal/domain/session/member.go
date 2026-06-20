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

// NewMember assembles a session member from an app-minted id, the participating
// agent, and the model it ran with (empty for a model-less agent), and validates
// it. The id is supplied by the application/adapter.
func NewMember(id, agentID, modelID string) (Member, error) {
	m := Member{ID: id, AgentID: agentID, ModelID: modelID}
	return m, m.Validate()
}

// Validate checks the member's invariants.
func (m Member) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("%w: member id is required", ErrInvalidSession)
	}
	if m.AgentID == "" {
		return fmt.Errorf("%w: member agent id is required", ErrInvalidSession)
	}
	return nil
}
