package repo_test

import (
	"context"
	"testing"
	"time"

	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

func samplePlan(t *testing.T) domain.Plan {
	t.Helper()
	src := domain.OrchestrationPlan{
		Intent:      "implement",
		Goal:        "build explorer",
		Scope:       domain.Scope{Primary: []string{"domain"}, OutOfScope: []string{"schema"}},
		Assumptions: []string{"explorer already exists"},
		Risks:       []domain.Risk{{Level: "medium", Description: "ambiguity"}},
		Tasks: []domain.PlannedTask{
			{ID: "T1", Category: domain.CategoryExplore, Assignee: domain.Assignee{Agent: "explorer"}, Objective: "map", Wave: 1, DoD: []string{"listed"}, Gate: "advisory", RiskLevel: "low"},
			{ID: "T2", Category: domain.CategoryPlan, Assignee: domain.Assignee{Agent: "planner", Lens: "architect"}, Objective: "design", Inputs: []string{"T1"}, DependsOn: []string{"T1"}, Wave: 2, DoD: []string{"plan"}, Gate: "validating", RiskLevel: "medium"},
		},
		Approvals: []domain.Approval{{Type: "human-confirmation", RequiredBefore: "T3", Reason: "ambiguous", Question: "expose it?"}},
		Waves:     []domain.Wave{{Wave: 1, Tasks: []string{"T1"}}, {Wave: 2, Tasks: []string{"T2"}}},
		Report:    domain.Report{Status: "planned", Summary: "ok", DefinitionOfDone: []string{"done"}},
	}
	// Empty session id → stored as NULL, so the test needs no session row.
	p, err := domain.NewPlan("", "council", src, time.Now())
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return p
}

func TestPlanWriterSave(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)
	w := repo.NewPlanWriter(db)
	p := samplePlan(t)

	if err := w.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	for table, want := range map[string]int64{
		"plans": 1, "plan_tasks": 2, "plan_risks": 1, "plan_approvals": 1, "plan_waves": 2, "plan_wave_tasks": 2,
	} {
		var n int64
		if err := db.WithContext(ctx).Table(table).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != want {
			t.Fatalf("%s rows = %d, want %d", table, n, want)
		}
	}

	// Re-saving the same plan is an idempotent no-op (no duplicate rows).
	if err := w.Save(ctx, p); err != nil {
		t.Fatalf("re-Save: %v", err)
	}
	var plans, tasks int64
	_ = db.WithContext(ctx).Table("plans").Count(&plans).Error
	_ = db.WithContext(ctx).Table("plan_tasks").Count(&tasks).Error
	if plans != 1 || tasks != 2 {
		t.Fatalf("after re-save: plans=%d tasks=%d, want 1/2", plans, tasks)
	}
}
