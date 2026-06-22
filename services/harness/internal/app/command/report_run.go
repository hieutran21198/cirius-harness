package command

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// ReportTask is an optional per-task progress update within a ReportRun.
type ReportTask struct {
	Ref     string
	Status  domain.TaskStatus
	Summary string
}

// ReportRun records drive progress for a plan (ADR-0021): an optional plan-level status move and
// an optional per-task status update, persisted on a PlanRun. The plan itself stays immutable.
// The run is created on first report (seeded from the plan's task refs) and updated thereafter.
type ReportRun struct {
	PlanID     domain.PlanID
	SessionID  domain.SessionID  // attached to the run when it is first created
	PlanStatus domain.PlanStatus // "" leaves the run status unchanged
	Task       *ReportTask       // nil for a plan-status-only report
	Now        time.Time
}

// ReportRunResult reports the run's id and current status.
type ReportRunResult struct {
	PlanRunID domain.PlanRunID
	Status    domain.PlanStatus
}

// ReportRunHandler is the use-case contract for reporting drive progress.
type ReportRunHandler decorator.CommandHandler[ReportRun, ReportRunResult]

type reportRunHandler struct {
	uow UnitOfWork
}

// NewReportRunHandler builds the decorated report-run handler.
func NewReportRunHandler(uow UnitOfWork, logger *slog.Logger) ReportRunHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(reportRunHandler{uow: uow}, logger, uow.Events())
}

// Handle loads the plan's run (creating it from the plan's task refs on first report), applies the
// requested status moves, and saves it — all in one transaction, so the seed-read and the write
// share a lock. An illegal transition or unknown task ref is returned as an error (rolled back).
func (h reportRunHandler) Handle(ctx context.Context, cmd ReportRun) (ReportRunResult, error) {
	var res ReportRunResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		run, err := tx.PlanRunReader().FindByPlan(ctx, cmd.PlanID)
		if errors.Is(err, domain.ErrPlanRunNotFound) {
			plan, perr := tx.PlanReader().FindByID(ctx, cmd.PlanID)
			if perr != nil {
				return perr
			}
			run, err = domain.NewPlanRun(cmd.PlanID, cmd.SessionID, planTaskRefs(plan), cmd.Now)
		}
		if err != nil {
			return err
		}
		if cmd.PlanStatus != "" {
			if err := run.SetStatus(cmd.PlanStatus, cmd.Now); err != nil {
				return err
			}
		}
		if cmd.Task != nil {
			if err := run.SetTaskStatus(cmd.Task.Ref, cmd.Task.Status, cmd.Task.Summary, cmd.Now); err != nil {
				return err
			}
		}
		if err := tx.PlanRuns().Save(ctx, run); err != nil {
			return err
		}
		snap := run.Snapshot()
		res = ReportRunResult{PlanRunID: snap.ID, Status: snap.Status}
		return nil
	})
	if err != nil {
		return ReportRunResult{}, err
	}
	return res, nil
}

// planTaskRefs extracts the plan's task refs (in order) to seed a fresh run.
func planTaskRefs(p domain.Plan) []string {
	snap := p.Snapshot()
	refs := make([]string, len(snap.Tasks))
	for i, t := range snap.Tasks {
		refs[i] = t.Ref
	}
	return refs
}
