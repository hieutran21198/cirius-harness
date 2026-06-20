// Command harness is the control-plane entrypoint. Today it exposes one
// subcommand, `serve`, the inbound wire for the Pi coding client: a
// newline-delimited JSON handshake over stdio (see ADR-0008). The Pi extension
// in .pi/extensions/harness launches `harness serve` per session.
//
// Usage:
//
//	harness serve [db-path]   speak the Pi stdio (NDJSON) protocol on stdin/stdout
//
// stdout is the protocol channel; all logs go to stderr.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/services/harness/internal/adapter/inbound/pilink"
	"harness-workspace/services/harness/migrations"
)

const defaultDBPath = ".cirius-harness/state/harness.sqlite"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	if err := dispatch(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "harness:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: harness serve [db-path]")
}

func dispatch(cmd string, args []string) error {
	switch cmd {
	case "serve":
		dbPath := defaultDBPath
		if len(args) >= 1 {
			dbPath = args[0]
		}
		return serve(dbPath)
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// serve opens the harness store and runs the Pi stdio handshake loop until stdin
// closes or the process is signalled (SIGINT/SIGTERM).
func serve(dbPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	defer sqlDB.Close()

	m, err := migrate.New(sqlDB, migrations.FS, migrate.DialectSQLite)
	if err != nil {
		return err
	}

	h := &handler{dbPath: dbPath, migrator: m}

	fmt.Fprintf(os.Stderr, "harness serve: ready (db=%s, pid=%d)\n", dbPath, os.Getpid())
	if err := pilink.Serve(ctx, os.Stdin, os.Stdout, h); err != nil {
		// Context cancellation (signal) is a clean shutdown, not a failure.
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	return nil
}

// handler implements pilink.Handler against the harness store. The ready frame
// reports the applied schema version — proof the harness reached its migrated
// state, the smallest honest liveness signal for the connect-only slice.
type handler struct {
	dbPath   string
	migrator *migrate.Migrator
}

func (h *handler) Hello(ctx context.Context, req pilink.HelloReq) (pilink.ReadyResp, error) {
	version, err := h.migrator.Version(ctx)
	if err != nil {
		return pilink.ReadyResp{}, fmt.Errorf("read schema version: %w", err)
	}
	return pilink.ReadyResp{
		SchemaVersion: version,
		DBPath:        h.dbPath,
		PID:           os.Getpid(),
	}, nil
}
