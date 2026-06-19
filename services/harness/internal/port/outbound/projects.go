package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/project"
)

// Projects is the aggregate repository for the project domain. The plural name
// denotes the collection of Project aggregates.
type Projects interface {
	// Get returns the project with the given name, or ErrNotFound.
	Get(ctx context.Context, name string) (project.Project, error)
	// List returns all projects.
	List(ctx context.Context) ([]project.Project, error)
	// Save inserts or updates the project, keyed by Name.
	Save(ctx context.Context, p project.Project) error
	// Delete removes the project with the given name, or returns ErrNotFound.
	Delete(ctx context.Context, name string) error
}
