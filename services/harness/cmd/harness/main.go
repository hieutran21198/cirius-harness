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
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"harness-workspace/packages/go/gormdb"
	"harness-workspace/packages/go/gormdb/sqlite"
	"harness-workspace/packages/go/migrate"
	"harness-workspace/packages/go/slogx"
	"harness-workspace/services/harness/internal/app"
	"harness-workspace/services/harness/internal/app/appctx"
	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/delivery/pilink"
	"harness-workspace/services/harness/internal/domain"
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

	logger, logPath, closeLog, err := newLogger(stateDir, sessionID, level)
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
	h := &handler{dbPath: dbPath, migrator: m, app: application, logger: logger, sessionID: sessionID}

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

// newLogger builds the per-session serve logger. Logs always go to stderr and, by
// default, ALSO to a per-session file at <stateDir>/logging/<sessionID>.log so they
// are discoverable however the harness is launched — the Pi extension forwards the
// child's stderr to its own console, where it is hard to inspect, but the file
// survives. HARNESS_LOG_FILE overrides the path; "-" disables the file (stderr only).
// HARNESS_LOG_FORMAT selects text (default) or json. Every line is tagged with the
// session id. It returns the resolved log path ("" when disabled) and a closer for the
// file sink. Logs never touch stdout — that is the protocol channel.
func newLogger(stateDir, sessionID string, level slog.Level) (*slog.Logger, string, func(), error) {
	format := slogx.FormatText
	if strings.EqualFold(os.Getenv("HARNESS_LOG_FORMAT"), "json") {
		format = slogx.FormatJSON
	}

	logPath := filepath.Join(stateDir, "logging", sessionID+".log")
	if v, ok := os.LookupEnv("HARNESS_LOG_FILE"); ok {
		logPath = v // explicit override; "-" disables the file
	}

	var w io.Writer = os.Stderr
	closeLog := func() {}
	if logPath != "" && logPath != "-" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, "", nil, fmt.Errorf("create log dir: %w", err)
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, "", nil, fmt.Errorf("open log file %q: %w", logPath, err)
		}
		w = io.MultiWriter(os.Stderr, f)
		closeLog = func() { _ = f.Close() }
	} else {
		logPath = "" // stderr only
	}
	logger := slogx.New(w, level, format).With(slog.String("session", sessionID))
	return logger, logPath, closeLog, nil
}

// handler implements pilink.Handler against the harness store. The ready frame
// reports the applied schema version — proof the harness reached its migrated
// state, the smallest honest liveness signal for the connect-only slice.
type handler struct {
	dbPath    string
	migrator  *migrate.Migrator
	app       app.Application
	logger    *slog.Logger
	sessionID string
	// sessionStarted is set once the session row exists (after a hello with a cwd), so
	// later agent-run recording has a session to attach to.
	sessionStarted bool
}

func (h *handler) Hello(ctx context.Context, req pilink.HelloReq) (pilink.ReadyResp, error) {
	version, err := h.migrator.Version(ctx)
	if err != nil {
		return pilink.ReadyResp{}, fmt.Errorf("read schema version: %w", err)
	}
	h.logger.Info("client hello", slog.String("cwd", req.CWD), slog.Int("client_pid", req.PID))

	// Record the session start (best-effort: a recording failure must not abort the
	// handshake). Needs the project root from the client's cwd; skip if absent.
	if req.CWD != "" {
		_, err := h.app.Commands.StartSession.Handle(ctx, command.StartSession{
			SessionID:   domain.SessionID(h.sessionID),
			ProjectRoot: req.CWD,
			ProjectName: filepath.Base(req.CWD),
			CreatedAt:   time.Now(),
		})
		if err != nil {
			h.logger.Warn("record session failed", slog.Any("err", err))
		} else {
			h.sessionStarted = true
			h.logger.Info("session started", slog.String("session", h.sessionID))
		}
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
	// The client is frame-level (one frame is one client's report) and part of every
	// reported model's catalog identity, so it must be a known client.
	client := domain.ClientKind(req.Client)
	if !client.Valid() {
		return pilink.ModelsSyncedResp{}, fmt.Errorf("unknown or missing client %q", req.Client)
	}
	reported := make([]domain.Ref, len(req.Models))
	for i, ref := range req.Models {
		reported[i] = domain.Ref{Client: client, Provider: ref.Provider, Slug: ref.Slug}
	}
	ctx = appctx.WithActor(ctx, string(client))
	res, err := h.app.Commands.SyncModels.Handle(ctx, command.SyncModels{Reported: reported})
	if err != nil {
		return pilink.ModelsSyncedResp{}, err
	}
	h.logger.Info("models synced", slog.String("client", string(client)), slog.Int("added", res.Added), slog.Int("total", res.Total))
	return pilink.ModelsSyncedResp{Added: res.Added, Total: res.Total}, nil
}

// ResolveAgent adapts the wire frame to the ResolveAgent query: it validates the
// client, drives the application query, and maps the resolved persona back to the
// wire. No business logic lives here (ADR-0004, ADR-0012).
func (h *handler) ResolveAgent(ctx context.Context, req pilink.ResolveAgentReq) (pilink.AgentResolvedResp, error) {
	// The client is part of the (later) client-specific model resolution, so it must be
	// a known client even though the persona itself is client-agnostic.
	client := domain.ClientKind(req.Client)
	if !client.Valid() {
		return pilink.AgentResolvedResp{}, fmt.Errorf("unknown or missing client %q", req.Client)
	}
	res, err := h.app.Queries.ResolveAgent.Handle(ctx, query.ResolveAgent{Name: req.Agent, Client: client})
	if err != nil {
		return pilink.AgentResolvedResp{}, err
	}
	h.logger.Info("agent resolved", slog.String("agent", res.Name), slog.String("client", string(client)))

	// Record that this agent ran in the session (best-effort; needs a started session).
	if h.sessionStarted {
		ctx = appctx.WithActor(ctx, string(client))
		_, rerr := h.app.Commands.RecordAgentRun.Handle(ctx, command.RecordAgentRun{
			SessionID: domain.SessionID(h.sessionID),
			AgentID:   res.AgentID,
			ModelID:   domain.ModelID(res.Model),
		})
		if rerr != nil {
			h.logger.Warn("record agent run failed", slog.String("agent", res.Name), slog.Any("err", rerr))
		}
	}

	return pilink.AgentResolvedResp{Name: res.Name, Persona: res.Persona, Model: res.Model}, nil
}
