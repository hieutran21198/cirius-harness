package domain

import (
	"reflect"
	"strings"
	"testing"
)

// TestReportContractRendersEveryField asserts the prompt's report schema lists every json field
// of TaskReportEnvelope and every nested contract struct it references, so the Go contract and the
// rendered prompt cannot drift.
func TestReportContractRendersEveryField(t *testing.T) {
	t.Parallel()
	spec := reportContractSpec()
	types := []reflect.Type{
		reflect.TypeFor[TaskReportEnvelope](), reflect.TypeFor[ReportOutput](),
		reflect.TypeFor[ReportFinding](),
	}
	for _, ty := range types {
		for _, f := range reflect.VisibleFields(ty) {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				name = f.Name
			}
			if !strings.Contains(spec, name) {
				t.Fatalf("report contract spec missing %s field %q", ty.Name(), name)
			}
		}
	}
}
