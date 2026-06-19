// Package gormdb is a dialect-agnostic GORM bootstrap. The caller builds a
// gorm.Dialector first (e.g. with gormdb/sqlite) and passes it to New, which
// wraps it with the shared logging and a liveness ping. Keeping the dialect out
// of this package lets it back any GORM-supported engine.
package gormdb

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Options configures New. Use the With* helpers rather than building this
// directly. Only dialect-agnostic settings live here; engine-specific tuning
// (e.g. SQLite pragmas) belongs to the dialector builder.
type Options struct {
	// Logger receives GORM's SQL logs. Nil disables GORM logging.
	Logger *slog.Logger
}

// Option mutates Options.
type Option func(*Options)

// WithLogger routes GORM SQL logs through the given slog.Logger.
func WithLogger(l *slog.Logger) Option { return func(o *Options) { o.Logger = l } }

// New opens a *gorm.DB over the given dialector, applying the shared logger and
// verifying the connection with a ping. Build the dialector with a dialect
// helper first, e.g.:
//
//	dialect, err := sqlite.New(path)
//	db, err := gormdb.New(ctx, dialect)
func New(ctx context.Context, dialector gorm.Dialector, opts ...Option) (*gorm.DB, error) {
	var cfg Options
	for _, opt := range opts {
		opt(&cfg)
	}

	gcfg := &gorm.Config{Logger: gormlogger.Discard}
	if cfg.Logger != nil {
		gcfg.Logger = newSlogLogger(cfg.Logger)
	}

	db, err := gorm.Open(dialector, gcfg)
	if err != nil {
		return nil, fmt.Errorf("gormdb.New: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gormdb.New: sql handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("gormdb.New: ping: %w", err)
	}
	return db, nil
}
