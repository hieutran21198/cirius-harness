// Package pilink is the inbound adapter for the Pi coding client: a
// newline-delimited JSON (NDJSON) request/response loop spoken over stdio
// (see ADR-0008). The Pi extension launches `harness serve`, which calls
// Serve; the extension sends one JSON object per line on stdin and reads one
// JSON object per line on stdout.
//
// Channel discipline (matches Pi's own RPC framing):
//   - stdout is the protocol channel — only JSON messages, one per LF line.
//   - stderr is for logs/diagnostics — never write logs to the protocol writer.
//   - framing is LF-only ("\n"); a trailing "\r" is tolerated on input.
//
// This package is transport only: it decodes messages, dispatches to a Handler,
// and encodes replies. The Handler (which touches persistence) is implemented by
// the composition root in cmd/harness.
package pilink

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

// maxLine bounds a single NDJSON record. Handshake messages are tiny; this guards
// against an unbounded line while staying well above any real message.
const maxLine = 1 << 20 // 1 MiB

// Message types on the wire. Each frame is a JSON object carrying a "type" and an
// optional "id" used by the client to correlate a reply with its request.
const (
	typeHello         = "hello"          // in:  client announces itself
	typePing          = "ping"           // in:  liveness probe
	typeModels        = "models"         // in:  client reports its enabled models
	typeResolveAgent  = "resolve_agent"  // in:  client asks the harness to resolve an agent
	typeSubmitPlan    = "submit_plan"    // in:  client submits an approved council plan to persist
	typeGetPlan       = "get_plan"       // in:  client fetches a persisted plan to drive
	typeReportRun     = "report_run"     // in:  client reports drive progress (plan/task status)
	typeReady         = "ready"          // out: handshake accepted, harness is live
	typePong          = "pong"           // out: reply to ping
	typeModelsSynced  = "models_synced"  // out: catalog sync result
	typeAgentResolved = "agent_resolved" // out: resolved agent (persona, and later model)
	typePlanRecorded  = "plan_recorded"  // out: plan persisted (id + task count)
	typePlanFetched   = "plan"           // out: the fetched plan (contract shape + ids + status)
	typeRunReported   = "run_reported"   // out: drive progress recorded (run id + status)
	typeError         = "error"          // out: a frame could not be handled
)

// envelope is the common header decoded from every inbound frame to route it.
type envelope struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

// HelloReq is the inbound "hello" frame: the client identifying itself.
type HelloReq struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	// CWD is the client's working directory (the Pi session's project root).
	CWD string `json:"cwd,omitempty"`
	// PID is the client process id, for diagnostics.
	PID int `json:"pid,omitempty"`
}

// ReadyResp is the outbound "ready" frame: the harness is live and reachable.
type ReadyResp struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	// SchemaVersion is the applied DB migration version — proof the harness
	// reached its migrated state, not just that the process started.
	SchemaVersion int64 `json:"schemaVersion"`
	// DBPath is the database the harness opened.
	DBPath string `json:"dbPath"`
	// PID is the harness process id.
	PID int `json:"pid"`
}

// PongResp is the outbound reply to a ping.
type PongResp struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

// ModelRef is one model the client offers, by provider and slug (Pi's model id).
type ModelRef struct {
	Provider string `json:"provider"`
	Slug     string `json:"slug"`
}

// ModelsReq is the inbound "models" frame: the client's enabled models, synced
// into the catalog at session start.
type ModelsReq struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	// Client identifies the reporting client (e.g. "pi", "opencode"). Model names are
	// client-specific, so it is part of each entry's catalog identity (ADR-0015) and
	// is required, not just diagnostic.
	Client string `json:"client,omitempty"`
	// Models is the client's enabled (provider, slug) set.
	Models []ModelRef `json:"models"`
}

// ModelsSyncedResp is the outbound reply to a "models" frame: how many refs were
// newly added and the catalog total after the sync.
type ModelsSyncedResp struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Added int    `json:"added"`
	Total int    `json:"total"`
}

// ResolveAgentReq is the inbound "resolve_agent" frame: the client asks the harness
// to resolve an agent (e.g. council for the /council command) so it can govern a turn
// as that agent. Client is the reporting client, for the (later) client-specific model.
type ResolveAgentReq struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Agent  string `json:"agent"`
	Client string `json:"client,omitempty"`
}

// AgentResolvedResp is the outbound reply to a "resolve_agent" frame: the harness-owned
// persona the client runs the turn as. Model is the model to run it on; it is empty
// until the config resolver lands (model governance is a separate milestone, ADR-0016).
type AgentResolvedResp struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Persona string `json:"persona"`
	Model   string `json:"model,omitempty"`
}

// SubmitPlanReq is the inbound "submit_plan" frame: the client submits a council-produced
// orchestration plan (after a human approved it) for the harness to persist. Plan is the raw
// plan JSON, decoded against the harness's plan contract by the handler. Client is the
// reporting client; Agent is the producing agent (council).
type SubmitPlanReq struct {
	Type   string          `json:"type"`
	ID     string          `json:"id,omitempty"`
	Agent  string          `json:"agent"`
	Client string          `json:"client,omitempty"`
	Plan   json.RawMessage `json:"plan"`
}

// PlanRecordedResp is the outbound reply to a "submit_plan" frame: the persisted plan's id and
// how many tasks it holds.
type PlanRecordedResp struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	PlanID    string `json:"planId"`
	TaskCount int    `json:"taskCount"`
}

// GetPlanReq is the inbound "get_plan" frame: the client fetches a persisted plan to drive. An
// empty PlanID means "the latest plan produced in the current session". Client is the reporting
// client.
type GetPlanReq struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	PlanID string `json:"planId,omitempty"`
	Client string `json:"client,omitempty"`
}

// PlanResp is the outbound reply to a "get_plan" frame: the plan in the OrchestrationPlan contract
// shape (the same vocabulary submit_plan uses), its id and current status, and the ref→task-id map
// so the driver can target a task when reporting progress.
type PlanResp struct {
	Type    string            `json:"type"`
	ID      string            `json:"id,omitempty"`
	PlanID  string            `json:"planId"`
	Status  string            `json:"status"`
	Plan    json.RawMessage   `json:"plan"`
	TaskIDs map[string]string `json:"taskIds"`
}

// ReportRunReq is the inbound "report_run" frame: the client records drive progress for a plan —
// an optional plan-level status move (driving→done) and/or an optional per-task status update.
type ReportRunReq struct {
	Type       string `json:"type"`
	ID         string `json:"id,omitempty"`
	Client     string `json:"client,omitempty"`
	PlanID     string `json:"planId"`
	PlanStatus string `json:"planStatus,omitempty"`
	Task       *struct {
		Ref     string `json:"ref"`
		Status  string `json:"status"`
		Summary string `json:"summary,omitempty"`
	} `json:"task,omitempty"`
}

// RunReportedResp is the outbound reply to a "report_run" frame: the run's id and current status.
type RunReportedResp struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	PlanRunID string `json:"planRunId"`
	Status    string `json:"status"`
}

// errorResp is the outbound frame for an unhandled inbound frame.
type errorResp struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message"`
}

// Handler answers the harness-specific frames. Implementations live in the
// composition root (cmd/harness), where persistence is wired.
type Handler interface {
	// Hello builds the ReadyResp for a hello frame (e.g. reads the schema version).
	Hello(ctx context.Context, req HelloReq) (ReadyResp, error)
	// SyncModels upserts the client's reported models into the catalog and reports
	// how many were newly added and the catalog total.
	SyncModels(ctx context.Context, req ModelsReq) (ModelsSyncedResp, error)
	// ResolveAgent resolves the named agent (its persona, and later its model) so the
	// client can govern a turn as that agent.
	ResolveAgent(ctx context.Context, req ResolveAgentReq) (AgentResolvedResp, error)
	// SubmitPlan persists a council-produced orchestration plan the human approved, reporting
	// the stored plan's id and task count.
	SubmitPlan(ctx context.Context, req SubmitPlanReq) (PlanRecordedResp, error)
	// GetPlan fetches a persisted plan (by id, or the latest for the session) so the client can
	// drive it, returning it in the OrchestrationPlan contract shape with its ids and status.
	GetPlan(ctx context.Context, req GetPlanReq) (PlanResp, error)
	// ReportRun records drive progress for a plan (plan-level status and/or a per-task update),
	// returning the run's id and current status.
	ReportRun(ctx context.Context, req ReportRunReq) (RunReportedResp, error)
}

// Serve runs the NDJSON request/response loop until in reaches EOF or ctx is
// cancelled. It reads one JSON frame per LF-delimited line from in, dispatches to
// h, and writes one JSON frame per line to out. Decode/handler errors are reported
// back as an "error" frame and do not stop the loop; only EOF, ctx cancellation,
// or a write/transport failure ends it. Serve writes protocol frames only to out;
// diagnostics go to logger (never to out). A nil logger discards.
func Serve(ctx context.Context, in io.Reader, out io.Writer, h Handler, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLine)

	enc := json.NewEncoder(out)
	// NDJSON: one compact object per line. json.Encoder already appends "\n".

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue // tolerate blank keepalive lines
		}

		var env envelope
		if err := json.Unmarshal(line, &env); err != nil {
			logger.Warn("invalid json frame", slog.Any("err", err))
			if werr := enc.Encode(errorResp{
				Type:    typeError,
				Message: fmt.Sprintf("invalid json: %v", err),
			}); werr != nil {
				return werr
			}
			continue
		}

		logger.Debug("frame received", slog.String("frame", env.Type), slog.String("id", env.ID))
		if err := dispatch(ctx, enc, h, env, line, logger); err != nil {
			return err // write/transport failure — the channel is gone
		}
	}

	if err := scanner.Err(); err != nil {
		// EOF surfaces as a nil scanner error; a real read error propagates.
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// dispatch handles one decoded frame. It returns a non-nil error only on a
// write/transport failure (which ends the loop); handler-level problems are
// encoded as an "error" frame (logged at Warn) and return nil.
func dispatch(ctx context.Context, enc *json.Encoder, h Handler, env envelope, raw []byte, logger *slog.Logger) error {
	replyErr := func(format string, a ...any) error {
		msg := fmt.Sprintf(format, a...)
		logger.Warn("error frame", slog.String("frame", env.Type), slog.String("id", env.ID), slog.String("message", msg))
		return enc.Encode(errorResp{Type: typeError, ID: env.ID, Message: msg})
	}

	switch env.Type {
	case typeHello:
		var req HelloReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid hello: %v", err)
		}
		resp, err := h.Hello(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typeReady
		resp.ID = env.ID
		return enc.Encode(resp)

	case typePing:
		return enc.Encode(PongResp{Type: typePong, ID: env.ID})

	case typeModels:
		var req ModelsReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid models: %v", err)
		}
		resp, err := h.SyncModels(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typeModelsSynced
		resp.ID = env.ID
		return enc.Encode(resp)

	case typeResolveAgent:
		var req ResolveAgentReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid resolve_agent: %v", err)
		}
		resp, err := h.ResolveAgent(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typeAgentResolved
		resp.ID = env.ID
		return enc.Encode(resp)

	case typeSubmitPlan:
		var req SubmitPlanReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid submit_plan: %v", err)
		}
		resp, err := h.SubmitPlan(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typePlanRecorded
		resp.ID = env.ID
		return enc.Encode(resp)

	case typeGetPlan:
		var req GetPlanReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid get_plan: %v", err)
		}
		resp, err := h.GetPlan(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typePlanFetched
		resp.ID = env.ID
		return enc.Encode(resp)

	case typeReportRun:
		var req ReportRunReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return replyErr("invalid report_run: %v", err)
		}
		resp, err := h.ReportRun(ctx, req)
		if err != nil {
			return replyErr("%s", err.Error())
		}
		resp.Type = typeRunReported
		resp.ID = env.ID
		return enc.Encode(resp)

	default:
		return replyErr("unknown type %q", env.Type)
	}
}
