package domain

import (
	"reflect"
	"strings"
)

// TaskReportEnvelope is the machine-readable report a specialist agent emits when it finishes a
// task it was driven (ADR-0023). A driven worker closes its turn with this object as a single
// fenced ```json block; the coordinator extracts it (keeping the full raw output for audit) and
// submits it on the report_run frame, and the harness validates and persists it as a TaskReport.
// Council's post-execution decision stage consumes these normalized envelopes — not the raw
// conversation — so the result of every agent has one shape council can reason over.
//
// One envelope for every archetype: the common fields (status, summary, dod_met, confidence) are
// what council weighs; the optional slices (outputs, findings, verification) carry the
// archetype-specific richness (a reviewer fills findings, an implementer fills verification). As
// with OrchestrationPlan, these Go types are the single source of the contract —
// reportContractSpec() renders their JSON shape into every specialist's prompt, so the prompt and
// the types cannot drift (a test asserts every field, nested included, is rendered).
type TaskReportEnvelope struct {
	Status        string          `json:"status"`         // done | failed | blocked — the worker's self-assessed outcome
	Summary       string          `json:"summary"`        // one-paragraph result, in terms of the task's objective
	DoDMet        bool            `json:"dod_met"`        // whether the task's definition of done was met
	Confidence    string          `json:"confidence"`     // high | medium | low — how sure the worker is of the result
	Outputs       []ReportOutput  `json:"outputs"`        // the artifacts the task produced
	Findings      []ReportFinding `json:"findings"`       // assessments (review/research), most severe first
	Verification  []string        `json:"verification"`   // commands run and their result (build/test/lint)
	FollowUps     []string        `json:"follow_ups"`     // deferred work, debt incurred, or handoffs
	OpenQuestions []string        `json:"open_questions"` // what remains unknown, contested, or unverifiable
}

// ReportOutput is one artifact a task produced: its kind, a reference to it (path, id, or short
// inline value), and a one-line description.
type ReportOutput struct {
	Kind        string `json:"kind"` // patch | doc | plan | finding | report | knowledge | other
	Ref         string `json:"ref"`  // file path, id, or short inline value
	Description string `json:"description"`
}

// ReportFinding is one assessment a task surfaced (a reviewer's or researcher's deliverable): how
// severe, where, what it is, and a suggested fix.
type ReportFinding struct {
	Severity   string `json:"severity"` // blocking | major | minor | nit
	Location   string `json:"location"` // file:line or component
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
}

// reportStatuses are the valid self-assessed outcomes a worker may report.
var reportStatuses = map[string]bool{"done": true, "failed": true, "blocked": true}

// reportConfidences are the valid confidence levels a worker may report.
var reportConfidences = map[string]bool{"high": true, "medium": true, "low": true}

// reportContractSpec renders the required JSON report format into a specialist's prompt, derived
// by reflection from the TaskReportEnvelope type (and every nested struct it references) so the
// contract has one source. It reuses writeShape/structElem (see plan_contract.go).
func reportContractSpec() string {
	var b strings.Builder
	b.WriteString("When you finish, close your turn with your structured report as a SINGLE fenced " +
		"```json block containing one object matching TaskReportEnvelope, and nothing else in that " +
		"block. Put your human-readable deliverable above it; the JSON is what the harness captures " +
		"and what council reads. status is your self-assessed outcome (done | failed | blocked); " +
		"confidence is high | medium | low. Fill the optional arrays only where they apply to your " +
		"work (a reviewer fills findings; an implementer fills verification). The shapes (a name " +
		"after a field is its nested shape, listed below):\n")
	writeShape(&b, "TaskReportEnvelope", reflect.TypeFor[TaskReportEnvelope](), map[string]bool{})
	return b.String()
}
