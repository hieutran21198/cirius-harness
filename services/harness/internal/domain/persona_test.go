package domain

import (
	"strings"
	"testing"
)

func TestPersonaForCouncil(t *testing.T) {
	t.Parallel()
	p, ok := PersonaFor("council")
	if !ok {
		t.Fatal("PersonaFor(council) = false, want the council persona")
	}
	if p.Agent() != "council" {
		t.Fatalf("Agent() = %q, want council", p.Agent())
	}
	prompt := p.SystemPrompt()
	// The rendered prompt must carry the orchestration framework and stay plan-only.
	for _, want := range []string{
		"Classify intent", "Definition of done", "ROUTING RULES",
		"TaskType + Risk + Scope", "do not write or edit code", "human approval",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("council prompt missing %q", want)
		}
	}
}

func TestPersonaForUnknown(t *testing.T) {
	t.Parallel()
	if _, ok := PersonaFor("nope"); ok {
		t.Fatal("PersonaFor(nope) = true, want no persona for an unknown agent")
	}
}

// TestCouncilReferencesRealAgents keeps council's capability roster and routing rules in sync
// with the actual team. team mirrors the agents seeded from .cirius-harness/00-system.yaml; if
// a role is renamed or removed, council must not route to a name that no longer exists.
func TestCouncilReferencesRealAgents(t *testing.T) {
	t.Parallel()
	team := map[string]bool{
		"prayer": true, "council": true, "planner": true, "implementer": true,
		"researcher": true, "explorer": true, "reviewer": true, "scribe": true,
	}
	for _, c := range council.capabilities {
		if !team[c.Agent] {
			t.Fatalf("council models capability for %q, not a real team role", c.Agent)
		}
	}
	// Routing rules name agents in prose; assert every team agent council relies on is real by
	// checking the capability roster covers the non-prayer working roles.
	for _, want := range []string{"planner", "implementer", "researcher", "explorer", "reviewer", "scribe"} {
		found := false
		for _, c := range council.capabilities {
			if c.Agent == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("council capability roster missing working role %q", want)
		}
	}
}
