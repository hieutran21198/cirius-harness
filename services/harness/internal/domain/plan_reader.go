package domain

import (
	"context"
	"errors"
)

// ErrPlanNotFound is returned by a PlanReader when no plan matches the lookup.
var ErrPlanNotFound = errors.New("plan: not found")

// PlanReader reads persisted plans. A domain-owned driven port (ADR-0013): the methods a query
// needs to fetch a plan to drive. It is obtained from a ReadStore and implemented by the infra
// adapter.
type PlanReader interface {
	// FindByID returns the plan with the given id, or ErrPlanNotFound if none.
	FindByID(ctx context.Context, id PlanID) (Plan, error)
	// LatestForSession returns the most recently created plan produced in the session, or
	// ErrPlanNotFound if the session has none.
	LatestForSession(ctx context.Context, sessionID SessionID) (Plan, error)
}
