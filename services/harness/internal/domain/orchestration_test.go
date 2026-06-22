package domain

import (
	"reflect"
	"strings"
	"testing"
)

// TestPlanContractRendersEveryTaskField asserts the prompt's output schema lists every
// PlannedTask json field, so the Go contract and the rendered prompt cannot drift.
func TestPlanContractRendersEveryTaskField(t *testing.T) {
	t.Parallel()
	spec := planContractSpec()
	for _, f := range jsonFields(reflect.TypeFor[PlannedTask]()) {
		if !strings.Contains(spec, f) {
			t.Fatalf("plan contract spec missing PlannedTask field %q", f)
		}
	}
	for _, f := range jsonFields(reflect.TypeFor[OrchestrationPlan]()) {
		if !strings.Contains(spec, f) {
			t.Fatalf("plan contract spec missing OrchestrationPlan field %q", f)
		}
	}
}

// TestCouncilPromptRendersFramework asserts the major framework sections all reach the prompt.
func TestCouncilPromptRendersFramework(t *testing.T) {
	t.Parallel()
	prompt := council.SystemPrompt()
	for _, want := range []string{
		"HOW YOU OPERATE", "WEIGH EVERY REQUEST", "TASK CATEGORIES", "YOUR TEAM",
		"ASSIGNMENT", "QUALITY GATES", "OUTPUT", "EFFORT",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("council prompt missing section %q", want)
		}
	}
	// All 7 dimensions render.
	if got := len(council.dimensions); got != 7 {
		t.Fatalf("council has %d dimensions, want 7", got)
	}
	for _, d := range council.dimensions {
		if !strings.Contains(prompt, d.Name) {
			t.Fatalf("council prompt missing dimension %q", d.Name)
		}
	}
}
