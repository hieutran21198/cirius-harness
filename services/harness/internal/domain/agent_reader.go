package domain

import (
	"context"
	"errors"
)

// ErrAgentNotFound is returned by an AgentReader when no agent matches the lookup.
var ErrAgentNotFound = errors.New("agent: not found")

// AgentReader reads the agent team. It is a domain-owned driven port (the methods a
// query needs to resolve an agent): a lookup by the agent's natural key (its name).
// It is obtained from a ReadStore and implemented by the infra adapter (ADR-0013).
type AgentReader interface {
	// FindByName returns the agent with the given name, or ErrAgentNotFound if none.
	FindByName(ctx context.Context, name string) (Agent, error)
}
