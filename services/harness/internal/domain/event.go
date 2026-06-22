package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidEvent is returned by Validate for a structurally invalid event.
var ErrInvalidEvent = errors.New("event: invalid")

// Event status values.
const (
	EventOK    = "ok"
	EventError = "error"
)

// EventID is an Event's surrogate identity (a UUID v7). A named string type so an event
// id can't be silently passed where another aggregate's id is expected.
type EventID string

// Event is an append-only audit record of something the harness did — one row in the
// captured log (distinct from the ephemeral stderr logs). It records what happened
// (kind), who caused it (actor), how it ended (status), a human message, and an
// optional JSON detail. Events are written, never updated or deleted.
type Event struct {
	id         EventID
	occurredAt time.Time
	kind       string
	actor      string
	status     string
	message    string
	detail     string
}

// NewEvent assembles a fresh audit record, minting its own identity (UUID v7), and
// validates it. occurredAt is supplied by the caller (no clock in the domain).
func NewEvent(kind, actor, status, message, detail string, occurredAt time.Time) (Event, error) {
	e := Event{
		id:         newID[EventID](),
		occurredAt: occurredAt,
		kind:       kind,
		actor:      actor,
		status:     status,
		message:    message,
		detail:     detail,
	}
	return e, e.Validate()
}

// RehydrateEvent reconstitutes an Event from its persisted state and validates it.
func RehydrateEvent(id EventID, occurredAt time.Time, kind, actor, status, message, detail string) (Event, error) {
	e := Event{
		id:         id,
		occurredAt: occurredAt,
		kind:       kind,
		actor:      actor,
		status:     status,
		message:    message,
		detail:     detail,
	}
	return e, e.Validate()
}

// Validate checks the event's invariants.
func (e Event) Validate() error {
	if e.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidEvent)
	}
	if e.kind == "" {
		return fmt.Errorf("%w: kind is required", ErrInvalidEvent)
	}
	if e.status == "" {
		return fmt.Errorf("%w: status is required", ErrInvalidEvent)
	}
	if e.occurredAt.IsZero() {
		return fmt.Errorf("%w: occurredAt is required", ErrInvalidEvent)
	}
	return nil
}

// EventSnapshot is the persistence grouped view of an Event. It is the only way an
// Event's state leaves the domain; the repository maps it to a row.
type EventSnapshot struct {
	ID         EventID
	OccurredAt time.Time
	Kind       string
	Actor      string
	Status     string
	Message    string
	Detail     string
}

// Snapshot returns the event's persistence view.
func (e Event) Snapshot() EventSnapshot {
	return EventSnapshot{
		ID:         e.id,
		OccurredAt: e.occurredAt,
		Kind:       e.kind,
		Actor:      e.actor,
		Status:     e.status,
		Message:    e.message,
		Detail:     e.detail,
	}
}
