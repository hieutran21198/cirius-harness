package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// OrchestrationPlan is the machine-readable plan council emits (ADR-0017, ADR-0019): a human
// reviews council's markdown rendering, and on approval council emits this object as a single
// JSON block. The Pi extension captures it and submits it over the submit_plan frame; the
// harness validates it into a domain.Plan and persists it (a future runtime engine drives it).
// These Go types are the single source of the output contract — planContractSpec() renders
// their JSON shape into council's prompt, so the prompt and the types cannot drift (a test
// asserts every field, nested included, is rendered).
type OrchestrationPlan struct {
	Intent      string        `json:"intent"`
	Goal        string        `json:"goal"`
	Scope       Scope         `json:"scope"`
	Assumptions []string      `json:"assumptions"`
	Risks       []Risk        `json:"risks"`
	Tasks       []PlannedTask `json:"tasks"`
	Approvals   []Approval    `json:"approvals"`
	Waves       []Wave        `json:"waves"`
	Report      Report        `json:"report"`
}

// Scope bounds what the plan touches: what is in play and what is deliberately excluded.
type Scope struct {
	Primary    []string `json:"primary"`
	OutOfScope []string `json:"out_of_scope_by_default"`
}

// Risk is one weighed risk: how severe, and what it is.
type Risk struct {
	Level       string `json:"level"`
	Description string `json:"description"`
}

// Approval is a human gate the plan requires before a given task is driven.
type Approval struct {
	Type           string `json:"type"`
	RequiredBefore string `json:"required_before"`
	Reason         string `json:"reason"`
	Question       string `json:"question"`
}

// Wave groups task ids that can run concurrently — one rung of the dependency-ordered DAG.
type Wave struct {
	Wave  int      `json:"wave"`
	Tasks []string `json:"tasks"`
}

// Report is the plan's closing summary: its status, a prose summary, and the overall
// definition of done.
type Report struct {
	Status           string   `json:"status"`
	Summary          string   `json:"summary"`
	DefinitionOfDone []string `json:"definition_of_done"`
}

// Assignee is the agent (optionally in a lens) a task is routed to.
type Assignee struct {
	Agent string `json:"agent"`
	Lens  string `json:"lens,omitempty"`
}

// UnmarshalJSON accepts either the object form {"agent","lens"} or a bare string "agent"
// (council may emit either, and early plans used a bare string); a bare string sets Agent
// with no lens.
func (a *Assignee) UnmarshalJSON(data []byte) error {
	if t := strings.TrimSpace(string(data)); strings.HasPrefix(t, `"`) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		a.Agent, a.Lens = s, ""
		return nil
	}
	type alias Assignee // avoid recursing into this method
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*a = Assignee(v)
	return nil
}

// PlannedTask is one node of the plan's task DAG. ID is the plan-local task ref ("T1") that
// DependsOn and Wave.Tasks reference.
type PlannedTask struct {
	ID             string   `json:"id"`
	Category       Category `json:"category"`
	Assignee       Assignee `json:"assignee"`
	Objective      string   `json:"objective"`
	Inputs         []string `json:"inputs,omitempty"`
	ExpectedOutput string   `json:"expected_output"`
	DependsOn      []string `json:"depends_on"`
	Wave           int      `json:"wave"`
	DoD            []string `json:"dod"`
	Gate           string   `json:"gate"`
	RiskLevel      string   `json:"risk_level"`
}

// planContractSpec renders the required JSON output format into council's prompt, derived by
// reflection from the OrchestrationPlan type (and every nested struct it references) so the
// contract has one source.
func planContractSpec() string {
	var b strings.Builder
	b.WriteString("When you emit the final plan (only after the human approves — see below), " +
		"output it as a SINGLE fenced ```json block containing one object matching " +
		"OrchestrationPlan, and nothing else in that block. The harness captures and persists " +
		"it. The shapes (a name after a field is its nested shape, listed below):\n")
	writeShape(&b, "OrchestrationPlan", reflect.TypeFor[OrchestrationPlan](), map[string]bool{})
	return b.String()
}

// writeShape renders a struct type's JSON field names as a pseudo-schema block, then renders
// the shape of every nested struct type it references (each once), so the contract is complete.
func writeShape(b *strings.Builder, name string, t reflect.Type, seen map[string]bool) {
	if seen[name] {
		return
	}
	seen[name] = true

	var nested []reflect.Type
	fmt.Fprintf(b, "\n%s { ", name)
	fields := make([]string, 0, t.NumField())
	for _, f := range reflect.VisibleFields(t) {
		jsonName, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if jsonName == "" || jsonName == "-" {
			jsonName = f.Name
		}
		fields = append(fields, jsonName)
		if et := structElem(f.Type); et != nil {
			nested = append(nested, et)
		}
	}
	b.WriteString(strings.Join(fields, ", "))
	b.WriteString(" }")
	for _, et := range nested {
		writeShape(b, et.Name(), et, seen)
	}
}

// structElem returns the struct type a field carries directly or as a slice element, or nil
// when the field is not (a slice of) a struct. Named string enums like Category are not
// structs and return nil.
func structElem(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return nil
}
