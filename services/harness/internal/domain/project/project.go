// Package project is the project bounded context: the Project aggregate and the
// Projects repository port. A project is the codebase the harness operates on;
// a monorepo is just a Kind. Worktrees (see internal/domain/worktree) belong to
// a project.
package project

import (
	"errors"
	"fmt"
)

// ErrInvalidProject is returned by Validate for a structurally invalid project.
var ErrInvalidProject = errors.New("project: invalid")

// Project is the aggregate root describing one codebase the harness operates on.
type Project struct {
	// ID is the surrogate identity (UUID v7), assigned by the application/adapter.
	ID string
	// Name is the project's unique business key.
	Name string
	// RootPath is the absolute filesystem path of the project root (unique).
	RootPath string
	// Kind records whether the project is a single repo or a monorepo.
	Kind Kind
	// Description is a human-facing summary of the project.
	Description string
}

// New assembles a project from an app-minted id and its attributes and validates
// it. The id is supplied by the application/adapter.
func New(id, name, rootPath string, kind Kind, description string) (Project, error) {
	p := Project{ID: id, Name: name, RootPath: rootPath, Kind: kind, Description: description}
	return p, p.Validate()
}

// Validate checks the project's invariants.
func (p Project) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidProject)
	}
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidProject)
	}
	if p.RootPath == "" {
		return fmt.Errorf("%w: root path is required", ErrInvalidProject)
	}
	if !p.Kind.Valid() {
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidProject, p.Kind)
	}
	return nil
}
