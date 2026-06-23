package command

import (
	"context"
	"log/slog"
	"time"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// SubmitDecision records council's post-execution verdict over a plan's run (ADR-0023): it
// resolves the run for the plan, maps the wire contract into a domain.PlanDecision, and persists
// it (append-only) in one transaction. The decision is council's judgement of the drive against
// the plan's definition of done, formed from the run's structured task reports.
type SubmitDecision struct {
	PlanID   domain.PlanID
	Decision domain.CouncilDecision
	Now      time.Time
}

// SubmitDecisionResult reports the persisted decision's id and the run it belongs to.
type SubmitDecisionResult struct {
	DecisionID domain.PlanDecisionID
	PlanRunID  domain.PlanRunID
}

// SubmitDecisionHandler is the use-case contract for submitting a decision.
type SubmitDecisionHandler decorator.CommandHandler[SubmitDecision, SubmitDecisionResult]

type submitDecisionHandler struct {
	uow UnitOfWork
}

// NewSubmitDecisionHandler builds the decorated submit-decision handler.
func NewSubmitDecisionHandler(uow UnitOfWork, logger *slog.Logger) SubmitDecisionHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(submitDecisionHandler{uow: uow}, logger, uow.Events())
}

// Handle resolves the plan's run, maps the contract to a domain.PlanDecision, and saves it in one
// transaction. A plan without a run (never driven) is an error — there is nothing to decide on.
func (h submitDecisionHandler) Handle(ctx context.Context, cmd SubmitDecision) (SubmitDecisionResult, error) {
	var res SubmitDecisionResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		run, err := tx.PlanRunReader().FindByPlan(ctx, cmd.PlanID)
		if err != nil {
			return err
		}
		runID := run.Snapshot().ID
		d, err := domain.NewPlanDecision(runID, cmd.Decision, cmd.Now)
		if err != nil {
			return err
		}
		if err := tx.PlanDecisions().Save(ctx, d); err != nil {
			return err
		}
		res = SubmitDecisionResult{DecisionID: d.Snapshot().ID, PlanRunID: runID}
		return nil
	})
	if err != nil {
		return SubmitDecisionResult{}, err
	}
	return res, nil
}
