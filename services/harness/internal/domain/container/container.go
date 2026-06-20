// Package container is the container bounded context: the Container aggregate and
// its repository port. A container is an execution environment belonging to a
// project (see internal/domain/project), a sibling to a worktree
// (see internal/domain/worktree); a session may run inside one.
package container

import (
	"errors"
	"fmt"
)

// ErrInvalidContainer is returned by Validate for a structurally invalid container.
var ErrInvalidContainer = errors.New("container: invalid")

// Status reports a container's lifecycle state.
type Status string

const (
	// StatusPending marks a container that is being provisioned.
	StatusPending Status = "pending"
	// StatusRunning marks a live container.
	StatusRunning Status = "running"
	// StatusStopped marks a container that has exited.
	StatusStopped Status = "stopped"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusStopped:
		return true
	default:
		return false
	}
}

// Container is the aggregate root describing one execution environment.
type Container struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// ProjectID is the id of the owning project.
	ProjectID string
	// Image is the container image reference (e.g. "ubuntu:24.04"); may be empty.
	Image string
	// Status reports the container's lifecycle state.
	Status Status
}

// New assembles a container from an app-minted id and its attributes, pending by
// default, and validates it. The id is supplied by the application/adapter.
func New(id, projectID, image string) (Container, error) {
	c := Container{ID: id, ProjectID: projectID, Image: image, Status: StatusPending}
	return c, c.Validate()
}

// Validate checks the container's invariants.
func (c Container) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidContainer)
	}
	if c.ProjectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidContainer)
	}
	if !c.Status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidContainer, c.Status)
	}
	return nil
}
