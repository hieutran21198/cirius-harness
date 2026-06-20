package agent_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/agent"
)

func TestNew(t *testing.T) {
	a, err := agent.New("1", "scribe", agent.ArchetypeCommunicator, "keeps the knowledge store", "", agent.SourceSystem, []string{"t1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !a.Enabled {
		t.Fatal("New should enable the agent by default")
	}
	if a.ID != "1" || a.Name != "scribe" {
		t.Fatalf("New = %+v, want id=1 name=scribe", a)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct {
		id, name  string
		archetype agent.Archetype
		source    agent.Source
	}{
		"empty id":      {"", "scribe", agent.ArchetypeCommunicator, agent.SourceSystem},
		"empty name":    {"1", "", agent.ArchetypeCommunicator, agent.SourceSystem},
		"bad archetype": {"1", "scribe", agent.Archetype("nope"), agent.SourceSystem},
		"bad source":    {"1", "scribe", agent.ArchetypeCommunicator, agent.Source("nope")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := agent.New(c.id, c.name, c.archetype, "", "", c.source, nil); !errors.Is(err, agent.ErrInvalidAgent) {
				t.Fatalf("New err = %v, want ErrInvalidAgent", err)
			}
		})
	}
}
