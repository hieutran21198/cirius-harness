// Package command holds the application's write-side use cases. Each is a
// CommandHandler: a plain command struct (the intent), a concrete handler over the
// driven ports it needs (a UnitOfWork — see port.go), and a constructor that applies
// the cross-cutting decorators. The Handle method is pure business logic (ADR-0012).
package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain/model"
)

// SyncModels is the command to sync a client's reported models into the catalog.
// Each reported model carries (provider, slug); the id is minted for new entries.
type SyncModels struct {
	Reported []model.Model
}

// SyncModelsResult reports the outcome of a catalog sync.
type SyncModelsResult struct {
	// Added is the number of newly inserted (provider, slug) refs.
	Added int
	// Total is the catalog size after the sync.
	Total int
}

// SyncModelsHandler is the use-case contract for the model-sync command.
type SyncModelsHandler decorator.CommandHandler[SyncModels, SyncModelsResult]

type syncModelsHandler struct {
	uow UnitOfWork
}

// NewSyncModelsHandler builds the decorated model-sync handler over the unit of work.
func NewSyncModelsHandler(uow UnitOfWork, logger *slog.Logger) SyncModelsHandler {
	if uow == nil {
		panic("command: nil unit of work")
	}
	return decorator.ApplyCommandDecorators(syncModelsHandler{uow: uow}, logger)
}

// Handle upserts the reported models into the catalog cumulatively (ADR-0011):
// reported (provider, slug) refs absent from the catalog are inserted with a freshly
// minted UUID v7 (enabled); nothing is deleted. The whole sync runs in one
// transaction (ADR-0013); it returns how many were newly added and the catalog
// total afterwards.
func (h syncModelsHandler) Handle(ctx context.Context, cmd SyncModels) (SyncModelsResult, error) {
	var res SyncModelsResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		models := tx.Models()
		added := 0
		for _, m := range cmd.Reported {
			m.Enabled = true
			if err := m.Validate(); err != nil {
				return err
			}
			exists, err := models.Exists(ctx, m.Provider, m.Slug)
			if err != nil {
				return fmt.Errorf("check model %s: %w", m.Ref(), err)
			}
			if exists {
				continue // already in the cumulative catalog
			}
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("mint model id: %w", err)
			}
			m.ID = id.String()
			if err := models.Save(ctx, m); err != nil {
				return fmt.Errorf("save model %s: %w", m.Ref(), err)
			}
			added++
		}
		total, err := models.Count(ctx)
		if err != nil {
			return fmt.Errorf("count models: %w", err)
		}
		res = SyncModelsResult{Added: added, Total: total}
		return nil
	})
	if err != nil {
		return SyncModelsResult{}, err
	}
	return res, nil
}
