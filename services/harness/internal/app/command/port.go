package command

import (
	"context"

	"harness-workspace/services/harness/internal/domain/model"
)

// The command side's driven ports, defined where they are consumed (ADR-0013):
// commands mutate through a UnitOfWork, which exposes the per-aggregate domain
// Writers and runs them inside a transaction. The infra adapter implements these.

// TransactionalUnitOfWork exposes the writers available within a unit of work.
// Inside DoTx they are bound to the open transaction.
type TransactionalUnitOfWork interface {
	Models() model.Writer
}

// UnitOfWork is a TransactionalUnitOfWork whose writers autocommit per call, and
// which can also run a closure atomically via DoTx — every write inside fn shares
// one transaction that commits on nil and rolls back on error.
type UnitOfWork interface {
	TransactionalUnitOfWork
	DoTx(ctx context.Context, fn func(ctx context.Context, tx TransactionalUnitOfWork) error) error
}
