package decorator

import (
	"context"
	"log/slog"
	"time"

	"harness-workspace/services/harness/internal/app/appctx"
	"harness-workspace/services/harness/internal/domain"
)

// commandAuditDecorator records each command execution as an append-only audit event —
// the persisted counterpart to the (ephemeral) logging decorator. It captures the real
// outcome (recorded after the wrapped handler returns): the command name as the kind,
// ok/error as the status, and the actor from the context. Audit is observational: a
// failed append is logged, not propagated, so it never changes a command's outcome.
type commandAuditDecorator[C, R any] struct {
	base   CommandHandler[C, R]
	events domain.EventWriter
	logger *slog.Logger
}

func (d commandAuditDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	result, err = d.base.Handle(ctx, cmd)

	status, message := domain.EventOK, ""
	if err != nil {
		status, message = domain.EventError, err.Error()
	}
	ev, mkErr := domain.NewEvent(actionName(cmd), appctx.Actor(ctx), status, message, "", time.Now())
	if mkErr != nil {
		d.logger.Warn("audit event invalid", slog.Any("err", mkErr))
		return result, err
	}
	if aerr := d.events.Append(ctx, ev); aerr != nil {
		d.logger.Warn("audit append failed", slog.String("kind", actionName(cmd)), slog.Any("err", aerr))
	}
	return result, err
}
