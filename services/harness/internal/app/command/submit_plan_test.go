package command_test

import (
	"context"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain"
)

// fakePlanWriter captures the plan a SubmitPlan handler saves.
type fakePlanWriter struct {
	saved  domain.Plan
	called bool
}

func (w *fakePlanWriter) Save(_ context.Context, p domain.Plan) error {
	w.saved = p
	w.called = true
	return nil
}

// planUoW is a unit of work whose only live writer is the plan writer; DoTx runs the closure
// with itself (no real transaction).
type planUoW struct{ pw *fakePlanWriter }

func (u *planUoW) Models() domain.ModelWriter               { return nil }
func (u *planUoW) Events() domain.EventWriter               { return nil }
func (u *planUoW) Projects() domain.ProjectWriter           { return nil }
func (u *planUoW) Sessions() domain.SessionWriter           { return nil }
func (u *planUoW) Plans() domain.PlanWriter                 { return u.pw }
func (u *planUoW) PlanRuns() domain.PlanRunWriter           { return nil }
func (u *planUoW) TaskReports() domain.TaskReportWriter     { return nil }
func (u *planUoW) PlanDecisions() domain.PlanDecisionWriter { return nil }
func (u *planUoW) PlanReader() domain.PlanReader            { return nil }
func (u *planUoW) PlanRunReader() domain.PlanRunReader      { return nil }
func (u *planUoW) DoTx(ctx context.Context, fn func(context.Context, command.TransactionalUnitOfWork) error) error {
	return fn(ctx, u)
}

func TestSubmitPlanMapsAndSaves(t *testing.T) {
	pw := &fakePlanWriter{}
	h := command.NewSubmitPlanHandler(&planUoW{pw: pw}, discardLogger())

	src := domain.OrchestrationPlan{
		Intent: "implement",
		Goal:   "build explorer",
		Tasks: []domain.PlannedTask{
			{ID: "T1", Category: domain.CategoryExplore, Assignee: domain.Assignee{Agent: "explorer"}, Objective: "map", Wave: 1},
			{ID: "T2", Category: domain.CategoryImplement, Assignee: domain.Assignee{Agent: "implementer"}, Objective: "do", DependsOn: []string{"T1"}, Wave: 2},
		},
		Report: domain.Report{Status: "approved"},
	}
	res, err := h.Handle(context.Background(), command.SubmitPlan{
		SessionID: "sess-1", Agent: "council", Plan: src, CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !pw.called {
		t.Fatal("Save was not called")
	}
	if res.TaskCount != 2 || res.PlanID == "" {
		t.Fatalf("result = %+v, want 2 tasks + non-empty id", res)
	}
	snap := pw.saved.Snapshot()
	if snap.Agent != "council" || snap.SessionID != "sess-1" || snap.Status != domain.PlanApproved {
		t.Fatalf("saved plan header = %+v", snap)
	}
	if snap.ID != res.PlanID {
		t.Fatalf("result id %q != saved id %q", res.PlanID, snap.ID)
	}
}

func TestSubmitPlanRejectsInvalid(t *testing.T) {
	pw := &fakePlanWriter{}
	h := command.NewSubmitPlanHandler(&planUoW{pw: pw}, discardLogger())

	_, err := h.Handle(context.Background(), command.SubmitPlan{
		Agent: "council", Plan: domain.OrchestrationPlan{Intent: "implement"}, CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for a plan with no tasks")
	}
	if pw.called {
		t.Fatal("Save must not be called for an invalid plan")
	}
}
