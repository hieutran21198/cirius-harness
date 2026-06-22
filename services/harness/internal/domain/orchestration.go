package domain

import (
	"fmt"
	"strings"
)

// This file is council's orchestration framework as typed data (ADR-0017). Council is a
// control-plane brain: it classifies a request, weighs it across dimensions, decomposes it
// into categorized tasks, matches each to the best-fit team agent by capability, sequences
// them into dependency-ordered waves, and governs the flow with quality gates and human
// approval. The CouncilProfile below holds that framework; SystemPrompt() renders it into
// the mechanics-heavy prompt handed over the wire. Council never executes — it plans; a
// human reviews the plan and (later) a runtime engine drives it.

// Intent is the kind of work a request asks for; council classifies the request into one
// before planning.
type Intent struct {
	Name string
	When string
}

// TaskDimension is one lens council weighs every request through before decomposing it.
type TaskDimension struct {
	Name     string
	Question string
}

// Category is a kind of work a task falls into. Council groups a request into categories and
// routes each to the best-fit agent (possibly in a lens) — the taxonomy is richer than the
// team so several categories map onto one agent (ADR-0017 / PDR-0002).
type Category string

// The work categories council recognises. The taxonomy is richer than the team; several
// categories route onto one agent (possibly in a lens).
const (
	CategoryExplore     Category = "explore"
	CategoryResearch    Category = "research"
	CategoryArchitect   Category = "architect"
	CategoryPlan        Category = "plan"
	CategoryImplement   Category = "implement"
	CategoryTest        Category = "test"
	CategoryReview      Category = "review"
	CategorySecurity    Category = "security"
	CategoryPerformance Category = "performance"
	CategoryDocs        Category = "docs"
	CategoryMigration   Category = "migration"
	CategoryDevops      Category = "devops"
	CategoryIntegrate   Category = "integrate"
)

// AgentCapability is council's model of one real team agent: what it is good at, what it may
// touch, and how it trades cost for depth. Authored from .cirius-harness/00-system.yaml
// (tools/permissions), docs/research/model-families.md (cost/speed/reliability), and
// docs/pdr/0001 (model per agent). Lenses are focus-modes an agent can be summoned in so the
// team stays lean (PDR-0002).
type AgentCapability struct {
	Agent         string
	Handles       []Category
	Tools         []string
	Archetype     Archetype
	CostSpeed     string
	Reliability   string
	RiskTolerance string
	Permissions   string
	Lenses        []string
}

// RoutingRule is a when→assign heuristic council applies during decomposition.
type RoutingRule struct {
	When     string
	AssignTo string
	Output   string
}

// StageOwner is who drives a pipeline stage.
type StageOwner string

// Who drives a pipeline stage.
const (
	OwnerCouncil StageOwner = "council"
	OwnerAgent   StageOwner = "agent"
	OwnerHuman   StageOwner = "human"
)

// PipelineStage is one step of council's end-to-end orchestration flow.
type PipelineStage struct {
	Name    string
	Purpose string
	Owner   StageOwner
}

// QualityGate is one rung of the four-gate human-in-the-loop model (advisory → validating →
// blocking → escalating): how much oversight a task needs before it proceeds.
type QualityGate struct {
	Name   string
	When   string
	Action string
}

// CouncilProfile is council's full orchestration framework. It implements Persona: Agent()
// names the role and SystemPrompt() renders the framework into the governing prompt.
type CouncilProfile struct {
	identity     string
	mission      string
	effort       string
	intents      []Intent
	dimensions   []TaskDimension
	categories   []Category
	capabilities []AgentCapability
	rules        []RoutingRule
	pipeline     []PipelineStage
	gates        []QualityGate
	dod          []string
	formula      []string
}

// Agent reports the role this profile governs.
func (c CouncilProfile) Agent() string { return "council" }

// SystemPrompt renders the orchestration framework into the mechanics-driven prompt handed to
// the client. Council is Claude-family, which rewards this explicit, checklisted style
// (docs/research/model-families.md, agent-orchestration.md).
func (c CouncilProfile) SystemPrompt() string {
	var b strings.Builder
	b.WriteString(c.identity)
	b.WriteString("\n\n")
	b.WriteString(c.mission)

	section(&b, "HOW YOU OPERATE — run the request through this flow")
	for i, s := range c.pipeline {
		fmt.Fprintf(&b, "\n%d. %s (%s) — %s", i+1, s.Name, s.Owner, s.Purpose)
	}

	section(&b, "CLASSIFY THE INTENT — what kind of work is this")
	for _, in := range c.intents {
		fmt.Fprintf(&b, "\n- %s — %s", in.Name, in.When)
	}

	section(&b, "WEIGH EVERY REQUEST ACROSS THESE DIMENSIONS")
	for i, d := range c.dimensions {
		fmt.Fprintf(&b, "\n%d. %s — %s", i+1, d.Name, d.Question)
	}

	section(&b, "TASK CATEGORIES — group the work, then route each to an agent")
	b.WriteString("\n")
	b.WriteString(strings.Join(categoryNames(c.categories), " · "))

	section(&b, "YOUR TEAM — capabilities (route to the best fit; prefer the cheapest capable agent)")
	for _, a := range c.capabilities {
		fmt.Fprintf(&b, "\n- %s [%s] — handles %s; tools %s; %s, reliability %s; %s; %s",
			a.Agent, a.Archetype, strings.Join(categoryNames(a.Handles), "/"),
			strings.Join(a.Tools, ","), a.CostSpeed, a.Reliability, a.Permissions, a.RiskTolerance)
		if len(a.Lenses) > 0 {
			fmt.Fprintf(&b, "; lenses: %s", strings.Join(a.Lenses, ", "))
		}
	}

	section(&b, "ROUTING RULES")
	for _, r := range c.rules {
		fmt.Fprintf(&b, "\n- %s → %s → %s", r.When, r.AssignTo, r.Output)
	}

	section(&b, "ASSIGNMENT — weigh each task by")
	b.WriteString("\n")
	b.WriteString(strings.Join(c.formula, " + "))
	b.WriteString("\nMatch the task to the agent whose capability best fits: strongest agent for " +
		"high-risk or deep-reasoning work, cheapest capable agent for mechanical work. Summon an " +
		"agent in a lens when its base role covers the category.")

	section(&b, "QUALITY GATES — four-gate human-in-the-loop")
	for _, g := range c.gates {
		fmt.Fprintf(&b, "\n- %s — %s → %s", g.Name, g.When, g.Action)
	}
	b.WriteString("\nDefinition of done: ")
	b.WriteString(strings.Join(c.dod, " · "))

	section(&b, "OUTPUT")
	b.WriteString("\n")
	b.WriteString(planContractSpec())

	if c.effort != "" {
		section(&b, "EFFORT — scale depth to the request")
		b.WriteString("\n")
		b.WriteString(c.effort)
	}
	return b.String()
}

// section writes a blank line, an uppercase header, and a rule under it.
func section(b *strings.Builder, title string) {
	b.WriteString("\n\n")
	b.WriteString(title)
}

// categoryNames maps categories to their string form for rendering.
func categoryNames(cs []Category) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = string(c)
	}
	return out
}
