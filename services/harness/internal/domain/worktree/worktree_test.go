package worktree_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/worktree"
)

func TestNew(t *testing.T) {
	w, err := worktree.New("1", "/repo/.wt/feature", "p1", "feature")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if w.Status != worktree.StatusActive {
		t.Fatalf("New status = %q, want active", w.Status)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct{ id, path, projectID, branch string }{
		"empty id":         {"", "/wt", "p1", "main"},
		"empty path":       {"1", "", "p1", "main"},
		"empty project id": {"1", "/wt", "", "main"},
		"empty branch":     {"1", "/wt", "p1", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := worktree.New(c.id, c.path, c.projectID, c.branch); !errors.Is(err, worktree.ErrInvalidWorktree) {
				t.Fatalf("New err = %v, want ErrInvalidWorktree", err)
			}
		})
	}
}
