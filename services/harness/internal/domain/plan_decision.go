package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidPlanDecision is returned for a structurally invalid council decision.
var ErrInvalidPlanDecision = errors.New("plan decision: invalid")

// PlanDecisionID is a PlanDecision's surrogate identity (UUID v7).
type PlanDecisionID string

// PlanDecision is council's persisted post-execution verdict over a run (ADR-0023): the validated
// CouncilDecision council emitted after consuming the run's task reports. Append-only — each
// iteration of a drive records a new decision; the latest by created_at is the current verdict.
// (Distinct from domain.Decision, the authorization outcome.)
type PlanDecision struct {
	id        PlanDecisionID
	planRunID PlanRunID
	decision  CouncilDecision
	createdAt time.Time
}

// NewPlanDecision builds a decision for a run, minting its identity, and validates it.
func NewPlanDecision(planRunID PlanRunID, src CouncilDecision, now time.Time) (PlanDecision, error) {
	d := PlanDecision{
		id:        newID[PlanDecisionID](),
		planRunID: planRunID,
		decision:  src,
		createdAt: now,
	}
	return d, d.Validate()
}

// RehydratePlanDecision reconstitutes a decision from a persisted snapshot and validates it.
func RehydratePlanDecision(snap PlanDecisionSnapshot) (PlanDecision, error) {
	d := PlanDecision{id: snap.ID, planRunID: snap.PlanRunID, decision: snap.Decision, createdAt: snap.CreatedAt}
	return d, d.Validate()
}

// Validate checks the decision's invariants: ids present, a known top-level verdict, a non-empty
// summary, and a known verdict with a ref on every per-task verdict.
func (d PlanDecision) Validate() error {
	if d.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidPlanDecision)
	}
	if d.planRunID == "" {
		return fmt.Errorf("%w: plan run id is required", ErrInvalidPlanDecision)
	}
	if !decisionVerdicts[d.decision.Verdict] {
		return fmt.Errorf("%w: unknown verdict %q", ErrInvalidPlanDecision, d.decision.Verdict)
	}
	if d.decision.Summary == "" {
		return fmt.Errorf("%w: summary is required", ErrInvalidPlanDecision)
	}
	for _, tv := range d.decision.Tasks {
		if tv.Ref == "" {
			return fmt.Errorf("%w: task verdict ref is required", ErrInvalidPlanDecision)
		}
		if !decisionVerdicts[tv.Verdict] {
			return fmt.Errorf("%w: task %q unknown verdict %q", ErrInvalidPlanDecision, tv.Ref, tv.Verdict)
		}
	}
	return nil
}

// PlanDecisionSnapshot is the persistence grouped view of a PlanDecision.
type PlanDecisionSnapshot struct {
	ID        PlanDecisionID
	PlanRunID PlanRunID
	Decision  CouncilDecision
	CreatedAt time.Time
}

// Snapshot returns the decision's full persistence view.
func (d PlanDecision) Snapshot() PlanDecisionSnapshot {
	return PlanDecisionSnapshot{ID: d.id, PlanRunID: d.planRunID, Decision: d.decision, CreatedAt: d.createdAt}
}
