package decorator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// commandLoggingDecorator logs a command's execution around the wrapped handler:
// Debug on entry, Info on success or Error on failure (recorded in a defer so it
// reflects the real outcome). The action name is the command's type name.
type commandLoggingDecorator[C, R any] struct {
	base   CommandHandler[C, R]
	logger *slog.Logger
}

func (d commandLoggingDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	logger := d.logger.With(slog.String("command", actionName(cmd)))
	logger.Debug("executing command")
	defer func() {
		if err != nil {
			logger.Error("command failed", slog.Any("err", err))
			return
		}
		logger.Info("command executed")
	}()
	return d.base.Handle(ctx, cmd)
}

// queryLoggingDecorator is the read-side counterpart of commandLoggingDecorator.
type queryLoggingDecorator[Q, R any] struct {
	base   QueryHandler[Q, R]
	logger *slog.Logger
}

func (d queryLoggingDecorator[Q, R]) Handle(ctx context.Context, q Q) (result R, err error) {
	logger := d.logger.With(slog.String("query", actionName(q)))
	logger.Debug("executing query")
	defer func() {
		if err != nil {
			logger.Error("query failed", slog.Any("err", err))
			return
		}
		logger.Info("query executed")
	}()
	return d.base.Handle(ctx, q)
}

// actionName is the type's bare name (the last dotted segment of its fully
// qualified type), e.g. "SyncModels" for command.SyncModels.
func actionName(v any) string {
	name := fmt.Sprintf("%T", v)
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
