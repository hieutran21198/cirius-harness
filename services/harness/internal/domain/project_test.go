package domain

import (
	"errors"
	"testing"
)

func TestNewProject(t *testing.T) {
	t.Parallel()
	p, err := NewProject("harness", "/repo", KindMonorepo, "the control plane")
	if err != nil {
		t.Fatalf("NewProject: %v", err)
	}
	if p.id == "" || p.name != "harness" || p.kind != KindMonorepo {
		t.Fatalf("NewProject = %+v", p)
	}
}

func TestNewProjectInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		name, root string
		kind       Kind
	}{
		"empty name": {"", "/repo", KindSingle},
		"empty root": {"harness", "", KindSingle},
		"bad kind":   {"harness", "/repo", Kind("bogus")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewProject(c.name, c.root, c.kind, ""); !errors.Is(err, ErrInvalidProject) {
				t.Fatalf("NewProject err = %v, want ErrInvalidProject", err)
			}
		})
	}
}

func TestRehydrateProjectRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateProject("", "harness", "/repo", KindSingle, ""); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("RehydrateProject with empty id err = %v, want ErrInvalidProject", err)
	}
}
