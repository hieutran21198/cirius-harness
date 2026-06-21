package domain

import (
	"errors"
	"testing"
)

func TestNewContainer(t *testing.T) {
	t.Parallel()
	c, err := NewContainer("p1", "ubuntu:24.04")
	if err != nil {
		t.Fatalf("NewContainer: %v", err)
	}
	if c.id == "" {
		t.Fatal("NewContainer should mint a non-empty id")
	}
	if c.status != ContainerPending {
		t.Fatalf("NewContainer status = %q, want pending", c.status)
	}
}

func TestNewContainerInvalid(t *testing.T) {
	t.Parallel()
	if _, err := NewContainer("", "ubuntu:24.04"); !errors.Is(err, ErrInvalidContainer) {
		t.Fatalf("NewContainer with empty project id err = %v, want ErrInvalidContainer", err)
	}
}

func TestRehydrateContainerRejectsBadStatus(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateContainer("1", "p1", "", ContainerStatus("bogus")); !errors.Is(err, ErrInvalidContainer) {
		t.Fatalf("RehydrateContainer err = %v, want ErrInvalidContainer", err)
	}
}

func TestRehydrateContainerRejectsEmptyID(t *testing.T) {
	t.Parallel()
	if _, err := RehydrateContainer("", "p1", "", ContainerPending); !errors.Is(err, ErrInvalidContainer) {
		t.Fatalf("RehydrateContainer with empty id err = %v, want ErrInvalidContainer", err)
	}
}
