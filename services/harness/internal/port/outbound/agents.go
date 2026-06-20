package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/agent"
)

// Agents is the aggregate repository for the agent domain. The plural name
// denotes the collection of Agent aggregates.
type Agents interface {
	// Get returns the agent with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (agent.Agent, error)
	// List returns all agents.
	List(ctx context.Context) ([]agent.Agent, error)
	// Save inserts or updates the agent, keyed by ID.
	Save(ctx context.Context, a agent.Agent) error
	// Delete removes the agent with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
