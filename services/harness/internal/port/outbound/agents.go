package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/agent"
)

// Agents is the aggregate repository for the agent domain. The plural name
// denotes the collection of Agent aggregates.
type Agents interface {
	// Get returns the agent with the given name, or ErrNotFound.
	Get(ctx context.Context, name string) (agent.Agent, error)
	// List returns all agents.
	List(ctx context.Context) ([]agent.Agent, error)
	// Save inserts or updates the agent, keyed by Name.
	Save(ctx context.Context, a agent.Agent) error
	// Delete removes the agent with the given name, or returns ErrNotFound.
	Delete(ctx context.Context, name string) error
}
