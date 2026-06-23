package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain"
)

// fakePlanByID is a domain.PlanReader that returns one canned plan (or an error).
type fakePlanByID struct {
	plan domain.Plan
	err  error
}

func (r fakePlanByID) FindByID(_ context.Context, _ domain.PlanID) (domain.Plan, error) {
	return r.plan, r.err
}

func (r fakePlanByID) LatestForSession(_ context.Context, _ domain.SessionID) (domain.Plan, error) {
	return r.plan, r.err
}

// fakeRunStore is an in-memory PlanRunReader + PlanRunWriter sharing one run.
type fakeRunStore struct{ run *domain.PlanRun }

func (s *fakeRunStore) FindByPlan(_ context.Context, _ domain.PlanID) (domain.PlanRun, error) {
	if s.run == nil {
		return domain.PlanRun{}, domain.ErrPlanRunNotFound
	}
	return *s.run, nil
}

func (s *fakeRunStore) Save(_ context.Context, r domain.PlanRun) error {
	cp := r
	s.run = &cp
	return nil
}

// fakeReportStore is an in-memory domain.TaskReportWriter capturing every saved report.
type fakeReportStore struct{ saved []domain.TaskReport }

func (s *fakeReportStore) Save(_ context.Context, r domain.TaskReport) error {
	s.saved = append(s.saved, r)
	return nil
}

// runUoW is a unit of work wired with the plan reader and run store the ReportRun handler needs.
type runUoW struct {
	plans   domain.PlanReader
	runs    *fakeRunStore
	reports *fakeReportStore
}

func (u *runUoW) Models() domain.ModelWriter     { return nil }
func (u *runUoW) Events() domain.EventWriter     { return nil }
func (u *runUoW) Projects() domain.ProjectWriter { return nil }
func (u *runUoW) Sessions() domain.SessionWriter { return nil }
func (u *runUoW) Plans() domain.PlanWriter       { return nil }
func (u *runUoW) PlanRuns() domain.PlanRunWriter { return u.runs }
func (u *runUoW) TaskReports() domain.TaskReportWriter {
	if u.reports == nil {
		return nil
	}
	return u.reports
}
func (u *runUoW) PlanDecisions() domain.PlanDecisionWriter { return nil }
func (u *runUoW) PlanReader() domain.PlanReader            { return u.plans }
func (u *runUoW) PlanRunReader() domain.PlanRunReader      { return u.runs }
func (u *runUoW) DoTx(ctx context.Context, fn func(context.Context, command.TransactionalUnitOfWork) error) error {
	return fn(ctx, u)
}

func twoTaskPlan(t *testing.T) domain.Plan {
	t.Helper()
	src := domain.OrchestrationPlan{
		Intent: "implement",
		Goal:   "g",
		Tasks: []domain.PlannedTask{
			{ID: "T1", Category: domain.CategoryExplore, Assignee: domain.Assignee{Agent: "explorer"}, Objective: "scan", Wave: 1},
			{ID: "T2", Category: domain.CategoryImplement, Assignee: domain.Assignee{Agent: "implementer"}, Objective: "build", DependsOn: []string{"T1"}, Wave: 2},
		},
		Report: domain.Report{Status: "approved"},
	}
	p, err := domain.NewPlan("sess-1", "council", src, time.Now())
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return p
}

func TestReportRunCreatesThenUpdates(t *testing.T) {
	uow := &runUoW{plans: fakePlanByID{plan: twoTaskPlan(t)}, runs: &fakeRunStore{}}
	h := command.NewReportRunHandler(uow, discardLogger())
	ctx := context.Background()
	now := time.Now()

	// First report seeds the run from the plan's refs and sets it driving.
	res, err := h.Handle(ctx, command.ReportRun{PlanID: "plan-1", PlanStatus: domain.PlanDriving, Now: now})
	if err != nil {
		t.Fatalf("first report: %v", err)
	}
	if res.Status != domain.PlanDriving || res.PlanRunID == "" {
		t.Fatalf("result = %+v, want driving + run id", res)
	}
	if uow.runs.run == nil || len(uow.runs.run.Snapshot().Tasks) != 2 {
		t.Fatal("run not seeded with the plan's two tasks")
	}

	// A per-task report updates the existing run (same run id).
	if _, err = h.Handle(ctx, command.ReportRun{PlanID: "plan-1", Task: &command.ReportTask{Ref: "T1", Status: domain.TaskRunning}, Now: now}); err != nil {
		t.Fatalf("task running: %v", err)
	}
	res2, err := h.Handle(ctx, command.ReportRun{PlanID: "plan-1", Task: &command.ReportTask{Ref: "T1", Status: domain.TaskDone, Summary: "scanned"}, Now: now})
	if err != nil {
		t.Fatalf("task done: %v", err)
	}
	if res2.PlanRunID != res.PlanRunID {
		t.Fatalf("run id changed across reports: %q → %q", res.PlanRunID, res2.PlanRunID)
	}
	t1 := taskRunByRef(uow.runs.run.Snapshot().Tasks, "T1")
	if t1.Status != domain.TaskDone || t1.Summary != "scanned" {
		t.Fatalf("T1 = %+v, want done/'scanned'", t1)
	}

	// Finishing the plan is legal from driving.
	if _, err := h.Handle(ctx, command.ReportRun{PlanID: "plan-1", PlanStatus: domain.PlanDone, Now: now}); err != nil {
		t.Fatalf("plan done: %v", err)
	}
	if uow.runs.run.Status() != domain.PlanDone {
		t.Fatalf("run status = %q, want done", uow.runs.run.Status())
	}
}

func TestReportRunStoresTaskReport(t *testing.T) {
	reports := &fakeReportStore{}
	uow := &runUoW{plans: fakePlanByID{plan: twoTaskPlan(t)}, runs: &fakeRunStore{}, reports: reports}
	h := command.NewReportRunHandler(uow, discardLogger())
	ctx := context.Background()
	now := time.Now()

	if _, err := h.Handle(ctx, command.ReportRun{PlanID: "plan-1", Task: &command.ReportTask{Ref: "T1", Status: domain.TaskRunning}, Now: now}); err != nil {
		t.Fatalf("running: %v", err)
	}
	// A terminal report carries the structured envelope; it is persisted alongside the status move.
	env := domain.TaskReportEnvelope{Status: "done", Summary: "scanned the tree", Confidence: "high"}
	if _, err := h.Handle(ctx, command.ReportRun{
		PlanID: "plan-1",
		Task: &command.ReportTask{Ref: "T1", Status: domain.TaskDone, Summary: "scanned",
			Report: &command.TaskReportInput{Agent: "explorer", Envelope: env, Raw: "full raw output"}},
		Now: now,
	}); err != nil {
		t.Fatalf("done with report: %v", err)
	}
	if len(reports.saved) != 1 {
		t.Fatalf("saved %d reports, want 1", len(reports.saved))
	}
	snap := reports.saved[0].Snapshot()
	if snap.TaskRef != "T1" || snap.Agent != "explorer" || snap.Envelope.Summary != "scanned the tree" || snap.Raw != "full raw output" {
		t.Fatalf("report = %+v, want T1/explorer envelope+raw", snap)
	}
	if snap.PlanRunID == "" {
		t.Fatal("report not keyed to the run")
	}
}

func TestReportRunRejectsIllegalTaskTransition(t *testing.T) {
	uow := &runUoW{plans: fakePlanByID{plan: twoTaskPlan(t)}, runs: &fakeRunStore{}}
	h := command.NewReportRunHandler(uow, discardLogger())
	ctx := context.Background()

	// pending → done is not a legal task transition.
	_, err := h.Handle(ctx, command.ReportRun{PlanID: "plan-1", Task: &command.ReportTask{Ref: "T1", Status: domain.TaskDone}, Now: time.Now()})
	if !errors.Is(err, domain.ErrIllegalTransition) {
		t.Fatalf("err = %v, want ErrIllegalTransition", err)
	}
}

func TestReportRunUnknownPlan(t *testing.T) {
	uow := &runUoW{plans: fakePlanByID{err: domain.ErrPlanNotFound}, runs: &fakeRunStore{}}
	h := command.NewReportRunHandler(uow, discardLogger())

	_, err := h.Handle(context.Background(), command.ReportRun{PlanID: "missing", PlanStatus: domain.PlanDriving, Now: time.Now()})
	if !errors.Is(err, domain.ErrPlanNotFound) {
		t.Fatalf("err = %v, want ErrPlanNotFound", err)
	}
}

func taskRunByRef(tasks []domain.TaskRunSnapshot, ref string) domain.TaskRunSnapshot {
	for _, t := range tasks {
		if t.TaskRef == ref {
			return t
		}
	}
	return domain.TaskRunSnapshot{}
}
