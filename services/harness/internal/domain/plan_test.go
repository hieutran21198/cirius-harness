package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// validPlan is a small but complete contract plan used across the plan tests.
func validPlan() OrchestrationPlan {
	return OrchestrationPlan{
		Intent:      "implement",
		Goal:        "build explorer",
		Scope:       Scope{Primary: []string{"domain"}, OutOfScope: []string{"schema"}},
		Assumptions: []string{"explorer already exists"},
		Risks:       []Risk{{Level: "medium", Description: "ambiguity"}},
		Tasks: []PlannedTask{
			{ID: "T1", Category: CategoryExplore, Assignee: Assignee{Agent: "explorer"}, Objective: "map", Wave: 1, DoD: []string{"listed"}, Gate: "advisory", RiskLevel: "low"},
			{ID: "T2", Category: CategoryPlan, Assignee: Assignee{Agent: "planner", Lens: "architect"}, Objective: "design", Inputs: []string{"T1"}, DependsOn: []string{"T1"}, Wave: 2, DoD: []string{"plan"}, Gate: "validating", RiskLevel: "medium"},
		},
		Approvals: []Approval{{Type: "human-confirmation", RequiredBefore: "T3", Reason: "ambiguous", Question: "expose it?"}},
		Waves:     []Wave{{Wave: 1, Tasks: []string{"T1"}}, {Wave: 2, Tasks: []string{"T2"}}},
		Report:    Report{Status: "planned", Summary: "ok", DefinitionOfDone: []string{"done"}},
	}
}

func TestNewPlanValidAndSnapshotRoundTrips(t *testing.T) {
	t.Parallel()
	p, err := NewPlan("sess-1", "council", validPlan(), time.Now())
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	if p.TaskCount() != 2 {
		t.Fatalf("TaskCount = %d, want 2", p.TaskCount())
	}
	snap := p.Snapshot()
	if snap.Agent != "council" || snap.SessionID != "sess-1" || snap.Status != PlanPlanned {
		t.Fatalf("snapshot header = %+v", snap)
	}
	if len(snap.Tasks) != 2 || len(snap.Risks) != 1 || len(snap.Approvals) != 1 || len(snap.Waves) != 2 {
		t.Fatalf("child counts = tasks %d risks %d approvals %d waves %d", len(snap.Tasks), len(snap.Risks), len(snap.Approvals), len(snap.Waves))
	}
	if snap.Tasks[0].ID == "" || snap.Tasks[0].Ref != "T1" {
		t.Fatalf("task 0 = %+v (want minted id + ref T1)", snap.Tasks[0])
	}
	// Rehydrate from the snapshot and confirm it validates and re-snapshots identically.
	re, err := RehydratePlan(snap)
	if err != nil {
		t.Fatalf("RehydratePlan: %v", err)
	}
	if got := re.Snapshot(); got.ID != snap.ID || len(got.Tasks) != len(snap.Tasks) || got.Tasks[1].AssigneeLens != "architect" {
		t.Fatalf("rehydrated snapshot drifted: %+v", got)
	}
}

func TestNewPlanRejectsBadDAG(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*OrchestrationPlan){
		"empty intent":        func(p *OrchestrationPlan) { p.Intent = "" },
		"no tasks":            func(p *OrchestrationPlan) { p.Tasks = nil },
		"duplicate ref":       func(p *OrchestrationPlan) { p.Tasks[1].ID = "T1" },
		"dangling depends_on": func(p *OrchestrationPlan) { p.Tasks[1].DependsOn = []string{"T9"} },
		"wave unknown task":   func(p *OrchestrationPlan) { p.Waves[0].Tasks = []string{"T9"} },
		"empty ref":           func(p *OrchestrationPlan) { p.Tasks[0].ID = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src := validPlan()
			mutate(&src)
			if _, err := NewPlan("", "council", src, time.Now()); !errors.Is(err, ErrInvalidPlan) {
				t.Fatalf("NewPlan(%s) error = %v, want ErrInvalidPlan", name, err)
			}
		})
	}
}

func TestPlanStatusFromReport(t *testing.T) {
	t.Parallel()
	for status, want := range map[string]PlanStatus{
		"planned":  PlanPlanned,
		"approved": PlanApproved,
		"DONE":     PlanDone,
		"":         PlanPlanned,
		"garbage":  PlanPlanned,
	} {
		src := validPlan()
		src.Report.Status = status
		p, err := NewPlan("", "council", src, time.Now())
		if err != nil {
			t.Fatalf("NewPlan(status=%q): %v", status, err)
		}
		if got := p.Snapshot().Status; got != want {
			t.Fatalf("status %q → %q, want %q", status, got, want)
		}
	}
}

func TestAssigneeLenientDecode(t *testing.T) {
	t.Parallel()
	var bare Assignee
	if err := json.Unmarshal([]byte(`"explorer"`), &bare); err != nil {
		t.Fatalf("decode bare string: %v", err)
	}
	if bare.Agent != "explorer" || bare.Lens != "" {
		t.Fatalf("bare assignee = %+v", bare)
	}
	var obj Assignee
	if err := json.Unmarshal([]byte(`{"agent":"planner","lens":"architect"}`), &obj); err != nil {
		t.Fatalf("decode object: %v", err)
	}
	if obj.Agent != "planner" || obj.Lens != "architect" {
		t.Fatalf("object assignee = %+v", obj)
	}
	// A whole task with a bare-string assignee decodes via the same path.
	var task PlannedTask
	if err := json.Unmarshal([]byte(`{"id":"T1","assignee":"reviewer"}`), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if task.Assignee.Agent != "reviewer" {
		t.Fatalf("task assignee = %+v", task.Assignee)
	}
}
