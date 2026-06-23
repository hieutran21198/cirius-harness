package command

import (
	"context"

	"harness-workspace/services/harness/internal/domain"
)

// The command side's driven ports, defined where they are consumed (ADR-0013):
// commands mutate through a UnitOfWork, which exposes the per-aggregate domain
// Writers and runs them inside a transaction. The infra adapter implements these.

// TransactionalUnitOfWork exposes the writers available within a unit of work.
// Inside DoTx they are bound to the open transaction. It also exposes the plan readers a
// command needs for an in-transaction read-modify-write (ReportRun loads the plan's refs and
// its run before mutating — keeping the read and the write under one lock).
type TransactionalUnitOfWork interface {
	Models() domain.ModelWriter
	Events() domain.EventWriter
	Projects() domain.ProjectWriter
	Sessions() domain.SessionWriter
	Plans() domain.PlanWriter
	PlanRuns() domain.PlanRunWriter
	TaskReports() domain.TaskReportWriter
	PlanDecisions() domain.PlanDecisionWriter
	PlanReader() domain.PlanReader
	PlanRunReader() domain.PlanRunReader
}

// UnitOfWork is a TransactionalUnitOfWork whose writers autocommit per call, and
// which can also run a closure atomically via DoTx — every write inside fn shares
// one transaction that commits on nil and rolls back on error.
type UnitOfWork interface {
	TransactionalUnitOfWork
	DoTx(ctx context.Context, fn func(ctx context.Context, tx TransactionalUnitOfWork) error) error
}
