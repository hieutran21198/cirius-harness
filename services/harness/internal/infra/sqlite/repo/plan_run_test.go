package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

func TestPlanRunWriterUpserts(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)

	// A run's plan_id is an FK to plans, so persist a plan first (session "" → NULL).
	plan := samplePlan(t)
	if err := repo.NewPlanWriter(db).Save(ctx, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	planID := plan.Snapshot().ID

	w := repo.NewPlanRunWriter(db)
	rr := repo.NewPlanRunReader(db)

	run, err := domain.NewPlanRun(planID, "", []string{"T1", "T2"}, time.Now())
	if err != nil {
		t.Fatalf("NewPlanRun: %v", err)
	}
	if err = w.Save(ctx, run); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertRows(t, db, "plan_runs", 1)
	assertRows(t, db, "plan_task_runs", 2)

	// Advance the run + a task, then re-save: rows UPDATE in place (no new rows).
	if err = run.SetTaskStatus("T1", domain.TaskRunning, "scanning", time.Now()); err != nil {
		t.Fatalf("SetTaskStatus: %v", err)
	}
	if err = run.SetStatus(domain.PlanDone, time.Now()); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if err = w.Save(ctx, run); err != nil {
		t.Fatalf("re-Save: %v", err)
	}
	assertRows(t, db, "plan_runs", 1)
	assertRows(t, db, "plan_task_runs", 2)

	got, err := rr.FindByPlan(ctx, planID)
	if err != nil {
		t.Fatalf("FindByPlan: %v", err)
	}
	gs := got.Snapshot()
	if gs.Status != domain.PlanDone {
		t.Fatalf("run status = %q, want done", gs.Status)
	}
	t1 := taskRunByRef(gs.Tasks, "T1")
	if t1.Status != domain.TaskRunning || t1.Summary != "scanning" {
		t.Fatalf("T1 = %+v, want running/'scanning'", t1)
	}
}

func TestPlanRunReaderNotFound(t *testing.T) {
	db := newMigratedDB(t)
	if _, err := repo.NewPlanRunReader(db).FindByPlan(context.Background(), "missing"); !errors.Is(err, domain.ErrPlanRunNotFound) {
		t.Fatalf("FindByPlan(missing) err = %v, want ErrPlanRunNotFound", err)
	}
}

func assertRows(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if n != want {
		t.Fatalf("%s rows = %d, want %d", table, n, want)
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
