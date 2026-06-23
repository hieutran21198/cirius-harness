package repo_test

import (
	"context"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

func TestTaskReportWriterUpsertsAndReads(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)

	// task_reports.plan_run_id is an FK to plan_runs, so persist a plan and a run first.
	plan := samplePlan(t)
	if err := repo.NewPlanWriter(db).Save(ctx, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	run, err := domain.NewPlanRun(plan.Snapshot().ID, "", []string{"T1"}, time.Now())
	if err != nil {
		t.Fatalf("NewPlanRun: %v", err)
	}
	if err = repo.NewPlanRunWriter(db).Save(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	runID := run.Snapshot().ID

	w := repo.NewTaskReportWriter(db)
	rr := repo.NewTaskReportReader(db)

	env := domain.TaskReportEnvelope{Status: "done", Summary: "scanned the tree", Confidence: "high"}
	report, err := domain.NewTaskReport(runID, "T1", "explorer", env, "full raw output", time.Now())
	if err != nil {
		t.Fatalf("NewTaskReport: %v", err)
	}
	if err = w.Save(ctx, report); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertRows(t, db, "task_reports", 1)

	// A retried task overwrites its report in place (UPSERT on plan_run_id+task_ref).
	env2 := domain.TaskReportEnvelope{Status: "done", Summary: "scanned again", Confidence: "medium"}
	report2, err := domain.NewTaskReport(runID, "T1", "explorer", env2, "second raw", time.Now())
	if err != nil {
		t.Fatalf("NewTaskReport 2: %v", err)
	}
	if err = w.Save(ctx, report2); err != nil {
		t.Fatalf("re-Save: %v", err)
	}
	assertRows(t, db, "task_reports", 1)

	got, err := rr.FindByPlanRun(ctx, runID)
	if err != nil {
		t.Fatalf("FindByPlanRun: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d reports, want 1", len(got))
	}
	snap := got[0].Snapshot()
	if snap.TaskRef != "T1" || snap.Agent != "explorer" {
		t.Fatalf("report = %+v, want T1/explorer", snap)
	}
	if snap.Envelope.Summary != "scanned again" || snap.Envelope.Confidence != "medium" || snap.Raw != "second raw" {
		t.Fatalf("envelope not overwritten: %+v / raw=%q", snap.Envelope, snap.Raw)
	}
}

func TestPlanDecisionWriterInserts(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)

	plan := samplePlan(t)
	if err := repo.NewPlanWriter(db).Save(ctx, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	run, err := domain.NewPlanRun(plan.Snapshot().ID, "", []string{"T1"}, time.Now())
	if err != nil {
		t.Fatalf("NewPlanRun: %v", err)
	}
	if err = repo.NewPlanRunWriter(db).Save(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}

	src := domain.CouncilDecision{
		Verdict: "accept", Summary: "all tasks met the definition of done",
		Tasks: []domain.TaskVerdict{{Ref: "T1", Verdict: "accept", Rationale: "scanned cleanly"}},
	}
	d, err := domain.NewPlanDecision(run.Snapshot().ID, src, time.Now())
	if err != nil {
		t.Fatalf("NewPlanDecision: %v", err)
	}
	if err = repo.NewPlanDecisionWriter(db).Save(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertRows(t, db, "council_decisions", 1)
}
