package model

import "context"

// Writer mutates the model catalog. It is a domain-owned driven port (the methods
// a command needs to sync the catalog): a bulk read of the present natural keys for
// an in-memory existence check, a batch upsert, and the catalog count for the sync
// acknowledgement. It is obtained from a UnitOfWork and implemented by the infra
// adapter (ADR-0013).
type Writer interface {
	// ExistingKeys returns the natural keys ("provider/slug") already in the catalog,
	// as a set for an in-memory membership check before a batch upsert.
	ExistingKeys(ctx context.Context) (map[string]struct{}, error)
	// SaveAll upserts the given models on their natural key (provider, slug) in one batch.
	SaveAll(ctx context.Context, ms []Model) error
	// Count returns the number of models in the catalog.
	Count(ctx context.Context) (int, error)
}
