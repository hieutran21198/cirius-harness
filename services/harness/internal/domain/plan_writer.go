package domain

import "context"

// PlanWriter persists a council orchestration plan and all its children (tasks, risks,
// approvals, waves, and the wave→task membership). It is a domain-owned driven port obtained
// from a UnitOfWork and implemented by the infra adapter (ADR-0013).
type PlanWriter interface {
	// Save inserts the plan and its children. It is idempotent on the plan id (re-submitting
	// the same plan is a no-op), so it does not clobber an existing record.
	Save(ctx context.Context, p Plan) error
}
