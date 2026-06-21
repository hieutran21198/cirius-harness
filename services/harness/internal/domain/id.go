package domain

import "github.com/google/uuid"

// newID mints a fresh aggregate identity — a UUID v7 (time-ordered, so rows sort by
// creation) typed as the caller's id type T (e.g. newID[ModelID]()). The id format is a
// domain identity policy, owned here rather than by the caller: each NewXxx constructor
// calls newID[XxxID]() so fresh aggregates own their identity, minted in-process before any
// write (never DB-generated — ADR-0005). The crypto/rand source failing is a truly impossible
// branch, so uuid.Must's panic is acceptable (conventions/go.md "Errors").
func newID[T ~string]() T { return T(uuid.Must(uuid.NewV7()).String()) }
