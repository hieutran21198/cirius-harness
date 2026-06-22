// Package query holds the application's read-side use cases. Each is a QueryHandler:
// a plain query struct (the request), a concrete handler over the driven ports it
// needs (a ReadStore — see port.go), and a constructor that applies the cross-cutting
// decorators. The Handle method is pure read logic (ADR-0012).
package query

import (
	"context"
	"fmt"
	"log/slog"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// ResolveAgent is the query to resolve an agent the client wants to run (e.g. the
// /council command resolving the council role). Client is the active reporting client;
// it is accepted now so the resolver can bind a client-specific model later (ADR-0015),
// though this query does not resolve a model yet.
type ResolveAgent struct {
	Name   string
	Client domain.ClientKind
}

// ResolveAgentResult is the resolved agent the client governs a turn with: the
// harness-owned persona to run as, and (later) the model to run it on. Model is empty
// until the config resolver lands — model governance is a separate milestone (ADR-0016).
type ResolveAgentResult struct {
	AgentID domain.AgentID
	Name    string
	Persona string
	Model   string
}

// ResolveAgentHandler is the use-case contract for the agent-resolve query.
type ResolveAgentHandler decorator.QueryHandler[ResolveAgent, ResolveAgentResult]

type resolveAgentHandler struct {
	rs ReadStore
}

// NewResolveAgentHandler builds the decorated agent-resolve handler over the read store.
func NewResolveAgentHandler(rs ReadStore, logger *slog.Logger) ResolveAgentHandler {
	if rs == nil {
		panic("query: nil read store")
	}
	return decorator.ApplyQueryDecorators(resolveAgentHandler{rs: rs}, logger)
}

// Handle confirms the agent exists and is enabled (governance: the harness does not
// serve a persona for an unknown or disabled role), then attaches the harness-owned
// persona resolved by name from the domain registry. The persona is code, not a stored
// column (ADR-0016); most roles have none and resolve to an empty persona. Model is left
// empty: resolving the agent's client-specific model against the synced catalog is the
// config-resolver milestone (ADR-0016); the wire already carries the field.
func (h resolveAgentHandler) Handle(ctx context.Context, q ResolveAgent) (ResolveAgentResult, error) {
	agent, err := h.rs.Agents().FindByName(ctx, q.Name)
	if err != nil {
		return ResolveAgentResult{}, fmt.Errorf("resolve agent %q: %w", q.Name, err)
	}
	snap := agent.Snapshot()
	res := ResolveAgentResult{AgentID: snap.ID, Name: snap.Name}
	if persona, ok := domain.PersonaFor(snap.Name); ok {
		res.Persona = persona.SystemPrompt()
	}
	return res, nil
}
