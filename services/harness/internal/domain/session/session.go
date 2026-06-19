// Package session is the session bounded context: the Session aggregate (with
// its Member join) and the Sessions repository port. A session is one run of the
// harness inside a worktree (see internal/domain/worktree); its Members record
// which agents joined the run.
package session

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidSession is returned by Validate for a structurally invalid session.
var ErrInvalidSession = errors.New("session: invalid")

// Session is the aggregate root describing one run of the harness in a worktree.
// The ID is a generated UUID v7 — a session has no natural key.
type Session struct {
	// ID is the session's identifier (UUID v7).
	ID string
	// Worktree is the path of the worktree the session runs in (its natural key).
	Worktree string
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
	if s.Worktree == "" {
		return fmt.Errorf("%w: worktree is required", ErrInvalidSession)
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
