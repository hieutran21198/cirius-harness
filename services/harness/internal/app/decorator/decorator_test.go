package decorator_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"harness-workspace/services/harness/internal/app/decorator"
)

// stubHandler is a CommandHandler that returns canned values and records the call.
type stubHandler struct {
	result string
	err    error
	called bool
}

func (h *stubHandler) Handle(_ context.Context, cmd string) (string, error) {
	h.called = true
	return h.result, h.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestApplyCommandDecoratorsDelegatesResult(t *testing.T) {
	base := &stubHandler{result: "ok"}
	h := decorator.ApplyCommandDecorators(base, discardLogger(), nil)

	got, err := h.Handle(context.Background(), "ping")
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !base.called {
		t.Fatal("decorator did not delegate to the base handler")
	}
	if got != "ok" {
		t.Fatalf("got %q, want %q", got, "ok")
	}
}

func TestApplyCommandDecoratorsDelegatesError(t *testing.T) {
	want := errors.New("boom")
	h := decorator.ApplyCommandDecorators(&stubHandler{err: want}, discardLogger(), nil)

	if _, err := h.Handle(context.Background(), "ping"); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}
