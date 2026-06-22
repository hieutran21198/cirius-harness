package domain

import (
	"reflect"
	"strings"
	"testing"
)

// archetypeBySeed mirrors the archetypes seeded in
// migrations/20260619091805_seed_system_agents.sql (and .cirius-harness/00-system.yaml) for the
// working specialists. A persona's archetype drives its prompt STYLE, so it must match the
// archetype the agent actually runs on (the model family); if a seed archetype changes, this
// table must change too — a failing test here is the intended signal (ADR-0020). council is
// excluded (it is a CouncilProfile, not an AgentProfile); prayer is excluded (model-less, none).
var archetypeBySeed = map[string]Archetype{
	"planner":     ArchetypeCommunicator,
	"implementer": ArchetypePrincipleDriven,
	"researcher":  ArchetypePrincipleDriven,
	"explorer":    ArchetypeUtilityRunner,
	"reviewer":    ArchetypeCommunicator,
	"scribe":      ArchetypeCommunicator,
}

// distinctiveSubstring is one stable, role-distinctive phrase each specialist's prompt must carry.
// Kept deliberately short so the test guards "this is the right role" without being brittle to
// wording changes elsewhere in the prompt.
var distinctiveSubstring = map[string]string{
	"planner":     "implementation architect",
	"implementer": "only agent that edits source",
	"researcher":  "only web-enabled agent",
	"explorer":    "fast scanner",
	"reviewer":    "never change the work",
	"scribe":      "knowledge store",
}

func TestSpecialistPersonasResolve(t *testing.T) {
	t.Parallel()
	for name := range archetypeBySeed {
		p, ok := PersonaFor(name)
		if !ok {
			t.Fatalf("PersonaFor(%q) = false, want a persona", name)
		}
		if p.Agent() != name {
			t.Fatalf("Agent() = %q, want %q", p.Agent(), name)
		}
		prompt := p.SystemPrompt()
		if strings.TrimSpace(prompt) == "" {
			t.Fatalf("%s SystemPrompt() is empty", name)
		}
		if want := distinctiveSubstring[name]; !strings.Contains(prompt, want) {
			t.Fatalf("%s prompt missing distinctive %q", name, want)
		}
	}
}

// TestSpecialistArchetypesMatchSeed is the consistency guard: each profile's archetype field must
// equal the archetype the agent is seeded with, so the rendered style cannot drift from the model
// family the agent runs on.
func TestSpecialistArchetypesMatchSeed(t *testing.T) {
	t.Parallel()
	for name, want := range archetypeBySeed {
		p, ok := PersonaFor(name)
		if !ok {
			t.Fatalf("PersonaFor(%q) = false", name)
		}
		ap, ok := p.(AgentProfile)
		if !ok {
			t.Fatalf("%s persona is %T, want AgentProfile", name, p)
		}
		if ap.archetype != want {
			t.Fatalf("%s archetype = %q, want %q (seed)", name, ap.archetype, want)
		}
	}
}

// TestArchetypeRenderStyle asserts the archetype actually drives the prompt style: communicator
// profiles render council-style uppercase headers and numbered steps; principle-driven profiles
// render their principles as plain bullets under a quiet label (no uppercase header); the
// utility-runner explorer is terser than a communicator peer. Matches on stable anchors only.
func TestArchetypeRenderStyle(t *testing.T) {
	t.Parallel()

	communicator := mustProfile(t, "reviewer")
	cPrompt := communicator.SystemPrompt()
	if !strings.Contains(cPrompt, "HOW YOU WORK") {
		t.Fatalf("communicator prompt missing uppercase section header")
	}
	if !strings.Contains(cPrompt, "\n1. ") {
		t.Fatalf("communicator prompt missing a numbered list marker")
	}

	principle := mustProfile(t, "researcher")
	pPrompt := principle.SystemPrompt()
	if strings.Contains(pPrompt, "HOW YOU WORK") {
		t.Fatalf("principle-driven prompt should not use the uppercase header")
	}
	if !strings.Contains(pPrompt, "How you work:") {
		t.Fatalf("principle-driven prompt missing the quiet label")
	}
	// Its first principle renders verbatim as a bullet (analog of council's dimension-render check).
	if first := principle.principles[0]; !strings.Contains(pPrompt, "\n- "+first) {
		t.Fatalf("principle-driven prompt missing its first principle as a bullet")
	}

	explorer := mustProfile(t, "explorer")
	if len(explorer.SystemPrompt()) >= len(communicator.SystemPrompt()) {
		t.Fatalf("utility-runner prompt should be terser than a communicator peer")
	}
}

// TestAgentProfileRendersEveryField is the drift guard (analog of TestPlanContractRendersEveryField):
// every authored field of every specialist profile must appear in its rendered prompt, so content
// cannot be added to a profile yet silently dropped from the prompt. agent and archetype are
// rendering inputs (they select style / key the registry), not emitted text, so they are skipped.
func TestAgentProfileRendersEveryField(t *testing.T) {
	t.Parallel()
	skip := map[string]bool{"agent": true, "archetype": true}
	for name := range archetypeBySeed {
		ap := mustProfile(t, name)
		prompt := ap.SystemPrompt()
		v := reflect.ValueOf(ap)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			if skip[field.Name] {
				continue
			}
			switch field.Type.Kind() {
			case reflect.String:
				if s := v.Field(i).String(); s != "" && !strings.Contains(prompt, s) {
					t.Fatalf("%s prompt missing field %s", name, field.Name)
				}
			case reflect.Slice:
				sl := v.Field(i)
				for j := 0; j < sl.Len(); j++ {
					if s := sl.Index(j).String(); s != "" && !strings.Contains(prompt, s) {
						t.Fatalf("%s prompt missing %s[%d]", name, field.Name, j)
					}
				}
			}
		}
	}
}

// TestRegistryCoversWorkingAgents asserts every working role resolves to a persona and that the
// model-less prayer does not — pinning the one team name that must never have a persona.
func TestRegistryCoversWorkingAgents(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"council", "planner", "implementer", "researcher", "explorer", "reviewer", "scribe"} {
		if _, ok := PersonaFor(name); !ok {
			t.Fatalf("PersonaFor(%q) = false, want a persona for a working agent", name)
		}
	}
	if _, ok := PersonaFor("prayer"); ok {
		t.Fatal("PersonaFor(prayer) = true, want no persona for the model-less agent")
	}
}

func mustProfile(t *testing.T, name string) AgentProfile {
	t.Helper()
	p, ok := PersonaFor(name)
	if !ok {
		t.Fatalf("PersonaFor(%q) = false", name)
	}
	ap, ok := p.(AgentProfile)
	if !ok {
		t.Fatalf("%s persona is %T, want AgentProfile", name, p)
	}
	return ap
}
