package domain

import (
	"reflect"
	"strings"
	"testing"
)

// TestPlanContractRendersEveryField asserts the prompt's output schema lists every json field
// of OrchestrationPlan and every nested contract struct it references, so the Go contract and
// the rendered prompt cannot drift.
func TestPlanContractRendersEveryField(t *testing.T) {
	t.Parallel()
	spec := planContractSpec()
	types := []reflect.Type{
		reflect.TypeFor[OrchestrationPlan](), reflect.TypeFor[Scope](), reflect.TypeFor[Risk](),
		reflect.TypeFor[Approval](), reflect.TypeFor[Wave](), reflect.TypeFor[Report](),
		reflect.TypeFor[PlannedTask](), reflect.TypeFor[Assignee](),
	}
	for _, ty := range types {
		for _, f := range reflect.VisibleFields(ty) {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				name = f.Name
			}
			if !strings.Contains(spec, name) {
				t.Fatalf("plan contract spec missing %s field %q", ty.Name(), name)
			}
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
