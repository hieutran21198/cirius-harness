package outbound

import (
	"context"

	"harness-workspace/services/harness/internal/domain/tool"
)

// Tools is the aggregate repository for the tool catalog. The plural name denotes
// the collection of Tool aggregates.
type Tools interface {
	// Get returns the tool with the given id, or ErrNotFound.
	Get(ctx context.Context, id string) (tool.Tool, error)
	// List returns all tools.
	List(ctx context.Context) ([]tool.Tool, error)
	// Save inserts or updates the tool, keyed by ID.
	Save(ctx context.Context, t tool.Tool) error
	// Delete removes the tool with the given id, or returns ErrNotFound.
	Delete(ctx context.Context, id string) error
}
