// Package sqlite builds a pure-Go SQLite gorm.Dialector (glebarez/sqlite, no
// CGO) for use with gormdb.New. It owns the SQLite-specific concerns — the DSN
// pragmas and creating the parent state directory — so the core gormdb package
// stays dialect-agnostic.
//
// Usage:
//
//	dialect, err := sqlite.New(path)
//	db, err := gormdb.New(ctx, dialect)
package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	glebarez "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Options configures New. Use the With* helpers rather than building this
// directly; zero values fall back to the defaults documented on each field.
type Options struct {
	// BusyTimeout is the SQLite busy_timeout. Default 5s.
	BusyTimeout time.Duration
	// JournalMode is the SQLite journal_mode. Default "WAL".
	JournalMode string
	// ForeignKeys toggles the foreign_keys pragma. Default true.
	ForeignKeys bool
}

// Option mutates Options.
type Option func(*Options)

// WithBusyTimeout sets the SQLite busy_timeout.
func WithBusyTimeout(d time.Duration) Option { return func(o *Options) { o.BusyTimeout = d } }

// WithJournalMode sets the SQLite journal_mode (e.g. "WAL", "DELETE").
func WithJournalMode(mode string) Option { return func(o *Options) { o.JournalMode = mode } }

// WithForeignKeys toggles the foreign_keys pragma.
func WithForeignKeys(on bool) Option { return func(o *Options) { o.ForeignKeys = on } }

func defaults() Options {
	return Options{
		BusyTimeout: 5 * time.Second,
		JournalMode: "WAL",
		ForeignKeys: true,
	}
}

// New returns a SQLite gorm.Dialector for the database at path, creating the
// parent directory if needed. The conventional path is
// .cirius-harness/state/{service}.sqlite, chosen by the caller. It returns an
// error only when the parent directory cannot be created — the connection itself
// is opened later by gormdb.New.
func New(path string, opts ...Option) (gorm.Dialector, error) {
	cfg := defaults()
	for _, opt := range opts {
		opt(&cfg)
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("sqlite.New: create state dir: %w", err)
		}
	}

	fk := 0
	if cfg.ForeignKeys {
		fk = 1
	}
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(%d)&_pragma=busy_timeout(%d)&_pragma=journal_mode(%s)",
		path, fk, cfg.BusyTimeout.Milliseconds(), cfg.JournalMode,
	)
	return glebarez.Open(dsn), nil
}
