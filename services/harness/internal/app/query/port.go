package query

import (
	"harness-workspace/services/harness/internal/domain"
)

// The query side's driven ports, defined where they are consumed (ADR-0013):
// queries read through a ReadStore, which exposes the per-aggregate domain Readers.
// The infra adapter implements these. Reads need no transaction, so — unlike the
// command UnitOfWork — there is no DoTx; each reader autocommits per call.

// ReadStore exposes the readers available to the query side.
type ReadStore interface {
	Agents() domain.AgentReader
	Plans() domain.PlanReader
	PlanRuns() domain.PlanRunReader
	TaskReports() domain.TaskReportReader
}
