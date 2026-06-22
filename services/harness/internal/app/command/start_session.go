package command

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// StartSession records the start of a harness run: it ensures the project (by its root
// path) exists and saves a session row under the given id. The id is minted by the
// caller (serve) so it matches the per-session log file. Idempotent on the session id.
type StartSession struct {
	SessionID   domain.SessionID
	ProjectRoot string
	ProjectName string
	CreatedAt   time.Time
}

// StartSessionResult reports the project the session was scoped to.
type StartSessionResult struct {
	ProjectID domain.ProjectID
}

// StartSessionHandler is the use-case contract for starting a session.
type StartSessionHandler decorator.CommandHandler[StartSession, StartSessionResult]

type startSessionHandler struct {
	uow UnitOfWork
}

// NewStartSessionHandler builds the decorated start-session handler.
func NewStartSessionHandler(uow UnitOfWork, logger *slog.Logger) StartSessionHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(startSessionHandler{uow: uow}, logger, uow.Events())
}

// Handle ensures the project then saves the session, in one transaction.
func (h startSessionHandler) Handle(ctx context.Context, cmd StartSession) (StartSessionResult, error) {
	var res StartSessionResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		projectID, err := tx.Projects().EnsureByRoot(ctx, cmd.ProjectRoot, cmd.ProjectName)
		if err != nil {
			return fmt.Errorf("ensure project: %w", err)
		}
		s, err := domain.RehydrateSession(
			cmd.SessionID, projectID, domain.EnvUnset, "", "",
			domain.SessionRunning, cmd.CreatedAt, nil, nil, nil,
		)
		if err != nil {
			return err
		}
		if err := tx.Sessions().Save(ctx, s); err != nil {
			return fmt.Errorf("save session: %w", err)
		}
		res = StartSessionResult{ProjectID: projectID}
		return nil
	})
	if err != nil {
		return StartSessionResult{}, err
	}
	return res, nil
}
