// Package command holds the application's write-side use cases. Each is a
// CommandHandler: a plain command struct (the intent), a concrete handler over the
// driven ports it needs (a UnitOfWork — see port.go), and a constructor that applies
// the cross-cutting decorators. The Handle method is pure business logic (ADR-0012).
package command

import (
	"context"
	"fmt"
	"log/slog"

	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// SyncModels is the command to sync a client's reported models into the catalog.
// Each reported ref carries (provider, slug); the id is minted for new entries.
type SyncModels struct {
	Reported []domain.Ref
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
// minted UUID v7 (enabled); nothing is deleted. It reads the present keys once, then
// batch-inserts the new ones — the whole sync runs in one transaction (ADR-0013).
// It returns how many were newly added and the catalog total afterwards.
func (h syncModelsHandler) Handle(ctx context.Context, cmd SyncModels) (SyncModelsResult, error) {
	var res SyncModelsResult
	err := h.uow.DoTx(ctx, func(ctx context.Context, tx TransactionalUnitOfWork) error {
		models := tx.Models()
		// Dedup the reported refs first: keeps Added accurate and collapses
		// within-batch duplicates before the existence lookup.
		uniq := make([]domain.Ref, 0, len(cmd.Reported))
		seen := make(map[domain.Ref]struct{}, len(cmd.Reported))
		for _, r := range cmd.Reported {
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			uniq = append(uniq, r)
		}
		existing, err := models.Existing(ctx, uniq)
		if err != nil {
			return fmt.Errorf("load model keys: %w", err)
		}
		newModels := make([]domain.Model, 0, len(uniq))
		for _, r := range uniq {
			if _, ok := existing[r]; ok {
				continue // already in the cumulative catalog
			}
			m, mkErr := domain.NewModel(r.Client, r.Provider, r.Slug)
			if mkErr != nil {
				return mkErr
			}
			newModels = append(newModels, m)
		}
		if len(newModels) > 0 {
			if err = models.SaveAll(ctx, newModels); err != nil {
				return fmt.Errorf("save models: %w", err)
			}
		}
		total, err := models.Count(ctx)
		if err != nil {
			return fmt.Errorf("count models: %w", err)
		}
		res = SyncModelsResult{Added: len(newModels), Total: total}
		return nil
	})
	if err != nil {
		return SyncModelsResult{}, err
	}
	return res, nil
}
