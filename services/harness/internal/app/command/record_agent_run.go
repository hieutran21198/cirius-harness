package command

import (
	"context"
	"log/slog"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// RecordAgentRun records that an agent participated in a session (a session_agents
// row), with the model it ran with (empty for a model-less run, stored as NULL).
// Idempotent on (session, agent).
type RecordAgentRun struct {
	SessionID domain.SessionID
	AgentID   domain.AgentID
	ModelID   domain.ModelID
}

// RecordAgentRunResult is empty: the command only acknowledges the write.
type RecordAgentRunResult struct{}

// RecordAgentRunHandler is the use-case contract for recording an agent run.
type RecordAgentRunHandler decorator.CommandHandler[RecordAgentRun, RecordAgentRunResult]

type recordAgentRunHandler struct {
	uow UnitOfWork
}

// NewRecordAgentRunHandler builds the decorated record-agent-run handler.
func NewRecordAgentRunHandler(uow UnitOfWork, logger *slog.Logger) RecordAgentRunHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(recordAgentRunHandler{uow: uow}, logger, uow.Events())
}

// Handle appends the agent as a session member.
func (h recordAgentRunHandler) Handle(ctx context.Context, cmd RecordAgentRun) (RecordAgentRunResult, error) {
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		m, err := domain.NewMember(cmd.AgentID, cmd.ModelID)
		if err != nil {
			return err
		}
		return tx.Sessions().AddMember(ctx, cmd.SessionID, m)
	})
	if err != nil {
		return RecordAgentRunResult{}, err
	}
	return RecordAgentRunResult{}, nil
}
