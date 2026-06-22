package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidPlanRun is returned for a structurally invalid run or an unknown task ref.
	ErrInvalidPlanRun = errors.New("plan run: invalid")
	// ErrIllegalTransition is returned when a status change is not a legal transition.
	ErrIllegalTransition = errors.New("plan run: illegal status transition")
)

// Run-side surrogate identities (UUID v7).
type (
	// PlanRunID is a PlanRun's surrogate identity.
	PlanRunID string
	// TaskRunID is a TaskRun's surrogate identity.
	TaskRunID string
)

// TaskStatus is where one task is in a drive.
type TaskStatus string

// Task drive states.
const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskDone    TaskStatus = "done"
	TaskFailed  TaskStatus = "failed"
	TaskSkipped TaskStatus = "skipped"
)

// Valid reports whether s is a known task status.
func (s TaskStatus) Valid() bool {
	switch s {
	case TaskPending, TaskRunning, TaskDone, TaskFailed, TaskSkipped:
		return true
	}
	return false
}

// PlanRun is the execution state over an approved Plan (ADR-0021): the drive's status and the
// per-task progress a client-coordinated drive reports. The Plan is the immutable spec; the run
// records what happened — so an approved plan is never rewritten. One live run per plan.
type PlanRun struct {
	id        PlanRunID
	planID    PlanID
	sessionID SessionID
	status    PlanStatus
	createdAt time.Time
	updatedAt time.Time
	tasks     []TaskRun
}

// TaskRun is one task's progress within a run.
type TaskRun struct {
	id        TaskRunID
	taskRef   string
	status    TaskStatus
	summary   string
	updatedAt time.Time
}

// NewPlanRun starts a run for a plan, seeding one pending TaskRun per task ref, in PlanDriving.
func NewPlanRun(planID PlanID, sessionID SessionID, taskRefs []string, now time.Time) (PlanRun, error) {
	tasks := make([]TaskRun, len(taskRefs))
	for i, ref := range taskRefs {
		tasks[i] = TaskRun{id: newID[TaskRunID](), taskRef: ref, status: TaskPending, updatedAt: now}
	}
	r := PlanRun{
		id:        newID[PlanRunID](),
		planID:    planID,
		sessionID: sessionID,
		status:    PlanDriving,
		createdAt: now,
		updatedAt: now,
		tasks:     tasks,
	}
	return r, r.Validate()
}

// RehydratePlanRun reconstitutes a run from a persisted snapshot and validates it.
func RehydratePlanRun(snap PlanRunSnapshot) (PlanRun, error) {
	tasks := make([]TaskRun, len(snap.Tasks))
	for i, t := range snap.Tasks {
		tasks[i] = TaskRun{id: t.ID, taskRef: t.TaskRef, status: t.Status, summary: t.Summary, updatedAt: t.UpdatedAt}
	}
	r := PlanRun{
		id: snap.ID, planID: snap.PlanID, sessionID: snap.SessionID, status: snap.Status,
		createdAt: snap.CreatedAt, updatedAt: snap.UpdatedAt, tasks: tasks,
	}
	return r, r.Validate()
}

// PlanID reports the plan this run drives.
func (r PlanRun) PlanID() PlanID { return r.planID }

// Status reports the run's current status.
func (r PlanRun) Status() PlanStatus { return r.status }

// SetStatus moves the run to next if the transition is legal (idempotent on the current status).
func (r *PlanRun) SetStatus(next PlanStatus, now time.Time) error {
	if !next.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidPlanRun, next)
	}
	if !legalPlanTransition(r.status, next) {
		return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, r.status, next)
	}
	r.status = next
	r.updatedAt = now
	return nil
}

// SetTaskStatus moves the task with the given ref to next, recording summary, if the transition
// is legal. An unknown ref is ErrInvalidPlanRun. A non-empty summary overwrites; an empty one is
// left unchanged (so a "running" report does not clear a later "done" summary out of order).
func (r *PlanRun) SetTaskStatus(ref string, next TaskStatus, summary string, now time.Time) error {
	if !next.Valid() {
		return fmt.Errorf("%w: unknown task status %q", ErrInvalidPlanRun, next)
	}
	for i := range r.tasks {
		if r.tasks[i].taskRef != ref {
			continue
		}
		if !legalTaskTransition(r.tasks[i].status, next) {
			return fmt.Errorf("%w: task %q %s → %s", ErrIllegalTransition, ref, r.tasks[i].status, next)
		}
		r.tasks[i].status = next
		if summary != "" {
			r.tasks[i].summary = summary
		}
		r.tasks[i].updatedAt = now
		r.updatedAt = now
		return nil
	}
	return fmt.Errorf("%w: unknown task ref %q", ErrInvalidPlanRun, ref)
}

// legalPlanTransition reports whether the run-level status move from→to is allowed (X→X is an
// idempotent re-report).
func legalPlanTransition(from, to PlanStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case PlanPlanned, PlanApproved:
		return to == PlanDriving || to == PlanCancelled
	case PlanDriving:
		return to == PlanDone || to == PlanCancelled
	}
	return false
}

// legalTaskTransition reports whether the task status move from→to is allowed (X→X idempotent;
// failed→running is a retry).
func legalTaskTransition(from, to TaskStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case TaskPending:
		return to == TaskRunning || to == TaskSkipped
	case TaskRunning:
		return to == TaskDone || to == TaskFailed || to == TaskSkipped
	case TaskFailed:
		return to == TaskRunning
	}
	return false
}

// Validate checks the run's invariants: ids present, status valid, unique non-empty task refs
// with valid statuses.
func (r PlanRun) Validate() error {
	if r.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidPlanRun)
	}
	if r.planID == "" {
		return fmt.Errorf("%w: plan id is required", ErrInvalidPlanRun)
	}
	if !r.status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidPlanRun, r.status)
	}
	refs := make(map[string]bool, len(r.tasks))
	for _, t := range r.tasks {
		if t.taskRef == "" {
			return fmt.Errorf("%w: task ref is required", ErrInvalidPlanRun)
		}
		if refs[t.taskRef] {
			return fmt.Errorf("%w: duplicate task ref %q", ErrInvalidPlanRun, t.taskRef)
		}
		refs[t.taskRef] = true
		if !t.status.Valid() {
			return fmt.Errorf("%w: task %q unknown status %q", ErrInvalidPlanRun, t.taskRef, t.status)
		}
	}
	return nil
}

// PlanRunSnapshot is the persistence grouped view of a PlanRun and its task runs.
type PlanRunSnapshot struct {
	ID        PlanRunID
	PlanID    PlanID
	SessionID SessionID
	Status    PlanStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Tasks     []TaskRunSnapshot
}

// TaskRunSnapshot is the persistence view of a TaskRun.
type TaskRunSnapshot struct {
	ID        TaskRunID
	TaskRef   string
	Status    TaskStatus
	Summary   string
	UpdatedAt time.Time
}

// Snapshot returns the run's full persistence view, including every task run.
func (r PlanRun) Snapshot() PlanRunSnapshot {
	tasks := make([]TaskRunSnapshot, len(r.tasks))
	for i, t := range r.tasks {
		tasks[i] = TaskRunSnapshot{ID: t.id, TaskRef: t.taskRef, Status: t.status, Summary: t.summary, UpdatedAt: t.updatedAt}
	}
	return PlanRunSnapshot{
		ID: r.id, PlanID: r.planID, SessionID: r.sessionID, Status: r.status,
		CreatedAt: r.createdAt, UpdatedAt: r.updatedAt, Tasks: tasks,
	}
}
