package domain

import (
	"errors"
	"testing"
)

func TestNewModel(t *testing.T) {
	t.Parallel()
	m, err := NewModel("openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	if !m.enabled {
		t.Fatal("NewModel should enable the model by default")
	}
	if m.id == "" {
		t.Fatal("NewModel should mint a non-empty id")
	}
	if m.String() != "openai/gpt-5.5" {
		t.Fatalf("String = %q, want openai/gpt-5.5", m.String())
	}
	if got, want := m.Reference(), (Ref{Provider: "openai", Slug: "gpt-5.5"}); got != want {
		t.Fatalf("Reference = %v, want %v", got, want)
	}
}

func TestNewModelMintsFreshID(t *testing.T) {
	t.Parallel()
	a, _ := NewModel("openai", "gpt-5.5")
	b, _ := NewModel("openai", "gpt-5.5")
	if a.id == b.id {
		t.Fatal("NewModel should mint a fresh id each call")
	}
}

func TestNewModelInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct{ provider, slug string }{
		"empty provider": {"", "gpt-5.5"},
		"empty slug":     {"openai", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewModel(c.provider, c.slug); !errors.Is(err, ErrInvalidModel) {
				t.Fatalf("NewModel(%q,%q) err = %v, want ErrInvalidModel", c.provider, c.slug, err)
			}
		})
	}
}

// TestModelSnapshotRoundTrip proves the persistence view and the reconstitution
// constructor are inverses: Snapshot out, RehydrateModel back in, same state.
func TestModelSnapshotRoundTrip(t *testing.T) {
	t.Parallel()
	orig, err := NewModel("openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	snap := orig.Snapshot()
	if snap.Provider != "openai" || snap.Slug != "gpt-5.5" || !snap.Enabled || snap.ID == "" {
		t.Fatalf("Snapshot = %+v", snap)
	}
	back, err := RehydrateModel(snap.ID, snap.Provider, snap.Slug, snap.Enabled)
	if err != nil {
		t.Fatalf("RehydrateModel: %v", err)
	}
	if back != orig {
		t.Fatalf("round-trip = %+v, want %+v", back, orig)
	}
}

func TestRehydrateModelKeepsDisabled(t *testing.T) {
	t.Parallel()
	m, err := RehydrateModel("1", "openai", "gpt-5.5", false)
	if err != nil {
		t.Fatalf("RehydrateModel: %v", err)
	}
	if m.enabled {
		t.Fatal("RehydrateModel must not re-apply the enabled default")
	}
}

func TestRehydrateModelRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateModel("", "openai", "gpt-5.5", true); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("RehydrateModel with empty id err = %v, want ErrInvalidModel", err)
	}
}
