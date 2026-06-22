package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidSession is returned by Validate for a structurally invalid session.
var ErrInvalidSession = errors.New("session: invalid")

// SessionID is a Session's surrogate identity (a UUID v7). A named string type so a
// session id can't be silently passed where another aggregate's id is expected.
type SessionID string

// Session is the aggregate root describing one run of the harness, scoped to a
// project and executed in a (container | worktree | unset) environment. The id is
// a generated UUID v7 — a session has no natural key. Its members record which
// agents joined the run and the model each ran with.
type Session struct {
	id        SessionID
	projectID ProjectID
	envType   EnvType
	// envID is a polymorphic reference — a WorktreeID or a ContainerID selected by
	// envType (empty when EnvUnset). It stays a plain string because a single field
	// cannot carry both id types; integrity is enforced in Validate, not by an FK.
	envID     string
	title     string
	status    SessionStatus
	createdAt time.Time
	startedAt *time.Time
	endedAt   *time.Time
	members   []Member
}

// NewSession assembles a fresh run of the harness scoped to a project, minting its
// own identity (UUID v7), pending and not yet provisioned (EnvUnset), and validates
// it. createdAt is supplied by the application/adapter (no clock in the domain);
// members are added by later use cases.
func NewSession(projectID ProjectID, title string, createdAt time.Time) (Session, error) {
	s := Session{
		id:        newID[SessionID](),
		projectID: projectID,
		envType:   EnvUnset,
		title:     title,
		status:    SessionPending,
		createdAt: createdAt,
	}
	return s, s.Validate()
}

// RehydrateSession reconstitutes a Session and its members from persisted state (no
// creation defaults) and validates structural integrity.
func RehydrateSession(id SessionID, projectID ProjectID, envType EnvType, envID, title string, status SessionStatus, createdAt time.Time, startedAt, endedAt *time.Time, members []Member) (Session, error) {
	s := Session{
		id:        id,
		projectID: projectID,
		envType:   envType,
		envID:     envID,
		title:     title,
		status:    status,
		createdAt: createdAt,
		startedAt: startedAt,
		endedAt:   endedAt,
		members:   members,
	}
	return s, s.Validate()
}

// Validate checks the session's invariants, including those of its members.
func (s Session) Validate() error {
	if s.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidSession)
	}
	if s.projectID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidSession)
	}
	if !s.envType.Valid() {
		return fmt.Errorf("%w: unknown env type %q", ErrInvalidSession, s.envType)
	}
	if s.envType == EnvUnset && s.envID != "" {
		return fmt.Errorf("%w: env id must be empty when env type is unset", ErrInvalidSession)
	}
	if s.envType != EnvUnset && s.envID == "" {
		return fmt.Errorf("%w: env id is required when env type is %q", ErrInvalidSession, s.envType)
	}
	if !s.status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidSession, s.status)
	}
	for _, m := range s.members {
		if err := m.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SessionSnapshot is the persistence grouped view of a Session and its members.
type SessionSnapshot struct {
	ID        SessionID
	ProjectID ProjectID
	EnvType   EnvType
	EnvID     string
	Title     string
	Status    SessionStatus
	CreatedAt time.Time
	StartedAt *time.Time
	EndedAt   *time.Time
	Members   []MemberSnapshot
}

// Snapshot returns the session's persistence view, including its members.
func (s Session) Snapshot() SessionSnapshot {
	members := make([]MemberSnapshot, len(s.members))
	for i, m := range s.members {
		members[i] = m.Snapshot()
	}
	return SessionSnapshot{
		ID:        s.id,
		ProjectID: s.projectID,
		EnvType:   s.envType,
		EnvID:     s.envID,
		Title:     s.title,
		Status:    s.status,
		CreatedAt: s.createdAt,
		StartedAt: s.startedAt,
		EndedAt:   s.endedAt,
		Members:   members,
	}
}
