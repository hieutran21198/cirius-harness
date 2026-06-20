package model

import "context"

// Writer mutates the model catalog. It is a domain-owned driven port (the methods
// a command needs to sync the catalog): existence check for the added-count
// decision, an upsert, and the catalog count for the sync acknowledgement. It is
// obtained from a UnitOfWork and implemented by the infra adapter (ADR-0013).
type Writer interface {
	// Exists reports whether a model with the natural key (provider, slug) is present.
	Exists(ctx context.Context, provider, slug string) (bool, error)
	// Save upserts the model on its natural key (provider, slug).
	Save(ctx context.Context, m Model) error
	// Count returns the number of models in the catalog.
	Count(ctx context.Context) (int, error)
}
