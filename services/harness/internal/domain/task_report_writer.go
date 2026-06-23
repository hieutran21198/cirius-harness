package domain

import "context"

// TaskReportWriter persists a TaskReport. Save UPSERTs on (plan_run_id, task_ref): a retried task
// overwrites its earlier report (ADR-0023). A domain-owned driven port, obtained from a
// UnitOfWork and implemented by the infra adapter.
type TaskReportWriter interface {
	// Save upserts the report (envelope, raw, status, updated_at) for its (run, task ref).
	Save(ctx context.Context, r TaskReport) error
}
