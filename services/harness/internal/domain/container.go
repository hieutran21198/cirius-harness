package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidContainer is returned by Validate for a structurally invalid container.
var ErrInvalidContainer = errors.New("container: invalid")

// ContainerStatus reports a container's lifecycle state.
type ContainerStatus string

const (
	// ContainerPending marks a container that is being provisioned.
	ContainerPending ContainerStatus = "pending"
	// ContainerRunning marks a live container.
	ContainerRunning ContainerStatus = "running"
	// ContainerStopped marks a container that has exited.
	ContainerStopped ContainerStatus = "stopped"
)

// Valid reports whether s is a known status.
func (s ContainerStatus) Valid() bool {
	switch s {
	case ContainerPending, ContainerRunning, ContainerStopped:
		return true
	default:
		return false
	}
}

// ContainerID is a Container's surrogate identity (a UUID v7). A named string type so a
// container id can't be silently passed where another aggregate's id is expected.
type ContainerID string

// Container is the aggregate root describing one execution environment belonging
// to a project — a sibling to a worktree; a session may run inside one.
type Container struct {
	id        ContainerID
	projectID ProjectID
	image     string
	status    ContainerStatus
}

// NewContainer assembles a fresh container from its attributes, minting its own
// identity (UUID v7), pending by default, and validates it.
func NewContainer(projectID ProjectID, image string) (Container, error) {
	c := Container{id: newID[ContainerID](), projectID: projectID, image: image, status: ContainerPending}
	return c, c.Validate()
}

// RehydrateContainer reconstitutes a Container from its persisted state (no
// creation defaults) and validates structural integrity.
func RehydrateContainer(id ContainerID, projectID ProjectID, image string, status ContainerStatus) (Container, error) {
	c := Container{id: id, projectID: projectID, image: image, status: status}
	return c, c.Validate()
}

// Validate checks the container's invariants.
func (c Container) Validate() error {
	if c.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidContainer)
	}
	if c.projectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidContainer)
	}
	if !c.status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidContainer, c.status)
	}
	return nil
}
