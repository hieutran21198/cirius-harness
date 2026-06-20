package container_test

import (
	"errors"
	"testing"

	"harness-workspace/services/harness/internal/domain/container"
)

func TestNew(t *testing.T) {
	c, err := container.New("1", "p1", "ubuntu:24.04")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Status != container.StatusPending {
		t.Fatalf("New status = %q, want pending", c.Status)
	}
}

func TestNewInvalid(t *testing.T) {
	cases := map[string]struct{ id, projectID string }{
		"empty id":         {"", "p1"},
		"empty project id": {"1", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := container.New(c.id, c.projectID, ""); !errors.Is(err, container.ErrInvalidContainer) {
				t.Fatalf("New err = %v, want ErrInvalidContainer", err)
			}
		})
	}
}
