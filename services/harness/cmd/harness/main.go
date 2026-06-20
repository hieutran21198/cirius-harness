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
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/services/harness/internal/app"
	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/delivery/pilink"
	"harness-workspace/services/harness/internal/domain/model"
	"harness-workspace/services/harness/internal/infra/sqlite/unitofwork"
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
	defer func() { _ = sqlDB.Close() }()

	m, err := migrate.New(sqlDB, migrations.FS, migrate.DialectSQLite)
	if err != nil {
		return err
	}
	// Apply migrations on start so the per-session child is self-sufficient: the
	// schema must exist before models can be synced (ADR-0008 spirit — the harness
	// lifecycle is bound to the session, with nothing to run out of band).
	if err := m.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	version, err := m.Version(ctx)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	application := app.New(unitofwork.New(db), logger)
	h := &handler{dbPath: dbPath, migrator: m, app: application}

	fmt.Fprintf(os.Stderr, "harness serve: ready (db=%s, schema=v%d, pid=%d)\n", dbPath, version, os.Getpid())
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
	app      app.Application
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

// SyncModels adapts the wire frame to the SyncModels command: it translates the
// reported refs into domain models, drives the application handler, and maps the
// result back to the wire. No business logic lives here (ADR-0004, ADR-0012).
func (h *handler) SyncModels(ctx context.Context, req pilink.ModelsReq) (pilink.ModelsSyncedResp, error) {
	reported := make([]model.Model, len(req.Models))
	for i, ref := range req.Models {
		reported[i] = model.Model{Provider: ref.Provider, Slug: ref.Slug}
	}
	res, err := h.app.Commands.SyncModels.Handle(ctx, command.SyncModels{Reported: reported})
	if err != nil {
		return pilink.ModelsSyncedResp{}, err
	}
	if req.Client != "" {
		fmt.Fprintf(os.Stderr, "harness: synced models from %s (added=%d, total=%d)\n", req.Client, res.Added, res.Total)
	}
	return pilink.ModelsSyncedResp{Added: res.Added, Total: res.Total}, nil
}
