// Package migrate is a thin, instance-based wrapper over goose's Provider. A
// Migrator binds one *sql.DB, one embedded migration filesystem, and one dialect
// — no process-wide globals. It is driver-agnostic: the caller supplies the
// connection (e.g. from gormdb) and the dialect.
//
// New migration files follow goose's timestamp naming —
// yyyymmddhhMMss_snake_case_purpose.sql — produced by Create.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"text/template"
	"time"

	"github.com/pressly/goose/v3"
)

// Dialect identifies the database engine goose generates SQL for.
type Dialect = goose.Dialect

// DialectSQLite is the dialect for SQLite databases.
const DialectSQLite = goose.DialectSQLite3

// sqlStub is the template for a freshly created migration: empty goose Up/Down
// sections, without goose's default placeholder statements.
var sqlStub = template.Must(template.New("migrate.sql-stub").Parse(`-- +goose Up

-- +goose Down
`))

// MigrationStatus reports whether a single migration has been applied.
type MigrationStatus struct {
	Version   int64
	Source    string // base filename, e.g. 20260619072802_initialize.sql
	Applied   bool
	AppliedAt time.Time
}

// Migrator applies and inspects the migrations in a fixed filesystem against a
// fixed database. Construct one with New.
type Migrator struct {
	provider *goose.Provider
}

// New builds a Migrator for db, reading migrations from fsys and generating SQL
// for dialect.
func New(db *sql.DB, fsys fs.FS, dialect Dialect) (*Migrator, error) {
	provider, err := goose.NewProvider(dialect, db, fsys)
	if err != nil {
		return nil, fmt.Errorf("migrate.New: %w", err)
	}
	return &Migrator{provider: provider}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context) error {
	if _, err := m.provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate.Up: %w", err)
	}
	return nil
}

// Down rolls back the most recently applied migration.
func (m *Migrator) Down(ctx context.Context) error {
	if _, err := m.provider.Down(ctx); err != nil {
		return fmt.Errorf("migrate.Down: %w", err)
	}
	return nil
}

// Version returns the current schema version recorded in the database.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	v, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate.Version: %w", err)
	}
	return v, nil
}

// Status returns the applied/pending state of each known migration, ordered by
// version.
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	statuses, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate.Status: %w", err)
	}
	out := make([]MigrationStatus, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, MigrationStatus{
			Version:   s.Source.Version,
			Source:    filepath.Base(s.Source.Path),
			Applied:   s.State == goose.StateApplied,
			AppliedAt: s.AppliedAt,
		})
	}
	return out, nil
}

// Create writes a new blank migration file to dir, named
// yyyymmddhhMMss_<snake_case_name>.sql (goose stamps the UTC timestamp and
// snake-cases the name). It writes to the source directory on disk; rebuild to
// re-embed it. It fails if a file with the computed name already exists.
//
// Create is dialect-independent — it only scaffolds a file — so it is a
// standalone function rather than a Migrator method.
func Create(dir, name string) error {
	if err := goose.CreateWithTemplate(nil, dir, sqlStub, name, "sql"); err != nil {
		return fmt.Errorf("migrate.Create: %w", err)
	}
	return nil
}
