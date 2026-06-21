package domain

import (
	"errors"
	"testing"
)

func TestNewWorktree(t *testing.T) {
	t.Parallel()
	w, err := NewWorktree("/repo/wt", "p1", "main")
	if err != nil {
		t.Fatalf("NewWorktree: %v", err)
	}
	if w.id == "" {
		t.Fatal("NewWorktree should mint a non-empty id")
	}
	if w.status != WorktreeActive {
		t.Fatalf("NewWorktree status = %q, want active", w.status)
	}
}

func TestNewWorktreeInvalid(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		path      string
		projectID ProjectID
		branch    string
	}{
		"empty path":       {"", "p1", "main"},
		"empty project id": {"/repo/wt", "", "main"},
		"empty branch":     {"/repo/wt", "p1", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewWorktree(c.path, c.projectID, c.branch); !errors.Is(err, ErrInvalidWorktree) {
				t.Fatalf("NewWorktree err = %v, want ErrInvalidWorktree", err)
			}
		})
	}
}

func TestRehydrateWorktreeRejectsBadStatus(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateWorktree("1", "/repo/wt", "p1", "main", WorktreeStatus("bogus")); !errors.Is(err, ErrInvalidWorktree) {
		t.Fatalf("RehydrateWorktree err = %v, want ErrInvalidWorktree", err)
	}
}
