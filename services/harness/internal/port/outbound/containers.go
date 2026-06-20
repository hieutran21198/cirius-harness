package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/container"
)

// Containers is the aggregate repository for the container domain. The plural name
// denotes the collection of Container aggregates.
type Containers interface {
	// Get returns the container with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (container.Container, error)
	// List returns all containers.
	List(ctx context.Context) ([]container.Container, error)
	// ListByProject returns the containers owned by the project with the given id.
	ListByProject(ctx context.Context, projectID string) ([]container.Container, error)
	// Save inserts or updates the container, keyed by ID.
	Save(ctx context.Context, c container.Container) error
	// Delete removes the container with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
