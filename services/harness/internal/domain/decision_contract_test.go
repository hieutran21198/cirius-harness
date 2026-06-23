package domain

import (
	"reflect"
	"strings"
	"testing"
)

// TestDecisionContractRendersEveryField asserts the prompt's decision schema lists every json
// field of CouncilDecision and every nested contract struct it references, so the Go contract and
// the rendered prompt cannot drift.
func TestDecisionContractRendersEveryField(t *testing.T) {
	t.Parallel()
	spec := decisionContractSpec()
	types := []reflect.Type{
		reflect.TypeFor[CouncilDecision](), reflect.TypeFor[TaskVerdict](),
	}
	for _, ty := range types {
		for _, f := range reflect.VisibleFields(ty) {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				name = f.Name
			}
			if !strings.Contains(spec, name) {
				t.Fatalf("decision contract spec missing %s field %q", ty.Name(), name)
			}
		}
	}
}
