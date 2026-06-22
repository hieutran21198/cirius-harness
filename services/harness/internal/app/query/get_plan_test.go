package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/app/query"
	"harness-workspace/services/harness/internal/domain"
)

// fakePlanReader is an in-memory domain.PlanReader keyed by plan id.
type fakePlanReader struct {
	byID map[domain.PlanID]domain.Plan
}

func (r fakePlanReader) FindByID(_ context.Context, id domain.PlanID) (domain.Plan, error) {
	p, ok := r.byID[id]
	if !ok {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	return p, nil
}

func (r fakePlanReader) LatestForSession(_ context.Context, _ domain.SessionID) (domain.Plan, error) {
	for _, p := range r.byID {
		return p, nil // one plan in the fake; order is irrelevant for the test
	}
	return domain.Plan{}, domain.ErrPlanNotFound
}

func samplePlan(t *testing.T) domain.Plan {
	t.Helper()
	src := domain.OrchestrationPlan{
		Intent: "implement",
		Goal:   "ship the thing",
		Tasks: []domain.PlannedTask{
			{ID: "T1", Category: domain.CategoryExplore, Assignee: domain.Assignee{Agent: "explorer"}, Objective: "scan", Wave: 1},
			{ID: "T2", Category: domain.CategoryImplement, Assignee: domain.Assignee{Agent: "implementer", Lens: "tester"}, Objective: "build", DependsOn: []string{"T1"}, Wave: 2},
		},
		Waves: []domain.Wave{
			{Wave: 1, Tasks: []string{"T1"}},
			{Wave: 2, Tasks: []string{"T2"}},
		},
		Report: domain.Report{Status: "approved"},
	}
	p, err := domain.NewPlan("sess-1", "council", src, time.Now())
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return p
}

func TestGetPlanReturnsContractShape(t *testing.T) {
	p := samplePlan(t)
	snap := p.Snapshot()
	rs := fakeReadStore{pr: fakePlanReader{byID: map[domain.PlanID]domain.Plan{snap.ID: p}}}
	h := query.NewGetPlanHandler(rs, discardLogger())

	res, err := h.Handle(context.Background(), query.GetPlan{PlanID: snap.ID})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if res.PlanID != snap.ID || res.Status != domain.PlanApproved {
		t.Fatalf("got id=%q status=%q, want %q/approved", res.PlanID, res.Status, snap.ID)
	}
	if len(res.Plan.Tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(res.Plan.Tasks))
	}
	// The contract shape carries refs, assignee+lens, deps, and the wave number per task.
	t2 := res.Plan.Tasks[1]
	if t2.ID != "T2" || t2.Assignee.Agent != "implementer" || t2.Assignee.Lens != "tester" {
		t.Fatalf("T2 assignee = %+v, want implementer/tester", t2.Assignee)
	}
	if len(t2.DependsOn) != 1 || t2.DependsOn[0] != "T1" || t2.Wave != 2 {
		t.Fatalf("T2 deps/wave = %v/%d, want [T1]/2", t2.DependsOn, t2.Wave)
	}
	// TaskIDByRef maps every ref to its minted id.
	if id, ok := res.TaskIDByRef["T1"]; !ok || id == "" {
		t.Fatalf("TaskIDByRef[T1] = %q, want a minted id", id)
	}
	if len(res.TaskIDByRef) != 2 {
		t.Fatalf("TaskIDByRef has %d entries, want 2", len(res.TaskIDByRef))
	}
}

func TestGetPlanNotFound(t *testing.T) {
	rs := fakeReadStore{pr: fakePlanReader{byID: map[domain.PlanID]domain.Plan{}}}
	h := query.NewGetPlanHandler(rs, discardLogger())

	_, err := h.Handle(context.Background(), query.GetPlan{PlanID: "missing"})
	if !errors.Is(err, domain.ErrPlanNotFound) {
		t.Fatalf("Handle err = %v, want ErrPlanNotFound", err)
	}
}
