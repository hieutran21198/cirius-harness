package domain

import (
	"fmt"
	"reflect"
	"strings"
)

// OrchestrationPlan is the machine-readable plan council emits (ADR-0017): a human reviews it
// and a future runtime engine drives it. These Go types are the single source of the output
// contract — planContractSpec() renders their JSON shape into council's prompt, so the prompt
// and the types cannot drift (a test asserts every PlannedTask field is rendered). The harness
// does not parse or execute plans yet; that is a deferred milestone.
type OrchestrationPlan struct {
	Intent      string        `json:"intent"`
	Goal        string        `json:"goal"`
	Scope       string        `json:"scope"`
	Assumptions []string      `json:"assumptions"`
	Risks       []string      `json:"risks"`
	Tasks       []PlannedTask `json:"tasks"`
	Approvals   []string      `json:"approvals"`
	Waves       [][]string    `json:"waves"`
	Report      string        `json:"report"`
}

// Assignee is the agent (optionally in a lens) a task is routed to.
type Assignee struct {
	Agent string `json:"agent"`
	Lens  string `json:"lens,omitempty"`
}

// PlannedTask is one node of the plan's task DAG.
type PlannedTask struct {
	ID             string   `json:"id"`
	Category       Category `json:"category"`
	Assignee       Assignee `json:"assignee"`
	Objective      string   `json:"objective"`
	Inputs         string   `json:"inputs,omitempty"`
	ExpectedOutput string   `json:"expected_output"`
	DependsOn      []string `json:"depends_on"`
	Wave           int      `json:"wave"`
	DoD            []string `json:"dod"`
	Gate           string   `json:"gate"`
	RiskLevel      string   `json:"risk_level"`
}

// planContractSpec renders the required output format into council's prompt, derived by
// reflection from the OrchestrationPlan / PlannedTask types so the contract has one source.
func planContractSpec() string {
	var b strings.Builder
	b.WriteString("Emit the plan FIRST as a single machine-readable JSON object, THEN a short " +
		"human-readable summary. A human reviews the plan before it is driven; do not act on it " +
		"yourself. The JSON must match OrchestrationPlan:\n")
	writeShape(&b, "OrchestrationPlan", reflect.TypeFor[OrchestrationPlan]())
	b.WriteString("\nEach entry in \"tasks\" is a PlannedTask:\n")
	writeShape(&b, "PlannedTask", reflect.TypeFor[PlannedTask]())
	return b.String()
}

// writeShape renders a struct's JSON field names as a pseudo-schema block.
func writeShape(b *strings.Builder, name string, t reflect.Type) {
	fmt.Fprintf(b, "%s { ", name)
	fields := jsonFields(t)
	b.WriteString(strings.Join(fields, ", "))
	b.WriteString(" }")
}

// jsonFields returns the json tag names (or field names) of a struct type, in order.
func jsonFields(t reflect.Type) []string {
	var out []string
	for _, f := range reflect.VisibleFields(t) {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			name = f.Name
		}
		out = append(out, name)
	}
	return out
}
