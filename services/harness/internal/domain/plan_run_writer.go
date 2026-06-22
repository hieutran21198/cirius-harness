package domain

import "context"

// PlanRunWriter persists a PlanRun. Unlike the insert-once writers, Save UPSERTs: a drive reports
// progress repeatedly, so the run row and its task rows are overwritten in place (ADR-0021). It
// is a domain-owned driven port, obtained from a UnitOfWork and implemented by the infra adapter.
type PlanRunWriter interface {
	// Save upserts the run (status, updated_at) and each task run (status, summary, updated_at).
	Save(ctx context.Context, r PlanRun) error
}
