package command

import (
	"context"
	"log/slog"
	"time"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// SubmitPlan records a council-produced orchestration plan after a human has approved it
// (ADR-0019): it maps the wire contract into a domain.Plan and persists it (and all its
// children) in one transaction. Idempotent on the resulting plan id.
type SubmitPlan struct {
	SessionID domain.SessionID // the session the plan was produced in ("" if none)
	Agent     string           // the agent that produced it (council)
	Plan      domain.OrchestrationPlan
	CreatedAt time.Time
}

// SubmitPlanResult reports the persisted plan's id and how many tasks it holds.
type SubmitPlanResult struct {
	PlanID    domain.PlanID
	TaskCount int
}

// SubmitPlanHandler is the use-case contract for submitting a plan.
type SubmitPlanHandler decorator.CommandHandler[SubmitPlan, SubmitPlanResult]

type submitPlanHandler struct {
	uow UnitOfWork
}

// NewSubmitPlanHandler builds the decorated submit-plan handler.
func NewSubmitPlanHandler(uow UnitOfWork, logger *slog.Logger) SubmitPlanHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(submitPlanHandler{uow: uow}, logger, uow.Events())
}

// Handle maps the contract to a domain.Plan and saves it in one transaction.
func (h submitPlanHandler) Handle(ctx context.Context, cmd SubmitPlan) (SubmitPlanResult, error) {
	var res SubmitPlanResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		p, err := domain.NewPlan(cmd.SessionID, cmd.Agent, cmd.Plan, cmd.CreatedAt)
		if err != nil {
			return err
		}
		if err := tx.Plans().Save(ctx, p); err != nil {
			return err
		}
		snap := p.Snapshot()
		res = SubmitPlanResult{PlanID: snap.ID, TaskCount: p.TaskCount()}
		return nil
	})
	if err != nil {
		return SubmitPlanResult{}, err
	}
	return res, nil
}
