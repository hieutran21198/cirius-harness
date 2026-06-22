package decorator_test

import (
	"context"
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/app/appctx"
	"harness-workspace/services/harness/internal/app/decorator"
	"harness-workspace/services/harness/internal/domain"
)

// fakeEvents is an in-memory domain.EventWriter that records appended events.
type fakeEvents struct{ appended []domain.Event }

func (f *fakeEvents) Append(_ context.Context, e domain.Event) error {
	f.appended = append(f.appended, e)
	return nil
}

func TestAuditDecoratorRecordsSuccess(t *testing.T) {
	ev := &fakeEvents{}
	h := decorator.ApplyCommandDecorators(&stubHandler{result: "ok"}, discardLogger(), ev)

	if _, err := h.Handle(appctx.WithActor(context.Background(), "pi"), "ping"); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(ev.appended) != 1 {
		t.Fatalf("appended %d events, want 1", len(ev.appended))
	}
	s := ev.appended[0].Snapshot()
	if s.Status != domain.EventOK || s.Actor != "pi" {
		t.Fatalf("event = %+v, want status ok + actor pi", s)
	}
}

func TestAuditDecoratorRecordsError(t *testing.T) {
	ev := &fakeEvents{}
	h := decorator.ApplyCommandDecorators(&stubHandler{err: errors.New("boom")}, discardLogger(), ev)

	if _, err := h.Handle(context.Background(), "ping"); err == nil {
		t.Fatal("Handle should propagate the base error")
	}
	if len(ev.appended) != 1 || ev.appended[0].Snapshot().Status != domain.EventError {
		t.Fatalf("want one event with status error, got %+v", ev.appended)
	}
}
