package domain

import (
	"context"
	"errors"
)

// ErrPlanRunNotFound is returned by a PlanRunReader when a plan has no run yet.
var ErrPlanRunNotFound = errors.New("plan run: not found")

// PlanRunReader reads the run for a plan. A domain-owned driven port: the command that records
// progress loads the existing run (or seeds a new one) before mutating it.
type PlanRunReader interface {
	// FindByPlan returns the run for the given plan, or ErrPlanRunNotFound if none yet.
	FindByPlan(ctx context.Context, planID PlanID) (PlanRun, error)
}
