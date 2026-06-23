// Command harness is the control-plane entrypoint. Today it exposes one
// subcommand, `serve`, the inbound wire for the Pi coding client: a
// newline-delimited JSON handshake over stdio (see ADR-0008). The Pi extension
// in .pi/extensions/harness launches `harness serve` per session.
//
// Usage:
//
//	harness serve [db-path]   speak the Pi stdio (NDJSON) protocol on stdin/stdout
//
// stdout is the protocol channel; logs go to a per-session file (see newLogger).
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/packages/go/slogx"
	"harness-workspace/services/harness/internal/app"
	"harness-workspace/services/harness/internal/delivery/pilink"
	"harness-workspace/services/harness/internal/infra/config"
	"harness-workspace/services/harness/internal/infra/sqlite/readstore"
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

	// A session is one serve lifecycle (one client⇄child-harness, ADR-0009). Mint its
	// id up front: it names the per-session log file, tags every log line, and is reused
	// by the session record.
	sessionID := uuid.Must(uuid.NewV7()).String()

	// State lives next to the DB; config (system base + user overlay) lives one level up
	// in the .cirius-harness directory.
	stateDir := filepath.Dir(dbPath)
	cfg, err := config.Load(filepath.Dir(stateDir))
	if err != nil {
		return err
	}
	level, err := resolveLogLevel(cfg.Logging.Level)
	if err != nil {
		return err
	}

	logger, logPath, closeLog, err := newLogger(os.Stderr, stateDir, sessionID, level)
	if err != nil {
		return err
	}
	defer closeLog()

	logger.Info("starting", slog.String("db", dbPath), slog.String("log", logPath), slog.Int("pid", os.Getpid()))

	dialect, err := sqlite.New(dbPath)
	if err != nil {
		return err
	}
	// Route GORM's own diagnostics through the same logger (Warn+; SQL stays quiet).
	db, err := gormdb.New(ctx, dialect, gormdb.WithLogger(logger))
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
	if err = m.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	version, err := m.Version(ctx)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	logger.Info("schema migrated", slog.Int64("version", version))

	application := app.New(unitofwork.New(db), readstore.New(db), logger)
	h := pilink.NewHandler(application, m, logger, dbPath, sessionID)

	logger.Info("ready", slog.String("db", dbPath), slog.Int64("schema", version), slog.Int("pid", os.Getpid()))
	if err := pilink.Serve(ctx, os.Stdin, os.Stdout, h, logger); err != nil {
		// Context cancellation (signal) is a clean shutdown, not a failure.
		if ctx.Err() != nil {
			logger.Info("shutting down", slog.String("reason", "signal"))
			return nil
		}
		return err
	}
	logger.Info("shutting down", slog.String("reason", "stdin closed"))
	return nil
}

// resolveLogLevel decides the log level: the config value (system base 00-system.yaml,
// overridden by the user overlay config.yaml) is the source of truth, and the
// HARNESS_LOG_LEVEL env var is a final ad-hoc override (handy for one-off debugging).
// An empty config + no env defaults to info.
func resolveLogLevel(configLevel string) (slog.Level, error) {
	name := configLevel
	if v := os.Getenv("HARNESS_LOG_LEVEL"); v != "" {
		name = v
	}
	if name == "" {
		return slog.LevelInfo, nil
	}
	return slogx.ParseLevel(name)
}

// newLogger builds the per-session serve logger. Logs go to a per-session file at
// <stateDir>/logging/<sessionID>.log — NOT to the console: an AI client launches the
// harness as a child and relays its stderr into the client's own TUI (the Pi extension
// forwards it to console.error), so teeing logs to stderr would mix log records into the
// client's UI. The console writer is used only as the fallback when the file is disabled.
// HARNESS_LOG_FILE overrides the path; "-" disables the file and logs to the console
// instead (the escape hatch for "I want console logs"). HARNESS_LOG_FORMAT selects text
// (default) or json. Every line is tagged with the session id. It returns the resolved
// log path ("" when the file is disabled) and a closer for the file sink. Logs never
// touch stdout — that is the protocol channel.
func newLogger(console io.Writer, stateDir, sessionID string, level slog.Level) (*slog.Logger, string, func(), error) {
	format := slogx.FormatText
	if strings.EqualFold(os.Getenv("HARNESS_LOG_FORMAT"), "json") {
		format = slogx.FormatJSON
	}

	logPath := filepath.Join(stateDir, "logging", sessionID+".log")
	if v, ok := os.LookupEnv("HARNESS_LOG_FILE"); ok {
		logPath = v // explicit override; "-" disables the file
	}

	w := console
	closeLog := func() {}
	if logPath != "" && logPath != "-" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, "", nil, fmt.Errorf("create log dir: %w", err)
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, "", nil, fmt.Errorf("open log file %q: %w", logPath, err)
		}
		w = f // file only — never tee to the console, where it would land in the client's UI
		closeLog = func() { _ = f.Close() }
	} else {
		logPath = "" // console only (file disabled)
	}
	logger := slogx.New(w, level, format).With(slog.String("session", sessionID))
	return logger, logPath, closeLog, nil
}
