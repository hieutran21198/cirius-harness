// Package agent is the agent bounded context: the Agent aggregate (a pure
// identity/role) and its repository port. An agent carries no model — which model
// plays the role is bound per session (see internal/domain/session). Tools are a
// separate catalog (see internal/domain/tool), granted via the agent_tools
// junction. Permissions are NOT modelled here — authorization is a separate
// concern backed by Casbin (see internal/domain/authz).
package agent

import (
	"errors"
	"fmt"
)

// ErrInvalidAgent is returned by Validate for a structurally invalid agent.
var ErrInvalidAgent = errors.New("agent: invalid")

// Agent is the aggregate root describing one member of the harness team — a pure
// identity/role. It carries no model (the model is bound per session) and no
// fallbacks; its tools are catalog grants in ToolIDs. The agent is also the
// authorization principal: its Name is the Casbin subject.
type Agent struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// Name is the agent's unique business key and the authorization principal
	// (the Casbin subject).
	Name string
	// Responsibility describes, in free text, what the agent is responsible for.
	Responsibility string
	// Archetype is the agent's purpose-level operating style.
	Archetype Archetype
	// Description is a longer human-facing purpose statement.
	Description string
	// Source records whether the agent is a system default or user-defined.
	Source Source
	// Enabled reports whether the agent is active.
	Enabled bool
	// ToolIDs are the ids of the tools granted to this agent (catalog references,
	// persisted via the agent_tools junction). The model is NOT here — it is bound
	// per session on session_agents.model_id (see internal/domain/session).
	ToolIDs []string
}

// New assembles an agent (a team role) from an app-minted id and its attributes,
// enabled by default, and validates it. The id is supplied by the
// application/adapter, not generated here.
func New(id, name string, archetype Archetype, responsibility, description string, source Source, toolIDs []string) (Agent, error) {
	a := Agent{
		ID:             id,
		Name:           name,
		Responsibility: responsibility,
		Archetype:      archetype,
		Description:    description,
		Source:         source,
		Enabled:        true,
		ToolIDs:        toolIDs,
	}
	return a, a.Validate()
}

// Validate checks the agent's invariants.
func (a Agent) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidAgent)
	}
	if a.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidAgent)
	}
	if !a.Source.Valid() {
		return fmt.Errorf("%w: unknown source %q", ErrInvalidAgent, a.Source)
	}
	if !a.Archetype.Valid() {
		return fmt.Errorf("%w: unknown archetype %q", ErrInvalidAgent, a.Archetype)
	}
	return nil
}
