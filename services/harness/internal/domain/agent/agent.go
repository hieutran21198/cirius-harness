// Package agent is the agent bounded context: the Agent aggregate and the
// Agents repository port. Permissions are NOT modelled here — authorization is
// a separate concern backed by Casbin (see internal/domain/authz).
package agent

import (
	"errors"
	"fmt"
)

// ErrInvalidAgent is returned by Validate for a structurally invalid agent.
var ErrInvalidAgent = errors.New("agent: invalid")

// Agent is the aggregate root describing one member of the harness team. The
// agent is also the authorization principal: its Name is the Casbin subject.
type Agent struct {
	// Name uniquely identifies the agent and is its natural key.
	Name string
	// Model is the primary "provider/model-id"; empty means model-less (prayer).
	Model string
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
	// Tools lists the capabilities the agent may use.
	Tools []Tool
	// Fallbacks is the ordered list of "provider/model-id" tried after Model.
	Fallbacks []string
}

// HasModel reports whether the agent has a model (prayer does not).
func (a Agent) HasModel() bool { return a.Model != "" }

// Validate checks the agent's invariants.
func (a Agent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidAgent)
	}
	if !a.Source.Valid() {
		return fmt.Errorf("%w: unknown source %q", ErrInvalidAgent, a.Source)
	}
	if !a.Archetype.Valid() {
		return fmt.Errorf("%w: unknown archetype %q", ErrInvalidAgent, a.Archetype)
	}
	for _, t := range a.Tools {
		if !t.Valid() {
			return fmt.Errorf("%w: unknown tool %q", ErrInvalidAgent, t)
		}
	}
	return nil
}
