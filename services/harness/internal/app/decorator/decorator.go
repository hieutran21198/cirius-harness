// Package decorator defines the CQRS handler contracts for the application layer
// and the cross-cutting decorators wrapped around them. Every use case is a
// CommandHandler (writes) or QueryHandler (reads); decorators add concerns like
// logging without touching the handler's business logic (the open/closed seam).
//
// The contracts are generic so one decorator serves every handler. Today the only
// decorator is logging (slog → stderr); metrics is intentionally absent until there
// is a backend. If a second service needs these, promote the package to
// packages/go/cqrs (see ADR-0012).
package decorator

import (
	"context"
	"log/slog"

	"harness-workspace/services/harness/internal/domain"
)

// CommandHandler executes a state-changing use case. Unlike canonical CQRS
// (commands return only an error), Handle returns a result: harness commands
// acknowledge an outcome over the wire (e.g. the model-sync counts). See ADR-0012.
type CommandHandler[C, R any] interface {
	Handle(ctx context.Context, cmd C) (R, error)
}

// QueryHandler executes a read-only use case. Defined now so the query side has
// its contract ready; there are no queries yet.
type QueryHandler[Q, R any] interface {
	Handle(ctx context.Context, q Q) (R, error)
}

// ApplyCommandDecorators wraps a command handler with the cross-cutting concerns:
// audit (persisted, when an EventWriter is given) then logging (ephemeral). The nesting
// is outermost-first — logging → audit → base handler — so logging brackets the audit
// write too. A nil events writer skips the audit layer (e.g. in tests).
func ApplyCommandDecorators[C, R any](handler CommandHandler[C, R], logger *slog.Logger, events domain.EventWriter) CommandHandler[C, R] {
	wrapped := handler
	if events != nil {
		wrapped = commandAuditDecorator[C, R]{base: handler, events: events, logger: logger}
	}
	return commandLoggingDecorator[C, R]{base: wrapped, logger: logger}
}

// ApplyQueryDecorators wraps a query handler with the cross-cutting concerns.
func ApplyQueryDecorators[Q, R any](handler QueryHandler[Q, R], logger *slog.Logger) QueryHandler[Q, R] {
	return queryLoggingDecorator[Q, R]{base: handler, logger: logger}
}
