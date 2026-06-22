package domain

import "fmt"

// MemberID is a Member's surrogate identity (a UUID v7). A named string type so a member
// id can't be silently passed where another aggregate's id is expected.
type MemberID string

// Member records one agent's participation in a session — the agent↔session join
// (session_agents). It carries the model the agent ran with this session, so the
// run is reproducible regardless of later edits to the agent or model catalog. It
// references the participating agent and model by their owning aggregates' ids.
type Member struct {
	id      MemberID
	agentID AgentID
	modelID ModelID
}

// NewMember assembles a fresh session member from the participating agent and the
// model it ran with (empty for a model-less agent), minting its own identity (UUID
// v7), and validates it.
func NewMember(agentID AgentID, modelID ModelID) (Member, error) {
	m := Member{id: newID[MemberID](), agentID: agentID, modelID: modelID}
	return m, m.Validate()
}

// RehydrateMember reconstitutes a Member from its persisted state and validates it.
func RehydrateMember(id MemberID, agentID AgentID, modelID ModelID) (Member, error) {
	m := Member{id: id, agentID: agentID, modelID: modelID}
	return m, m.Validate()
}

// Validate checks the member's invariants.
func (m Member) Validate() error {
	if m.id == "" {
		return fmt.Errorf("%w: member id is required", ErrInvalidSession)
	}
	if m.agentID == "" {
		return fmt.Errorf("%w: member agent id is required", ErrInvalidSession)
	}
	return nil
}

// MemberSnapshot is the persistence grouped view of a Member.
type MemberSnapshot struct {
	ID      MemberID
	AgentID AgentID
	ModelID ModelID
}

// Snapshot returns the member's persistence view.
func (m Member) Snapshot() MemberSnapshot {
	return MemberSnapshot{ID: m.id, AgentID: m.agentID, ModelID: m.modelID}
}
