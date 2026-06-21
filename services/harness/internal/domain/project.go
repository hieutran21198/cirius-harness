package domain

import (
	"errors"
	"fmt"
)

// ErrInvalidProject is returned by Validate for a structurally invalid project.
var ErrInvalidProject = errors.New("project: invalid")

// ProjectID is a Project's surrogate identity (a UUID v7). A named string type so a
// project id can't be silently passed where another aggregate's id is expected; sessions,
// worktrees, and containers reference their project by ProjectID.
type ProjectID string

// Project is the aggregate root describing one codebase the harness operates on;
// a monorepo is just a Kind. Worktrees belong to a project.
type Project struct {
	id          ProjectID
	name        string
	rootPath    string
	kind        Kind
	description string
}

// NewProject assembles a fresh project from its attributes, minting its own identity
// (UUID v7), and validates it.
func NewProject(name, rootPath string, kind Kind, description string) (Project, error) {
	p := Project{id: newID[ProjectID](), name: name, rootPath: rootPath, kind: kind, description: description}
	return p, p.Validate()
}

// RehydrateProject reconstitutes a Project from its persisted state (no creation
// defaults) and validates structural integrity.
func RehydrateProject(id ProjectID, name, rootPath string, kind Kind, description string) (Project, error) {
	p := Project{id: id, name: name, rootPath: rootPath, kind: kind, description: description}
	return p, p.Validate()
}

// Validate checks the project's invariants.
func (p Project) Validate() error {
	if p.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidProject)
	}
	if p.name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidProject)
	}
	if p.rootPath == "" {
		return fmt.Errorf("%w: root path is required", ErrInvalidProject)
	}
	if !p.kind.Valid() {
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidProject, p.kind)
	}
	return nil
}
