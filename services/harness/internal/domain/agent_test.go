package domain

import (
	"errors"
	"testing"
)

func TestNewAgent(t *testing.T) {
	t.Parallel()
	a, err := NewAgent("scout", ArchetypePrincipleDriven, "explore the code", "a research role", SourceSystem, []ToolID{"t1"})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if !a.enabled {
		t.Fatal("NewAgent should enable the agent by default")
	}
	if a.id == "" || a.name != "scout" {
		t.Fatalf("NewAgent = %+v, want minted id + name=scout", a)
	}
}

func TestNewAgentInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		name      string
		archetype Archetype
		source    Source
	}{
		"empty name":      {"", ArchetypeNone, SourceSystem},
		"unknown source":  {"scout", ArchetypeNone, Source("bogus")},
		"unknown archtyp": {"scout", Archetype("bogus"), SourceSystem},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewAgent(c.name, c.archetype, "", "", c.source, nil); !errors.Is(err, ErrInvalidAgent) {
				t.Fatalf("NewAgent err = %v, want ErrInvalidAgent", err)
			}
		})
	}
}

func TestRehydrateAgentKeepsDisabled(t *testing.T) {
	t.Parallel()
	a, err := RehydrateAgent("1", "scout", ArchetypeNone, "", "", SourceUser, false, nil)
	if err != nil {
		t.Fatalf("RehydrateAgent: %v", err)
	}
	if a.enabled {
		t.Fatal("RehydrateAgent must not re-apply the enabled default")
	}
}

func TestRehydrateAgentRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateAgent("", "scout", ArchetypeNone, "", "", SourceUser, true, nil); !errors.Is(err, ErrInvalidAgent) {
		t.Fatalf("RehydrateAgent with empty id err = %v, want ErrInvalidAgent", err)
	}
}
