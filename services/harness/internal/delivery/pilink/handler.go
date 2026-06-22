package pilink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"harness-workspace/packages/go/migrate"
	"harness-workspace/services/harness/internal/app"
	"harness-workspace/services/harness/internal/app/appctx"
	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/domain"
)

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

// NewHandler builds the harness-side pilink.Handler, wiring the composed application,
// migrator, and logger that the serve loop dispatches to.
func NewHandler(application app.Application, migrator *migrate.Migrator, logger *slog.Logger, dbPath, sessionID string) Handler {
	return &handler{dbPath: dbPath, migrator: migrator, app: application, logger: logger, sessionID: sessionID}
}

func (h *handler) Hello(ctx context.Context, req HelloReq) (ReadyResp, error) {
	version, err := h.migrator.Version(ctx)
	if err != nil {
		return ReadyResp{}, fmt.Errorf("read schema version: %w", err)
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

	return ReadyResp{
		SchemaVersion: version,
		DBPath:        h.dbPath,
		PID:           os.Getpid(),
	}, nil
}

// SyncModels adapts the wire frame to the SyncModels command: it translates the
// reported refs into domain models, drives the application handler, and maps the
// result back to the wire. No business logic lives here (ADR-0004, ADR-0012).
func (h *handler) SyncModels(ctx context.Context, req ModelsReq) (ModelsSyncedResp, error) {
	// The client is frame-level (one frame is one client's report) and part of every
	// reported model's catalog identity, so it must be a known client.
	client := domain.ClientKind(req.Client)
	if !client.Valid() {
		return ModelsSyncedResp{}, fmt.Errorf("unknown or missing client %q", req.Client)
	}
	reported := make([]domain.Ref, len(req.Models))
	for i, ref := range req.Models {
		reported[i] = domain.Ref{Client: client, Provider: ref.Provider, Slug: ref.Slug}
	}
	ctx = appctx.WithActor(ctx, string(client))
	res, err := h.app.Commands.SyncModels.Handle(ctx, command.SyncModels{Reported: reported})
	if err != nil {
		return ModelsSyncedResp{}, err
	}
	h.logger.Info("models synced", slog.String("client", string(client)), slog.Int("added", res.Added), slog.Int("total", res.Total))
	return ModelsSyncedResp{Added: res.Added, Total: res.Total}, nil
}

// ResolveAgent adapts the wire frame to the ResolveAgent query: it validates the
// client, drives the application query, and maps the resolved persona back to the
// wire. No business logic lives here (ADR-0004, ADR-0012).
func (h *handler) ResolveAgent(ctx context.Context, req ResolveAgentReq) (AgentResolvedResp, error) {
	// The client is part of the (later) client-specific model resolution, so it must be
	// a known client even though the persona itself is client-agnostic.
	client := domain.ClientKind(req.Client)
	if !client.Valid() {
		return AgentResolvedResp{}, fmt.Errorf("unknown or missing client %q", req.Client)
	}
	res, err := h.app.Queries.ResolveAgent.Handle(ctx, query.ResolveAgent{Name: req.Agent, Client: client})
	if err != nil {
		return AgentResolvedResp{}, err
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

	return AgentResolvedResp{Name: res.Name, Persona: res.Persona, Model: res.Model}, nil
}

// SubmitPlan adapts the wire frame to the SubmitPlan command: it validates the client, decodes
// the plan against the harness contract, attaches it to the current session, drives the
// application handler, and maps the result back to the wire. No business logic lives here.
func (h *handler) SubmitPlan(ctx context.Context, req SubmitPlanReq) (PlanRecordedResp, error) {
	client := domain.ClientKind(req.Client)
	if !client.Valid() {
		return PlanRecordedResp{}, fmt.Errorf("unknown or missing client %q", req.Client)
	}
	if !h.sessionStarted {
		return PlanRecordedResp{}, fmt.Errorf("no session to attach the plan to")
	}
	var plan domain.OrchestrationPlan
	if err := json.Unmarshal(req.Plan, &plan); err != nil {
		return PlanRecordedResp{}, fmt.Errorf("invalid plan: %w", err)
	}

	ctx = appctx.WithActor(ctx, string(client))
	res, err := h.app.Commands.SubmitPlan.Handle(ctx, command.SubmitPlan{
		SessionID: domain.SessionID(h.sessionID),
		Agent:     req.Agent,
		Plan:      plan,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return PlanRecordedResp{}, err
	}
	h.logger.Info("plan recorded", slog.String("agent", req.Agent), slog.String("plan", string(res.PlanID)), slog.Int("tasks", res.TaskCount))
	return PlanRecordedResp{PlanID: string(res.PlanID), TaskCount: res.TaskCount}, nil
}
