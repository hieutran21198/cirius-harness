// Package session is the session bounded context: the Session aggregate (with
// its Member join) and the Sessions repository port. A session is one run of the
// harness scoped to a project (see internal/domain/project), executed in an
// environment that is a container, a worktree, or not yet assigned; its Members
// record which agents joined the run and the model each ran with.
package session

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidSession is returned by Validate for a structurally invalid session.
var ErrInvalidSession = errors.New("session: invalid")

// Session is the aggregate root describing one run of the harness, scoped to a
// project and executed in a (container | worktree | unset) environment. The ID is
// a generated UUID v7 — a session has no natural key.
type Session struct {
	// ID is the session's identifier (UUID v7).
	ID string
	// ProjectID is the id of the project the session operates on.
	ProjectID string
	// EnvType is the kind of environment the session runs in.
	EnvType EnvType
	// EnvID is the polymorphic id of the environment (a container id or worktree
	// id), keyed by EnvType; empty when EnvType is EnvUnset. It has no foreign key.
	EnvID string
	// Title is a human-facing label or goal for the session.
	Title string
	// Status tracks the session through its lifecycle.
	Status Status
	// CreatedAt is when the session was created.
	CreatedAt time.Time
	// StartedAt is when the session began running; nil until started.
	StartedAt *time.Time
	// EndedAt is when the session reached a terminal state; nil until ended.
	EndedAt *time.Time
	// Members lists the agents that joined the session.
	Members []Member
}

// Validate checks the session's invariants, including those of its members.
func (s Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidSession)
	}
	if s.ProjectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidSession)
	}
	if !s.EnvType.Valid() {
		return fmt.Errorf("%w: unknown env type %q", ErrInvalidSession, s.EnvType)
	}
	if s.EnvType == EnvUnset && s.EnvID != "" {
		return fmt.Errorf("%w: env id must be empty when env type is unset", ErrInvalidSession)
	}
	if s.EnvType != EnvUnset && s.EnvID == "" {
		return fmt.Errorf("%w: env id is required when env type is %q", ErrInvalidSession, s.EnvType)
	}
	if !s.Status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidSession, s.Status)
	}
	for _, m := range s.Members {
		if err := m.Validate(); err != nil {
			return err
		}
	}
	return nil
}
