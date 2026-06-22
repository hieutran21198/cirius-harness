package domain

import "context"

// EventWriter appends to the audit log. It is a domain-owned driven port obtained from
// a UnitOfWork and implemented by the infra adapter (ADR-0013). The log is append-only,
// so the only operation is Append.
type EventWriter interface {
	// Append writes one audit event.
	Append(ctx context.Context, e Event) error
}
