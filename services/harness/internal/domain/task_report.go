package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTaskReport is returned for a structurally invalid task report or envelope.
var ErrInvalidTaskReport = errors.New("task report: invalid")

// TaskReportID is a TaskReport's surrogate identity (UUID v7).
type TaskReportID string

// TaskReport is one driven task's structured result (ADR-0023): the validated envelope the
// assigned agent emitted, keyed to the run and the plan-local task ref, with the worker's full raw
// output kept alongside for audit/debug. It is the normalized unit council's decision stage reads.
// Like PlanRun it UPSERTs — a retried task overwrites its earlier report.
type TaskReport struct {
	id        TaskReportID
	planRunID PlanRunID
	taskRef   string
	agent     string
	envelope  TaskReportEnvelope
	raw       string
	createdAt time.Time
	updatedAt time.Time
}

// NewTaskReport builds a report for a task within a run, minting its identity, and validates it
// (and its envelope).
func NewTaskReport(planRunID PlanRunID, taskRef, agent string, env TaskReportEnvelope, raw string, now time.Time) (TaskReport, error) {
	r := TaskReport{
		id:        newID[TaskReportID](),
		planRunID: planRunID,
		taskRef:   taskRef,
		agent:     agent,
		envelope:  env,
		raw:       raw,
		createdAt: now,
		updatedAt: now,
	}
	return r, r.Validate()
}

// RehydrateTaskReport reconstitutes a report from a persisted snapshot and validates it.
func RehydrateTaskReport(snap TaskReportSnapshot) (TaskReport, error) {
	r := TaskReport{
		id: snap.ID, planRunID: snap.PlanRunID, taskRef: snap.TaskRef, agent: snap.Agent,
		envelope: snap.Envelope, raw: snap.Raw, createdAt: snap.CreatedAt, updatedAt: snap.UpdatedAt,
	}
	return r, r.Validate()
}

// Validate checks the report's invariants: ids and ref present, and a well-formed envelope (known
// status and confidence, non-empty summary).
func (r TaskReport) Validate() error {
	if r.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidTaskReport)
	}
	if r.planRunID == "" {
		return fmt.Errorf("%w: plan run id is required", ErrInvalidTaskReport)
	}
	if r.taskRef == "" {
		return fmt.Errorf("%w: task ref is required", ErrInvalidTaskReport)
	}
	if !reportStatuses[r.envelope.Status] {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidTaskReport, r.envelope.Status)
	}
	if !reportConfidences[r.envelope.Confidence] {
		return fmt.Errorf("%w: unknown confidence %q", ErrInvalidTaskReport, r.envelope.Confidence)
	}
	if r.envelope.Summary == "" {
		return fmt.Errorf("%w: summary is required", ErrInvalidTaskReport)
	}
	return nil
}

// TaskReportSnapshot is the persistence grouped view of a TaskReport.
type TaskReportSnapshot struct {
	ID        TaskReportID
	PlanRunID PlanRunID
	TaskRef   string
	Agent     string
	Envelope  TaskReportEnvelope
	Raw       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Snapshot returns the report's full persistence view.
func (r TaskReport) Snapshot() TaskReportSnapshot {
	return TaskReportSnapshot{
		ID: r.id, PlanRunID: r.planRunID, TaskRef: r.taskRef, Agent: r.agent,
		Envelope: r.envelope, Raw: r.raw, CreatedAt: r.createdAt, UpdatedAt: r.updatedAt,
	}
}
