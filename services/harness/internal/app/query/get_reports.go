package query

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// GetReports fetches the structured task reports of a plan's run (ADR-0023): the normalized
// envelopes council's decision stage consumes. It resolves the plan's run, then loads every
// report for that run.
type GetReports struct {
	PlanID domain.PlanID
}

// TaskReportView is one task's normalized report in the result: its ref, the agent that produced
// it, and the validated envelope. The raw output is deliberately not surfaced here — it is for
// audit/debug, not for council's consumption.
type TaskReportView struct {
	TaskRef  string
	Agent    string
	Envelope domain.TaskReportEnvelope
}

// GetReportsResult is the run's reports plus the run id they belong to.
type GetReportsResult struct {
	PlanRunID domain.PlanRunID
	Reports   []TaskReportView
}

// GetReportsHandler is the use-case contract for fetching a run's reports.
type GetReportsHandler decorator.QueryHandler[GetReports, GetReportsResult]

type getReportsHandler struct {
	rs ReadStore
}

// NewGetReportsHandler builds the decorated get-reports handler over the read store.
func NewGetReportsHandler(rs ReadStore, logger *slog.Logger) GetReportsHandler {
	if rs == nil {
		panic("query: nil read store")
	}
	return decorator.ApplyQueryDecorators(getReportsHandler{rs: rs}, logger)
}

// Handle resolves the plan's run and returns its task reports as normalized envelopes. A plan with
// no run yet returns an empty result (no error) — there is simply nothing to decide on.
func (h getReportsHandler) Handle(ctx context.Context, q GetReports) (GetReportsResult, error) {
	run, err := h.rs.PlanRuns().FindByPlan(ctx, q.PlanID)
	if errors.Is(err, domain.ErrPlanRunNotFound) {
		return GetReportsResult{}, nil
	}
	if err != nil {
		return GetReportsResult{}, fmt.Errorf("get reports: %w", err)
	}
	runID := run.Snapshot().ID
	reports, err := h.rs.TaskReports().FindByPlanRun(ctx, runID)
	if err != nil {
		return GetReportsResult{}, fmt.Errorf("get reports: %w", err)
	}
	views := make([]TaskReportView, len(reports))
	for i, r := range reports {
		snap := r.Snapshot()
		views[i] = TaskReportView{TaskRef: snap.TaskRef, Agent: snap.Agent, Envelope: snap.Envelope}
	}
	return GetReportsResult{PlanRunID: runID, Reports: views}, nil
}
