// Command migrate manages the harness database migrations (embedded goose,
// pure-Go SQLite via gormdb).
//
// Usage:
//
//	migrate up|down|status [db-path]   apply / roll back / report migrations
//	migrate version [db-path]          print the current schema version
//	migrate create <purpose> [dir]     scaffold a new timestamped migration file
package main

import (
	"context"
	"fmt"
	"os"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/services/harness/migrations"
)

const (
	defaultDBPath        = ".cirius-harness/state/harness.sqlite"
	defaultMigrationsDir = "services/harness/migrations"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	if err := dispatch(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <up|down|status|version|create> [args]")
}

func dispatch(cmd string, args []string) error {
	switch cmd {
	case "up", "down", "status", "version":
		dbPath := defaultDBPath
		if len(args) >= 1 {
			dbPath = args[0]
		}
		return runDB(cmd, dbPath)
	case "create":
		if len(args) < 1 {
			usage()
			return fmt.Errorf("create needs a <purpose> argument")
		}
		dir := defaultMigrationsDir
		if len(args) >= 2 {
			dir = args[1]
		}
		// create writes to the source directory on disk; rebuild re-embeds it.
		return migrate.Create(dir, args[0])
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func runDB(cmd, dbPath string) error {
	ctx := context.Background()

	dialect, err := sqlite.New(dbPath)
	if err != nil {
		return err
	}
	db, err := gormdb.New(ctx, dialect)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	m, err := migrate.New(sqlDB, migrations.FS, migrate.DialectSQLite)
	if err != nil {
		return err
	}

	switch cmd {
	case "up":
		if err := m.Up(ctx); err != nil {
			return err
		}
		fmt.Printf("migrate up: ok (%s)\n", dbPath)
	case "down":
		if err := m.Down(ctx); err != nil {
			return err
		}
		fmt.Printf("migrate down: ok (%s)\n", dbPath)
	case "status":
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			state := "pending"
			if s.Applied {
				state = "applied"
			}
			fmt.Printf("%-7s  %d  %s\n", state, s.Version, s.Source)
		}
	case "version":
		v, err := m.Version(ctx)
		if err != nil {
			return err
		}
		fmt.Println(v)
	}
	return nil
}
