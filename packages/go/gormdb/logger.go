package gormdb

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// slogLogger adapts a *slog.Logger to gorm's logger.Interface.
type slogLogger struct {
	log   *slog.Logger
	level logger.LogLevel
}

func newSlogLogger(l *slog.Logger) logger.Interface {
	return &slogLogger{log: l, level: logger.Warn}
}

// LogMode returns a copy of the logger set to the given gorm log level.
func (s *slogLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *s
	clone.level = level
	return &clone
}

func (s *slogLogger) Info(ctx context.Context, msg string, data ...any) {
	if s.level >= logger.Info {
		s.log.InfoContext(ctx, msg, slog.Any("data", data))
	}
}

func (s *slogLogger) Warn(ctx context.Context, msg string, data ...any) {
	if s.level >= logger.Warn {
		s.log.WarnContext(ctx, msg, slog.Any("data", data))
	}
}

func (s *slogLogger) Error(ctx context.Context, msg string, data ...any) {
	if s.level >= logger.Error {
		s.log.ErrorContext(ctx, msg, slog.Any("data", data))
	}
}

func (s *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if s.level <= logger.Silent {
		return
	}
	sql, rows := fc()
	attrs := []any{
		slog.String("sql", sql),
		slog.Int64("rows", rows),
		slog.Duration("elapsed", time.Since(begin)),
	}
	switch {
	case err != nil && !errors.Is(err, logger.ErrRecordNotFound) && s.level >= logger.Error:
		s.log.ErrorContext(ctx, "gorm trace", append(attrs, slog.Any("error", err))...)
	case s.level >= logger.Info:
		s.log.DebugContext(ctx, "gorm trace", attrs...)
	}
}
