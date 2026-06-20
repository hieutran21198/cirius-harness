package command_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain/model"
)

// fakeWriter is an in-memory model.Writer keyed by (provider, slug) — mirroring the
// real catalog's natural key, so Save upserts on Ref() and ids stay stable.
type fakeWriter struct {
	byRef map[string]model.Model
	saves int
}

func (w *fakeWriter) Exists(_ context.Context, provider, slug string) (bool, error) {
	_, ok := w.byRef[provider+"/"+slug]
	return ok, nil
}

func (w *fakeWriter) Save(_ context.Context, m model.Model) error {
	w.saves++
	if existing, ok := w.byRef[m.Ref()]; ok {
		existing.Enabled = m.Enabled
		w.byRef[m.Ref()] = existing
		return nil
	}
	w.byRef[m.Ref()] = m
	return nil
}

func (w *fakeWriter) Count(_ context.Context) (int, error) { return len(w.byRef), nil }

// fakeUoW implements command.UnitOfWork; DoTx runs the closure with itself (no real
// transaction in the in-memory fake), exercising the handler's orchestration.
type fakeUoW struct{ w *fakeWriter }

func newFakeUoW() *fakeUoW { return &fakeUoW{w: &fakeWriter{byRef: map[string]model.Model{}}} }

func (u *fakeUoW) Models() model.Writer { return u.w }

func (u *fakeUoW) DoTx(ctx context.Context, fn func(context.Context, command.TransactionalUnitOfWork) error) error {
	return fn(ctx, u)
}

func refs(provSlug ...string) []model.Model {
	out := make([]model.Model, 0, len(provSlug)/2)
	for i := 0; i+1 < len(provSlug); i += 2 {
		out = append(out, model.Model{Provider: provSlug[i], Slug: provSlug[i+1]})
	}
	return out
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestSyncModelsAddsAndCountsTotal(t *testing.T) {
	ctx := context.Background()
	uow := newFakeUoW()
	h := command.NewSyncModelsHandler(uow, discardLogger())

	res, err := h.Handle(ctx, command.SyncModels{
		Reported: refs("anthropic", "claude-opus-4-8", "openai", "gpt-5.5"),
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if res.Added != 2 || res.Total != 2 {
		t.Fatalf("got %+v, want added=2 total=2", res)
	}
	// Enabled must be forced true by the use case, and an id minted.
	for _, m := range uow.w.byRef {
		if !m.Enabled {
			t.Fatalf("model %s should be enabled", m.Ref())
		}
		if m.ID == "" {
			t.Fatalf("model %s should have a minted id", m.Ref())
		}
	}
}

func TestSyncModelsCumulativeIdempotent(t *testing.T) {
	ctx := context.Background()
	uow := newFakeUoW()
	h := command.NewSyncModelsHandler(uow, discardLogger())

	if _, err := h.Handle(ctx, command.SyncModels{Reported: refs("openai", "gpt-5.5")}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	firstID := uow.w.byRef["openai/gpt-5.5"].ID

	// Re-sync the same ref plus a new one.
	res, err := h.Handle(ctx, command.SyncModels{
		Reported: refs("openai", "gpt-5.5", "deepseek", "deepseek-v3"),
	})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if res.Added != 1 || res.Total != 2 {
		t.Fatalf("got %+v, want added=1 total=2", res)
	}
	// The existing ref keeps its id and is not re-saved (skipped, not upserted).
	if got := uow.w.byRef["openai/gpt-5.5"].ID; got != firstID {
		t.Fatalf("existing id changed: %s -> %s", firstID, got)
	}
	if uow.w.saves != 2 {
		t.Fatalf("Save called %d times, want 2 (one per new ref, existing skipped)", uow.w.saves)
	}
}

func TestSyncModelsRejectsInvalid(t *testing.T) {
	ctx := context.Background()
	h := command.NewSyncModelsHandler(newFakeUoW(), discardLogger())
	if _, err := h.Handle(ctx, command.SyncModels{Reported: []model.Model{{Provider: "", Slug: "x"}}}); err == nil {
		t.Fatal("Handle should reject a ref with empty provider")
	}
}
