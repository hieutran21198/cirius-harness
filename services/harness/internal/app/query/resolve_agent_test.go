package query_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/domain"
)

// fakeAgentReader is an in-memory domain.AgentReader keyed by name.
type fakeAgentReader struct {
	byName map[string]domain.Agent
}

func (r fakeAgentReader) FindByName(_ context.Context, name string) (domain.Agent, error) {
	a, ok := r.byName[name]
	if !ok {
		return domain.Agent{}, domain.ErrAgentNotFound
	}
	return a, nil
}

// fakeReadStore implements query.ReadStore over the fake readers. pr is nil for tests that only
// exercise the agent reader.
type fakeReadStore struct {
	ar  domain.AgentReader
	pr  domain.PlanReader
	rr  domain.PlanRunReader
	trr domain.TaskReportReader
}

func (s fakeReadStore) Agents() domain.AgentReader           { return s.ar }
func (s fakeReadStore) Plans() domain.PlanReader             { return s.pr }
func (s fakeReadStore) PlanRuns() domain.PlanRunReader       { return s.rr }
func (s fakeReadStore) TaskReports() domain.TaskReportReader { return s.trr }

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newCouncil(t *testing.T) domain.Agent {
	t.Helper()
	a, err := domain.NewAgent("council", domain.ArchetypeCommunicator, "route", "orchestrator", domain.SourceSystem, nil)
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	return a
}

func TestResolveAgentReturnsPersona(t *testing.T) {
	council := newCouncil(t)
	rs := fakeReadStore{ar: fakeAgentReader{byName: map[string]domain.Agent{"council": council}}}
	h := query.NewResolveAgentHandler(rs, discardLogger())

	res, err := h.Handle(context.Background(), query.ResolveAgent{Name: "council", Client: domain.ClientPi})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// The persona is the harness-owned domain constant, rendered to a system prompt —
	// not a stored value. Assert the resolved prompt matches the registry.
	want, ok := domain.PersonaFor("council")
	if !ok {
		t.Fatal("PersonaFor(council) missing — registry should define council")
	}
	if res.Name != "council" || res.Persona != want.SystemPrompt() {
		t.Fatalf("got %+v, want council persona prompt", res)
	}
	// Model resolution is a later milestone — the query leaves it empty for now.
	if res.Model != "" {
		t.Fatalf("Model = %q, want empty (resolver milestone)", res.Model)
	}
}

func TestResolveAgentUnknown(t *testing.T) {
	rs := fakeReadStore{ar: fakeAgentReader{byName: map[string]domain.Agent{}}}
	h := query.NewResolveAgentHandler(rs, discardLogger())

	_, err := h.Handle(context.Background(), query.ResolveAgent{Name: "nope", Client: domain.ClientPi})
	if !errors.Is(err, domain.ErrAgentNotFound) {
		t.Fatalf("Handle err = %v, want ErrAgentNotFound", err)
	}
}
