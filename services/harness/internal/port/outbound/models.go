package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/model"
)

// Models is the aggregate repository for the model catalog. The plural name
// denotes the collection of Model aggregates.
type Models interface {
	// Get returns the model with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (model.Model, error)
	// List returns all models.
	List(ctx context.Context) ([]model.Model, error)
	// Save inserts or updates the model, keyed by ID.
	Save(ctx context.Context, m model.Model) error
	// Delete removes the model with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
