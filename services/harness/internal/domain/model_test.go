package domain

import (
	"errors"
	"testing"
)

func TestNewModel(t *testing.T) {
	t.Parallel()
	m, err := NewModel(ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	if !m.enabled {
		t.Fatal("NewModel should enable the model by default")
	}
	if m.id == "" {
		t.Fatal("NewModel should mint a non-empty id")
	}
	if m.String() != "pi:openai/gpt-5.5" {
		t.Fatalf("String = %q, want pi:openai/gpt-5.5", m.String())
	}
	if got, want := m.Reference(), (Ref{Client: ClientPi, Provider: "openai", Slug: "gpt-5.5"}); got != want {
		t.Fatalf("Reference = %v, want %v", got, want)
	}
}

func TestNewModelMintsFreshID(t *testing.T) {
	t.Parallel()
	a, _ := NewModel(ClientPi, "openai", "gpt-5.5")
	b, _ := NewModel(ClientPi, "openai", "gpt-5.5")
	if a.id == b.id {
		t.Fatal("NewModel should mint a fresh id each call")
	}
}

// TestNewModelSameNameDifferentClient proves the client is part of identity: the same
// (provider, slug) under two clients are distinct catalog entries (ADR-0015).
func TestNewModelSameNameDifferentClient(t *testing.T) {
	t.Parallel()
	pi, _ := NewModel(ClientPi, "openai", "gpt-5.5")
	oc, _ := NewModel(ClientOpencode, "openai", "gpt-5.5")
	if pi.Reference() == oc.Reference() {
		t.Fatal("same (provider, slug) under different clients must have distinct references")
	}
}

func TestNewModelInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		client         ClientKind
		provider, slug string
	}{
		"unknown client": {ClientKind("bogus"), "openai", "gpt-5.5"},
		"empty client":   {"", "openai", "gpt-5.5"},
		"empty provider": {ClientPi, "", "gpt-5.5"},
		"empty slug":     {ClientPi, "openai", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewModel(c.client, c.provider, c.slug); !errors.Is(err, ErrInvalidModel) {
				t.Fatalf("NewModel(%q,%q,%q) err = %v, want ErrInvalidModel", c.client, c.provider, c.slug, err)
			}
		})
	}
}

// TestModelSnapshotRoundTrip proves the persistence view and the reconstitution
// constructor are inverses: Snapshot out, RehydrateModel back in, same state.
func TestModelSnapshotRoundTrip(t *testing.T) {
	t.Parallel()
	orig, err := NewModel(ClientPi, "openai", "gpt-5.5")
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	snap := orig.Snapshot()
	if snap.Client != ClientPi || snap.Provider != "openai" || snap.Slug != "gpt-5.5" || !snap.Enabled || snap.ID == "" {
		t.Fatalf("Snapshot = %+v", snap)
	}
	back, err := RehydrateModel(snap.ID, snap.Client, snap.Provider, snap.Slug, snap.Enabled)
	if err != nil {
		t.Fatalf("RehydrateModel: %v", err)
	}
	if back != orig {
		t.Fatalf("round-trip = %+v, want %+v", back, orig)
	}
}

func TestRehydrateModelKeepsDisabled(t *testing.T) {
	t.Parallel()
	m, err := RehydrateModel("1", ClientPi, "openai", "gpt-5.5", false)
	if err != nil {
		t.Fatalf("RehydrateModel: %v", err)
	}
	if m.enabled {
		t.Fatal("RehydrateModel must not re-apply the enabled default")
	}
}

func TestRehydrateModelRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateModel("", ClientPi, "openai", "gpt-5.5", true); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("RehydrateModel with empty id err = %v, want ErrInvalidModel", err)
	}
}
