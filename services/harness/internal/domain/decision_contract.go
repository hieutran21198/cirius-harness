package domain

import (
	"reflect"
	"strings"
)

// CouncilDecision is the machine-readable verdict council emits after a plan's drive completes
// (ADR-0023): the post-execution counterpart to OrchestrationPlan. Council consumes the
// normalized TaskReportEnvelopes the workers produced and weighs them against the plan's
// definition of done, then emits this object as a single fenced ```json block. The Pi extension
// captures it and submits it over the submit_decision frame; the harness validates it into a
// domain.Decision and persists it. As with the plan contract, these Go types are the single
// source — decisionContractSpec() renders their JSON shape into council's prompt so the two
// cannot drift.
type CouncilDecision struct {
	Verdict     string        `json:"verdict"`      // accept | iterate | escalate | reject
	Summary     string        `json:"summary"`      // overall judgement of the drive against the plan's goal
	DoDMet      bool          `json:"dod_met"`      // whether the plan's definition of done was met
	Tasks       []TaskVerdict `json:"tasks"`        // per-task verdict over the reports
	FollowUps   []string      `json:"follow_ups"`   // debt, deferrals, or handoffs to carry forward
	NextActions []string      `json:"next_actions"` // what to do next (re-drive tasks, escalate, close)
}

// TaskVerdict is council's judgement of one task's reported result.
type TaskVerdict struct {
	Ref       string `json:"ref"`       // the plan-local task ref (T1)
	Verdict   string `json:"verdict"`   // accept | iterate | escalate | reject
	Rationale string `json:"rationale"` // why, grounded in the task's report
}

// decisionVerdicts are the valid verdict values for a decision and each task verdict.
var decisionVerdicts = map[string]bool{"accept": true, "iterate": true, "escalate": true, "reject": true}

// decisionContractSpec renders the required JSON decision format into council's prompt, derived by
// reflection from the CouncilDecision type (and every nested struct it references) so the contract
// has one source. It reuses writeShape/structElem (see plan_contract.go).
func decisionContractSpec() string {
	var b strings.Builder
	b.WriteString("When you emit the decision, output it as a SINGLE fenced ```json block " +
		"containing one object matching CouncilDecision, and nothing else in that block. The harness " +
		"captures and persists it. verdict is accept | iterate | escalate | reject. Ground every task " +
		"verdict in that task's reported result. The shapes (a name after a field is its nested shape, " +
		"listed below):\n")
	writeShape(&b, "CouncilDecision", reflect.TypeFor[CouncilDecision](), map[string]bool{})
	return b.String()
}
