package project_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/project"
)

func TestNew(t *testing.T) {
	p, err := project.New("1", "harness", "/repo", project.KindMonorepo, "the control plane")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.ID != "1" || p.Kind != project.KindMonorepo {
		t.Fatalf("New = %+v, want id=1 kind=monorepo", p)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct {
		id, name, rootPath string
		kind               project.Kind
	}{
		"empty id":        {"", "harness", "/repo", project.KindSingle},
		"empty name":      {"1", "", "/repo", project.KindSingle},
		"empty root path": {"1", "harness", "", project.KindSingle},
		"bad kind":        {"1", "harness", "/repo", project.Kind("nope")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := project.New(c.id, c.name, c.rootPath, c.kind, ""); !errors.Is(err, project.ErrInvalidProject) {
				t.Fatalf("New err = %v, want ErrInvalidProject", err)
			}
		})
	}
}
