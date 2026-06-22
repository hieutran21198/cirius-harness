package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidAgent is returned by Validate for a structurally invalid agent.
var ErrInvalidAgent = errors.New("agent: invalid")

// AgentID is an Agent's surrogate identity (a UUID v7). A named string type so an agent
// id can't be silently passed where another aggregate's id is expected.
type AgentID string

// Agent is the aggregate root describing one member of the harness team — a pure
// identity/role. It carries no model (the model is bound per session) and no
// fallbacks; its tools are catalog grants in toolIDs. The agent is also the
// authorization principal: its name is the Casbin subject. Permissions are NOT
// modelled here — authorization is a separate concern backed by Casbin (see
// Decision/Action and internal/infra/casbin).
//
// An agent's persona (its harness-owned system prompt) is NOT stored here either: it
// is harness-owned code, a domain constant resolved by name via PersonaFor, not
// persisted on the aggregate (ADR-0016, see persona.go).
type Agent struct {
	id             AgentID
	name           string
	responsibility string
	archetype      Archetype
	description    string
	source         Source
	enabled        bool
	toolIDs        []ToolID
}

// NewAgent assembles a fresh agent (a team role) from its attributes, minting its own
// identity (UUID v7), enabled by default, and validates it.
func NewAgent(name string, archetype Archetype, responsibility, description string, source Source, toolIDs []ToolID) (Agent, error) {
	a := Agent{
		id:             newID[AgentID](),
		name:           name,
		responsibility: responsibility,
		archetype:      archetype,
		description:    description,
		source:         source,
		enabled:        true,
		toolIDs:        toolIDs,
	}
	return a, a.Validate()
}

// RehydrateAgent reconstitutes an Agent from its persisted state (no creation
// defaults) and validates structural integrity.
func RehydrateAgent(id AgentID, name string, archetype Archetype, responsibility, description string, source Source, enabled bool, toolIDs []ToolID) (Agent, error) {
	a := Agent{
		id:             id,
		name:           name,
		responsibility: responsibility,
		archetype:      archetype,
		description:    description,
		source:         source,
		enabled:        enabled,
		toolIDs:        toolIDs,
	}
	return a, a.Validate()
}

// Validate checks the agent's invariants.
func (a Agent) Validate() error {
	if a.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidAgent)
	}
	if a.name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidAgent)
	}
	if !a.source.Valid() {
		return fmt.Errorf("%w: unknown source %q", ErrInvalidAgent, a.source)
	}
	if !a.archetype.Valid() {
		return fmt.Errorf("%w: unknown archetype %q", ErrInvalidAgent, a.archetype)
	}
	return nil
}

// AgentSnapshot is the persistence/read grouped view of an Agent: its whole state,
// grouped for storage and reconstitution. It is the only way an Agent's state leaves
// the domain; repositories map it to a row and RehydrateAgent mirrors its fields back.
type AgentSnapshot struct {
	ID             AgentID
	Name           string
	Archetype      Archetype
	Responsibility string
	Description    string
	Source         Source
	Enabled        bool
	ToolIDs        []ToolID
}

// Snapshot returns the agent's persistence/read view.
func (a Agent) Snapshot() AgentSnapshot {
	return AgentSnapshot{
		ID:             a.id,
		Name:           a.name,
		Archetype:      a.archetype,
		Responsibility: a.responsibility,
		Description:    a.description,
		Source:         a.source,
		Enabled:        a.enabled,
		ToolIDs:        a.toolIDs,
	}
}
