package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewPlanRunSeedsPendingTasks(t *testing.T) {
	now := time.Now()
	r, err := NewPlanRun("plan-1", "sess-1", []string{"T1", "T2"}, now)
	if err != nil {
		t.Fatalf("NewPlanRun: %v", err)
	}
	if r.Status() != PlanDriving {
		t.Fatalf("status = %q, want driving", r.Status())
	}
	snap := r.Snapshot()
	if len(snap.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(snap.Tasks))
	}
	for _, tr := range snap.Tasks {
		if tr.Status != TaskPending {
			t.Fatalf("task %q status = %q, want pending", tr.TaskRef, tr.Status)
		}
		if tr.ID == "" {
			t.Fatalf("task %q run id not minted", tr.TaskRef)
		}
	}
}

func TestPlanRunStatusTransitions(t *testing.T) {
	now := time.Now()
	r, _ := NewPlanRun("plan-1", "", []string{"T1"}, now)

	if err := r.SetStatus(PlanDone, now); err != nil {
		t.Fatalf("driving→done: %v", err)
	}
	if err := r.SetStatus(PlanDriving, now); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("done→driving err = %v, want ErrIllegalTransition", err)
	}
	if err := r.SetStatus(PlanDone, now); err != nil {
		t.Fatalf("done→done (idempotent): %v", err)
	}
	if err := r.SetStatus("bogus", now); !errors.Is(err, ErrInvalidPlanRun) {
		t.Fatalf("bogus status err = %v, want ErrInvalidPlanRun", err)
	}
}

func TestPlanRunTaskTransitions(t *testing.T) {
	now := time.Now()
	r, _ := NewPlanRun("plan-1", "", []string{"T1"}, now)

	if err := r.SetTaskStatus("T1", TaskRunning, "", now); err != nil {
		t.Fatalf("pending→running: %v", err)
	}
	if err := r.SetTaskStatus("T1", TaskDone, "did it", now); err != nil {
		t.Fatalf("running→done: %v", err)
	}
	if err := r.SetTaskStatus("T1", TaskRunning, "", now); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("done→running err = %v, want ErrIllegalTransition", err)
	}
	if err := r.SetTaskStatus("T9", TaskRunning, "", now); !errors.Is(err, ErrInvalidPlanRun) {
		t.Fatalf("unknown ref err = %v, want ErrInvalidPlanRun", err)
	}
	if snap := r.Snapshot(); snap.Tasks[0].Summary != "did it" || snap.Tasks[0].Status != TaskDone {
		t.Fatalf("T1 = %+v, want done/'did it'", snap.Tasks[0])
	}
}

func TestPlanRunSnapshotRoundTrip(t *testing.T) {
	now := time.Now()
	r, _ := NewPlanRun("plan-1", "sess-1", []string{"T1", "T2"}, now)
	_ = r.SetTaskStatus("T1", TaskRunning, "scanning", now)
	snap := r.Snapshot()

	got, err := RehydratePlanRun(snap)
	if err != nil {
		t.Fatalf("RehydratePlanRun: %v", err)
	}
	if !reflect.DeepEqual(got.Snapshot(), snap) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got.Snapshot(), snap)
	}
}
