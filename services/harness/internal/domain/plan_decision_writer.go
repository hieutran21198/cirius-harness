package domain

import "context"

// PlanDecisionWriter persists a PlanDecision. Save inserts a new decision row (append-only — each
// iteration records its own, ADR-0023). A domain-owned driven port, obtained from a UnitOfWork
// and implemented by the infra adapter.
type PlanDecisionWriter interface {
	// Save inserts the decision (its CouncilDecision JSON, created_at) for its run.
	Save(ctx context.Context, d PlanDecision) error
}
