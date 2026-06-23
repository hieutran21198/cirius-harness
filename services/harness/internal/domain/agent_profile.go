package domain

import (
	"fmt"
	"strings"
)

// AgentProfile is the harness-owned persona shared by every specialist agent — everyone but
// council, whose orchestration shape is bespoke (see CouncilProfile). One struct, one renderer:
// the prompt STYLE is a function of the agent's Archetype, not of the agent, so the
// archetype→prompt-style mapping (docs/research/model-families.md) lives in exactly one place
// (SystemPrompt below) instead of being duplicated per agent. The fields are the agent-independent
// anatomy of a governing prompt; SystemPrompt renders them in the voice the agent's model family
// rewards. It implements Persona (ADR-0020, refining ADR-0016).
type AgentProfile struct {
	agent      string    // role name; matches Agent.name and the seeded roster
	archetype  Archetype // selects the rendering style (communicator | principle-driven | utility-runner)
	identity   string    // one sentence: "You are X, the … of the harness team."
	mission    string    // what the agent is for and the one boundary that defines it
	principles []string  // how it works — its method
	output     []string  // the fixed sections of its deliverable — the output contract
	boundaries []string  // hard guardrails, phrased as behavioural intent (not a permission spec)
	effort     string    // the effort-scaling rule (the #1 orchestration lesson), embedded in the prompt
}

// AgentProfile is a Persona.
var _ Persona = AgentProfile{}

// Agent reports the role this profile governs.
func (p AgentProfile) Agent() string { return p.agent }

// promptStyle is the per-archetype rendering recipe: how loud the prompt's scaffolding is.
// Communicator (Claude family) gets council-style uppercase section headers and numbered steps —
// more structure earns more compliance. Principle-driven (GPT family) and utility-runner (MiniMax)
// get quiet labels and bullets — concise principles, less scaffolding, because more rules invite
// more drift; the utility-runner's brevity comes from its short content, not a different shape.
type promptStyle struct {
	heavyHeaders bool // uppercase section() headers vs a quiet "Label:" line
	numbered     bool // "1. item" vs "- item"
}

// styleFor selects the rendering recipe for an archetype. ArchetypeNone never reaches here —
// model-less agents have no persona to render (PersonaFor returns none).
func styleFor(a Archetype) promptStyle {
	if a == ArchetypeCommunicator {
		return promptStyle{heavyHeaders: true, numbered: true}
	}
	return promptStyle{}
}

// SystemPrompt renders the profile into the governing prompt handed over the wire, in the style
// the agent's archetype dictates (see promptStyle). The section order is fixed — identity,
// mission, how-you-work, output contract, boundaries, effort — so every specialist's prompt has
// the same anatomy whatever its style.
func (p AgentProfile) SystemPrompt() string {
	st := styleFor(p.archetype)
	var b strings.Builder
	b.WriteString(p.identity)
	b.WriteString("\n\n")
	b.WriteString(p.mission)

	heading(&b, st, "HOW YOU WORK", "How you work")
	writeList(&b, st, p.principles)

	heading(&b, st, "YOUR OUTPUT — every deliverable carries these", "Your output carries")
	writeList(&b, st, p.output)

	heading(&b, st, "BOUNDARIES — stay inside these", "Boundaries")
	writeList(&b, st, p.boundaries)

	if p.effort != "" {
		heading(&b, st, "EFFORT — scale depth to the request", "Effort")
		b.WriteString("\n")
		b.WriteString(p.effort)
	}

	// When this agent is driven as a task worker, it closes its turn with a structured report
	// envelope the harness validates and council consumes (ADR-0023). The contract is rendered
	// from the Go types so the prompt cannot drift; the prose output above is the human-readable
	// body, the JSON is the machine contract.
	heading(&b, st, "STRUCTURED REPORT — close your turn with this", "Structured report")
	b.WriteString("\n")
	b.WriteString(reportContractSpec())
	return b.String()
}

// heading emits either a council-style uppercase section header (communicator) or a quiet inline
// label (principle-driven / utility-runner). The heavy path reuses section() so communicator
// prompts match council's look.
func heading(b *strings.Builder, st promptStyle, heavy, quiet string) {
	if st.heavyHeaders {
		section(b, heavy)
		return
	}
	b.WriteString("\n\n")
	b.WriteString(quiet)
	b.WriteByte(':')
}

// writeList renders items numbered (communicator) or as bullets (everyone else).
func writeList(b *strings.Builder, st promptStyle, items []string) {
	for i, it := range items {
		if st.numbered {
			fmt.Fprintf(b, "\n%d. %s", i+1, it)
		} else {
			fmt.Fprintf(b, "\n- %s", it)
		}
	}
}
