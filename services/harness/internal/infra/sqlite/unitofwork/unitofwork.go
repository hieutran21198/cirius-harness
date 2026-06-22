// Package unitofwork is the GORM-backed implementation of the app's
// command.UnitOfWork (ADR-0013): it composes the repo writers and runs them inside
// a transaction.
package unitofwork

import (
	"context"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

// UnitOfWork is the command.UnitOfWork. Its writers autocommit per call over the
// base connection; DoTx runs a closure inside a single GORM transaction, handing
// the closure a UnitOfWork bound to that transaction.
type UnitOfWork struct {
	db *gorm.DB
}

// New builds a UnitOfWork over db.
func New(db *gorm.DB) *UnitOfWork { return &UnitOfWork{db: db} }

// Models returns the model catalog writer bound to this unit of work's handle.
func (u *UnitOfWork) Models() domain.ModelWriter { return repo.NewModelWriter(u.db) }

// Events returns the audit-log writer bound to this unit of work's handle.
func (u *UnitOfWork) Events() domain.EventWriter { return repo.NewEventWriter(u.db) }

// Projects returns the project writer bound to this unit of work's handle.
func (u *UnitOfWork) Projects() domain.ProjectWriter { return repo.NewProjectWriter(u.db) }

// Sessions returns the session writer bound to this unit of work's handle.
func (u *UnitOfWork) Sessions() domain.SessionWriter { return repo.NewSessionWriter(u.db) }

// Plans returns the orchestration-plan writer bound to this unit of work's handle.
func (u *UnitOfWork) Plans() domain.PlanWriter { return repo.NewPlanWriter(u.db) }

// PlanRuns returns the plan-run writer (drive progress) bound to this unit of work's handle.
func (u *UnitOfWork) PlanRuns() domain.PlanRunWriter { return repo.NewPlanRunWriter(u.db) }

// PlanReader returns the plan reader bound to this unit of work's handle, for an
// in-transaction read (e.g. ReportRun seeding a run from the plan's task refs).
func (u *UnitOfWork) PlanReader() domain.PlanReader { return repo.NewPlanReader(u.db) }

// PlanRunReader returns the plan-run reader bound to this unit of work's handle.
func (u *UnitOfWork) PlanRunReader() domain.PlanRunReader { return repo.NewPlanRunReader(u.db) }

// DoTx runs fn inside one transaction: every writer obtained from the txU shares
// it, committing on nil and rolling back on error (or panic).
func (u *UnitOfWork) DoTx(ctx context.Context, fn func(ctx context.Context, tx command.TransactionalUnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, &UnitOfWork{db: tx})
	})
}

// staticcheck: ensure UnitOfWork satisfies the command port.
var _ command.UnitOfWork = (*UnitOfWork)(nil)
