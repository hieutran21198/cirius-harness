package domain

import "context"

// TaskReportReader reads the structured reports of a run's tasks. A domain-owned driven port: the
// query that feeds council's decision stage loads every report for a run (ADR-0023).
type TaskReportReader interface {
	// FindByPlanRun returns the reports for the given run, ordered by task ref (empty if none).
	FindByPlanRun(ctx context.Context, planRunID PlanRunID) ([]TaskReport, error)
}
