package repo_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
	"harness-workspace/services/harness/internal/infra/sqlite/repo"
)

func TestPlanReaderFindByID(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)

	orig := samplePlan(t)
	want := orig.Snapshot()
	if err := repo.NewPlanWriter(db).Save(ctx, orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.NewPlanReader(db).FindByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	stored := got.Snapshot()

	// Scalars + leaf JSON round-trip (ids are persisted, not regenerated).
	if stored.ID != want.ID || stored.Agent != want.Agent || stored.Intent != want.Intent || stored.Goal != want.Goal {
		t.Fatalf("scalars = %+v, want id/agent/intent/goal of %+v", stored, want)
	}
	if stored.Status != domain.PlanPlanned {
		t.Fatalf("status = %q, want planned", stored.Status)
	}
	if !reflect.DeepEqual(stored.Scope, want.Scope) {
		t.Fatalf("scope = %+v, want %+v", stored.Scope, want.Scope)
	}
	if !reflect.DeepEqual(stored.Report, want.Report) {
		t.Fatalf("report = %+v, want %+v", stored.Report, want.Report)
	}
	if !reflect.DeepEqual(stored.Assumptions, want.Assumptions) {
		t.Fatalf("assumptions = %v, want %v", stored.Assumptions, want.Assumptions)
	}
	if len(stored.Tasks) != 2 || len(stored.Waves) != 2 || len(stored.Risks) != 1 || len(stored.Approvals) != 1 {
		t.Fatalf("child counts = tasks %d / waves %d / risks %d / approvals %d",
			len(stored.Tasks), len(stored.Waves), len(stored.Risks), len(stored.Approvals))
	}
	// Task T2 rehydrates with its assignee+lens, inputs, deps.
	t2 := taskByRef(stored.Tasks, "T2")
	if t2.AssigneeAgent != "planner" || t2.AssigneeLens != "architect" {
		t.Fatalf("T2 assignee = %q/%q, want planner/architect", t2.AssigneeAgent, t2.AssigneeLens)
	}
	if !reflect.DeepEqual(t2.DependsOn, []string{"T1"}) || !reflect.DeepEqual(t2.Inputs, []string{"T1"}) {
		t.Fatalf("T2 deps/inputs = %v/%v, want [T1]/[T1]", t2.DependsOn, t2.Inputs)
	}
	// Wave membership rebuilt from the join table.
	if w2 := waveByNumber(stored.Waves, 2); !reflect.DeepEqual(w2.TaskRefs, []string{"T2"}) {
		t.Fatalf("wave 2 refs = %v, want [T2]", w2.TaskRefs)
	}
	if stored.CreatedAt.IsZero() {
		t.Fatal("CreatedAt did not round-trip")
	}
}

func TestPlanReaderLatestForSession(t *testing.T) {
	ctx := context.Background()
	db := newMigratedDB(t)
	w := repo.NewPlanWriter(db)
	r := repo.NewPlanReader(db)

	seedSession(t, db, "sess-A") // plans.session_id is an FK to sessions
	older := planForSession(t, "sess-A", time.Now().Add(-time.Hour))
	newer := planForSession(t, "sess-A", time.Now())
	if err := w.Save(ctx, older); err != nil {
		t.Fatalf("Save older: %v", err)
	}
	if err := w.Save(ctx, newer); err != nil {
		t.Fatalf("Save newer: %v", err)
	}

	got, err := r.LatestForSession(ctx, "sess-A")
	if err != nil {
		t.Fatalf("LatestForSession: %v", err)
	}
	if got.Snapshot().ID != newer.Snapshot().ID {
		t.Fatalf("got plan %q, want the newer %q", got.Snapshot().ID, newer.Snapshot().ID)
	}

	if _, err := r.LatestForSession(ctx, "sess-none"); !errors.Is(err, domain.ErrPlanNotFound) {
		t.Fatalf("LatestForSession(none) err = %v, want ErrPlanNotFound", err)
	}
}

func TestPlanReaderFindByIDNotFound(t *testing.T) {
	db := newMigratedDB(t)
	if _, err := repo.NewPlanReader(db).FindByID(context.Background(), "missing"); !errors.Is(err, domain.ErrPlanNotFound) {
		t.Fatalf("FindByID(missing) err = %v, want ErrPlanNotFound", err)
	}
}

// seedSession inserts the project + session rows a plan's session_id FK requires.
func seedSession(t *testing.T, db *gorm.DB, id domain.SessionID) {
	t.Helper()
	pid := "proj-" + string(id)
	if err := db.Exec(`INSERT INTO projects (id, name, root_path, kind) VALUES (?, ?, ?, ?)`,
		pid, "name-"+string(id), "/p/"+string(id), "repo").Error; err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := db.Exec(`INSERT INTO sessions (id, project_id, status, created_at) VALUES (?, ?, ?, ?)`,
		string(id), pid, "running", time.Now()).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func planForSession(t *testing.T, session domain.SessionID, at time.Time) domain.Plan {
	t.Helper()
	src := domain.OrchestrationPlan{
		Intent: "implement",
		Goal:   "g",
		Tasks:  []domain.PlannedTask{{ID: "T1", Category: domain.CategoryExplore, Assignee: domain.Assignee{Agent: "explorer"}, Objective: "scan", Wave: 1}},
		Waves:  []domain.Wave{{Wave: 1, Tasks: []string{"T1"}}},
		Report: domain.Report{Status: "planned"},
	}
	p, err := domain.NewPlan(session, "council", src, at)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	return p
}

func taskByRef(tasks []domain.PlanTaskSnapshot, ref string) domain.PlanTaskSnapshot {
	for _, t := range tasks {
		if t.Ref == ref {
			return t
		}
	}
	return domain.PlanTaskSnapshot{}
}

func waveByNumber(waves []domain.PlanWaveSnapshot, n int) domain.PlanWaveSnapshot {
	for _, w := range waves {
		if w.Number == n {
			return w
		}
	}
	return domain.PlanWaveSnapshot{}
}
