package command_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"harness-workspace/services/harness/internal/app/command"
	"harness-workspace/services/harness/internal/domain"
)

// fakeWriter is an in-memory domain.ModelWriter keyed by Ref — mirroring the real
// catalog's natural key, so SaveAll upserts on the ref and ids stay stable. saves
// counts the total models passed to SaveAll across calls.
type fakeWriter struct {
	byRef map[domain.Ref]domain.Model
	saves int
}

// Existing returns the subset of refs already present — the targeted-lookup behaviour.
func (w *fakeWriter) Existing(_ context.Context, refs []domain.Ref) (map[domain.Ref]struct{}, error) {
	out := make(map[domain.Ref]struct{}, len(refs))
	for _, r := range refs {
		if _, ok := w.byRef[r]; ok {
			out[r] = struct{}{}
		}
	}
	return out, nil
}

func (w *fakeWriter) SaveAll(_ context.Context, ms []domain.Model) error {
	for _, m := range ms {
		w.saves++
		ref := m.Reference()
		if _, ok := w.byRef[ref]; ok {
			continue // upsert on the natural key keeps the existing entry (and its id)
		}
		w.byRef[ref] = m
	}
	return nil
}

func (w *fakeWriter) Count(_ context.Context) (int, error) { return len(w.byRef), nil }

// fakeUoW implements command.UnitOfWork; DoTx runs the closure with itself (no real
// transaction in the in-memory fake), exercising the handler's orchestration.
type fakeUoW struct{ w *fakeWriter }

func newFakeUoW() *fakeUoW { return &fakeUoW{w: &fakeWriter{byRef: map[domain.Ref]domain.Model{}}} }

func (u *fakeUoW) Models() domain.ModelWriter { return u.w }

// The audit/session writers are unused by the model-sync tests; a nil Events writer
// makes ApplyCommandDecorators skip the audit layer.
func (u *fakeUoW) Events() domain.EventWriter          { return nil }
func (u *fakeUoW) Projects() domain.ProjectWriter      { return nil }
func (u *fakeUoW) Sessions() domain.SessionWriter      { return nil }
func (u *fakeUoW) Plans() domain.PlanWriter            { return nil }
func (u *fakeUoW) PlanRuns() domain.PlanRunWriter      { return nil }
func (u *fakeUoW) PlanReader() domain.PlanReader       { return nil }
func (u *fakeUoW) PlanRunReader() domain.PlanRunReader { return nil }

func (u *fakeUoW) DoTx(ctx context.Context, fn func(context.Context, command.TransactionalUnitOfWork) error) error {
	return fn(ctx, u)
}

// refs builds (provider, slug) pairs all attributed to one client (ClientPi) — the
// client is frame-level, so a single sync carries one client's models.
func refs(provSlug ...string) []domain.Ref {
	out := make([]domain.Ref, 0, len(provSlug)/2)
	for i := 0; i+1 < len(provSlug); i += 2 {
		out = append(out, domain.Ref{Client: domain.ClientPi, Provider: provSlug[i], Slug: provSlug[i+1]})
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
		snap := m.Snapshot()
		if !snap.Enabled {
			t.Fatalf("model %s should be enabled", m.String())
		}
		if snap.ID == "" {
			t.Fatalf("model %s should have a minted id", m.String())
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
	gpt := domain.Ref{Client: domain.ClientPi, Provider: "openai", Slug: "gpt-5.5"}
	firstID := uow.w.byRef[gpt].Snapshot().ID

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
	if got := uow.w.byRef[gpt].Snapshot().ID; got != firstID {
		t.Fatalf("existing id changed: %s -> %s", firstID, got)
	}
	if uow.w.saves != 2 {
		t.Fatalf("Save called %d times, want 2 (one per new ref, existing skipped)", uow.w.saves)
	}
}

func TestSyncModelsRejectsInvalid(t *testing.T) {
	ctx := context.Background()
	h := command.NewSyncModelsHandler(newFakeUoW(), discardLogger())
	if _, err := h.Handle(ctx, command.SyncModels{Reported: []domain.Ref{{Client: domain.ClientPi, Provider: "", Slug: "x"}}}); err == nil {
		t.Fatal("Handle should reject a ref with empty provider")
	}
}
