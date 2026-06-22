package query

import (
	"context"
	"fmt"
	"log/slog"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// GetPlan fetches a persisted plan so a client can drive it. PlanID selects a specific plan; when
// empty, the latest plan produced in SessionID is returned (the driver knows the session but not
// always the id).
type GetPlan struct {
	PlanID    domain.PlanID
	SessionID domain.SessionID
}

// GetPlanResult is the resolved plan in the wire contract shape (OrchestrationPlan — the same
// vocabulary submit_plan uses), plus its id, current status, and the ref→task-id map so the
// driver can target a task when reporting progress.
type GetPlanResult struct {
	PlanID      domain.PlanID
	SessionID   domain.SessionID
	Status      domain.PlanStatus
	Plan        domain.OrchestrationPlan
	TaskIDByRef map[string]domain.PlanTaskID
}

// GetPlanHandler is the use-case contract for fetching a plan.
type GetPlanHandler decorator.QueryHandler[GetPlan, GetPlanResult]

type getPlanHandler struct {
	rs ReadStore
}

// NewGetPlanHandler builds the decorated get-plan handler over the read store.
func NewGetPlanHandler(rs ReadStore, logger *slog.Logger) GetPlanHandler {
	if rs == nil {
		panic("query: nil read store")
	}
	return decorator.ApplyQueryDecorators(getPlanHandler{rs: rs}, logger)
}

// Handle fetches the plan (by id, or the latest for the session) and maps the persisted snapshot
// back to the OrchestrationPlan contract — the inverse of NewPlan's mapping — so the client gets
// the plan in the same shape it submitted, with the minted task ids alongside.
func (h getPlanHandler) Handle(ctx context.Context, q GetPlan) (GetPlanResult, error) {
	var (
		p   domain.Plan
		err error
	)
	if q.PlanID != "" {
		p, err = h.rs.Plans().FindByID(ctx, q.PlanID)
	} else {
		p, err = h.rs.Plans().LatestForSession(ctx, q.SessionID)
	}
	if err != nil {
		return GetPlanResult{}, fmt.Errorf("get plan: %w", err)
	}
	snap := p.Snapshot()

	waveByRef := make(map[string]int)
	for _, w := range snap.Waves {
		for _, ref := range w.TaskRefs {
			waveByRef[ref] = w.Number
		}
	}
	tasks := make([]domain.PlannedTask, len(snap.Tasks))
	taskIDByRef := make(map[string]domain.PlanTaskID, len(snap.Tasks))
	for i, t := range snap.Tasks {
		tasks[i] = domain.PlannedTask{
			ID:             t.Ref,
			Category:       t.Category,
			Assignee:       domain.Assignee{Agent: t.AssigneeAgent, Lens: t.AssigneeLens},
			Objective:      t.Objective,
			Inputs:         t.Inputs,
			ExpectedOutput: t.ExpectedOutput,
			DependsOn:      t.DependsOn,
			Wave:           waveByRef[t.Ref],
			DoD:            t.DoD,
			Gate:           t.Gate,
			RiskLevel:      t.RiskLevel,
		}
		taskIDByRef[t.Ref] = t.ID
	}
	risks := make([]domain.Risk, len(snap.Risks))
	for i, r := range snap.Risks {
		risks[i] = domain.Risk{Level: r.Level, Description: r.Description}
	}
	approvals := make([]domain.Approval, len(snap.Approvals))
	for i, a := range snap.Approvals {
		approvals[i] = domain.Approval{Type: a.Kind, RequiredBefore: a.RequiredBefore, Reason: a.Reason, Question: a.Question}
	}
	waves := make([]domain.Wave, len(snap.Waves))
	for i, w := range snap.Waves {
		waves[i] = domain.Wave{Wave: w.Number, Tasks: w.TaskRefs}
	}

	plan := domain.OrchestrationPlan{
		Intent:      snap.Intent,
		Goal:        snap.Goal,
		Scope:       snap.Scope,
		Assumptions: snap.Assumptions,
		Risks:       risks,
		Tasks:       tasks,
		Approvals:   approvals,
		Waves:       waves,
		Report:      snap.Report,
	}
	return GetPlanResult{
		PlanID:      snap.ID,
		SessionID:   snap.SessionID,
		Status:      snap.Status,
		Plan:        plan,
		TaskIDByRef: taskIDByRef,
	}, nil
}
