package model

import "context"

// Writer mutates the model catalog. It is a domain-owned driven port (the methods
// a command needs to sync the catalog): a targeted existence check over the reported
// refs, a batch upsert, and the catalog count for the sync acknowledgement. It is
// obtained from a UnitOfWork and implemented by the infra adapter (ADR-0013).
type Writer interface {
	// Existing returns which of the given refs are already in the catalog, as a set
	// keyed by Ref — a membership check before a batch upsert. The query is scoped to
	// the refs, so its cost scales with the request, not the catalog.
	Existing(ctx context.Context, refs []Ref) (map[Ref]struct{}, error)
	// SaveAll upserts the given models on their natural key (provider, slug) in one batch.
	SaveAll(ctx context.Context, ms []Model) error
	// Count returns the number of models in the catalog.
	Count(ctx context.Context) (int, error)
}
