package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidPlan is returned by Validate for a structurally invalid plan.
var ErrInvalidPlan = errors.New("plan: invalid")

// Plan-side surrogate identities (UUID v7). Named string types so one aggregate's id can't be
// silently passed where another's is expected.
type (
	// PlanID is a Plan's surrogate identity.
	PlanID string
	// PlanTaskID is a PlanTask's surrogate identity.
	PlanTaskID string
	// PlanWaveID is a PlanWave's surrogate identity.
	PlanWaveID string
	// PlanRiskID is a PlanRisk's surrogate identity.
	PlanRiskID string
	// PlanApprovalID is a PlanApproval's surrogate identity.
	PlanApprovalID string
)

// PlanStatus is where a persisted plan is in its lifecycle. A submitted plan starts planned
// (or approved if council says so); the remaining values belong to the future executor.
type PlanStatus string

// Plan lifecycle states.
const (
	PlanPlanned   PlanStatus = "planned"
	PlanApproved  PlanStatus = "approved"
	PlanDriving   PlanStatus = "driving"
	PlanDone      PlanStatus = "done"
	PlanCancelled PlanStatus = "cancelled"
)

// Valid reports whether s is a known plan status.
func (s PlanStatus) Valid() bool {
	switch s {
	case PlanPlanned, PlanApproved, PlanDriving, PlanDone, PlanCancelled:
		return true
	}
	return false
}

// Plan is the aggregate root: a council-produced orchestration plan, persisted after a human
// approves it (ADR-0019). It captures the request analysis (intent, goal, scope, assumptions,
// risks), the task DAG (tasks grouped into waves), the human-approval gates, and a closing
// report. It is attached to the session it was produced in. A future executor drives it; the
// harness only records it.
type Plan struct {
	id          PlanID
	sessionID   SessionID // optional: the session it was produced in ("" if none)
	agent       string    // the agent that produced it (council)
	intent      string
	goal        string
	status      PlanStatus
	createdAt   time.Time
	scope       Scope
	assumptions []string
	report      Report
	tasks       []PlanTask
	risks       []PlanRisk
	approvals   []PlanApproval
	waves       []PlanWave
}

// PlanTask is one node of the plan's DAG. ref is the plan-local id ("T1") that dependsOn and a
// wave's task list reference.
type PlanTask struct {
	id             PlanTaskID
	ref            string
	category       Category
	assigneeAgent  string
	assigneeLens   string
	objective      string
	inputs         []string
	expectedOutput string
	dependsOn      []string
	dod            []string
	gate           string
	riskLevel      string
}

// PlanWave groups task refs that can run concurrently — one rung of the dependency-ordered DAG.
type PlanWave struct {
	id       PlanWaveID
	number   int
	taskRefs []string
}

// PlanRisk is one weighed risk on the plan.
type PlanRisk struct {
	id          PlanRiskID
	level       string
	description string
}

// PlanApproval is a human gate the plan requires before a given task is driven.
type PlanApproval struct {
	id             PlanApprovalID
	kind           string
	requiredBefore string
	reason         string
	question       string
}

// NewPlan maps a council-emitted OrchestrationPlan into a fresh aggregate, minting identities
// for the plan and each child, deriving the status from the plan's report, and validating the
// whole DAG. createdAt is supplied by the caller (no clock in the domain). sessionID is optional.
func NewPlan(sessionID SessionID, agent string, src OrchestrationPlan, createdAt time.Time) (Plan, error) {
	tasks := make([]PlanTask, len(src.Tasks))
	for i, t := range src.Tasks {
		tasks[i] = PlanTask{
			id:             newID[PlanTaskID](),
			ref:            t.ID,
			category:       t.Category,
			assigneeAgent:  t.Assignee.Agent,
			assigneeLens:   t.Assignee.Lens,
			objective:      t.Objective,
			inputs:         t.Inputs,
			expectedOutput: t.ExpectedOutput,
			dependsOn:      t.DependsOn,
			dod:            t.DoD,
			gate:           t.Gate,
			riskLevel:      t.RiskLevel,
		}
	}
	risks := make([]PlanRisk, len(src.Risks))
	for i, r := range src.Risks {
		risks[i] = PlanRisk{id: newID[PlanRiskID](), level: r.Level, description: r.Description}
	}
	approvals := make([]PlanApproval, len(src.Approvals))
	for i, a := range src.Approvals {
		approvals[i] = PlanApproval{
			id: newID[PlanApprovalID](), kind: a.Type, requiredBefore: a.RequiredBefore,
			reason: a.Reason, question: a.Question,
		}
	}
	waves := make([]PlanWave, len(src.Waves))
	for i, w := range src.Waves {
		waves[i] = PlanWave{id: newID[PlanWaveID](), number: w.Wave, taskRefs: w.Tasks}
	}

	p := Plan{
		id:          newID[PlanID](),
		sessionID:   sessionID,
		agent:       agent,
		intent:      src.Intent,
		goal:        src.Goal,
		status:      planStatusFrom(src.Report.Status),
		createdAt:   createdAt,
		scope:       src.Scope,
		assumptions: src.Assumptions,
		report:      src.Report,
		tasks:       tasks,
		risks:       risks,
		approvals:   approvals,
		waves:       waves,
	}
	return p, p.Validate()
}

// RehydratePlan reconstitutes a Plan and its children from a persisted snapshot and validates it.
func RehydratePlan(snap PlanSnapshot) (Plan, error) {
	tasks := make([]PlanTask, len(snap.Tasks))
	for i, t := range snap.Tasks {
		tasks[i] = PlanTask{
			id: t.ID, ref: t.Ref, category: t.Category, assigneeAgent: t.AssigneeAgent,
			assigneeLens: t.AssigneeLens, objective: t.Objective, inputs: t.Inputs,
			expectedOutput: t.ExpectedOutput, dependsOn: t.DependsOn, dod: t.DoD,
			gate: t.Gate, riskLevel: t.RiskLevel,
		}
	}
	risks := make([]PlanRisk, len(snap.Risks))
	for i, r := range snap.Risks {
		risks[i] = PlanRisk{id: r.ID, level: r.Level, description: r.Description}
	}
	approvals := make([]PlanApproval, len(snap.Approvals))
	for i, a := range snap.Approvals {
		approvals[i] = PlanApproval{
			id: a.ID, kind: a.Kind, requiredBefore: a.RequiredBefore, reason: a.Reason, question: a.Question,
		}
	}
	waves := make([]PlanWave, len(snap.Waves))
	for i, w := range snap.Waves {
		waves[i] = PlanWave{id: w.ID, number: w.Number, taskRefs: w.TaskRefs}
	}
	p := Plan{
		id: snap.ID, sessionID: snap.SessionID, agent: snap.Agent, intent: snap.Intent,
		goal: snap.Goal, status: snap.Status, createdAt: snap.CreatedAt, scope: snap.Scope,
		assumptions: snap.Assumptions, report: snap.Report, tasks: tasks, risks: risks,
		approvals: approvals, waves: waves,
	}
	return p, p.Validate()
}

// planStatusFrom normalises council's free-text report status to a known PlanStatus,
// defaulting to planned.
func planStatusFrom(s string) PlanStatus {
	if st := PlanStatus(strings.ToLower(strings.TrimSpace(s))); st.Valid() {
		return st
	}
	return PlanPlanned
}

// Validate checks the plan's invariants and the integrity of its task DAG: every task has a
// unique non-empty ref, and every dependency and wave-membership references a known task.
func (p Plan) Validate() error {
	if p.id == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidPlan)
	}
	if p.agent == "" {
		return fmt.Errorf("%w: agent is required", ErrInvalidPlan)
	}
	if p.intent == "" {
		return fmt.Errorf("%w: intent is required", ErrInvalidPlan)
	}
	if !p.status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidPlan, p.status)
	}
	if len(p.tasks) == 0 {
		return fmt.Errorf("%w: a plan needs at least one task", ErrInvalidPlan)
	}
	refs := make(map[string]bool, len(p.tasks))
	for _, t := range p.tasks {
		if t.ref == "" {
			return fmt.Errorf("%w: task ref is required", ErrInvalidPlan)
		}
		if refs[t.ref] {
			return fmt.Errorf("%w: duplicate task ref %q", ErrInvalidPlan, t.ref)
		}
		refs[t.ref] = true
	}
	for _, t := range p.tasks {
		for _, dep := range t.dependsOn {
			if !refs[dep] {
				return fmt.Errorf("%w: task %q depends on unknown task %q", ErrInvalidPlan, t.ref, dep)
			}
		}
	}
	for _, w := range p.waves {
		for _, ref := range w.taskRefs {
			if !refs[ref] {
				return fmt.Errorf("%w: wave %d references unknown task %q", ErrInvalidPlan, w.number, ref)
			}
		}
	}
	return nil
}

// TaskCount reports how many tasks the plan holds (the submit result echoes it).
func (p Plan) TaskCount() int { return len(p.tasks) }

// PlanSnapshot is the persistence grouped view of a Plan and all its children. It is the only
// way a Plan's state leaves the domain; the repository maps it to rows across the plan tables.
type PlanSnapshot struct {
	ID          PlanID
	SessionID   SessionID
	Agent       string
	Intent      string
	Goal        string
	Status      PlanStatus
	CreatedAt   time.Time
	Scope       Scope
	Assumptions []string
	Report      Report
	Tasks       []PlanTaskSnapshot
	Risks       []PlanRiskSnapshot
	Approvals   []PlanApprovalSnapshot
	Waves       []PlanWaveSnapshot
}

// PlanTaskSnapshot is the persistence view of a PlanTask.
type PlanTaskSnapshot struct {
	ID             PlanTaskID
	Ref            string
	Category       Category
	AssigneeAgent  string
	AssigneeLens   string
	Objective      string
	Inputs         []string
	ExpectedOutput string
	DependsOn      []string
	DoD            []string
	Gate           string
	RiskLevel      string
}

// PlanWaveSnapshot is the persistence view of a PlanWave.
type PlanWaveSnapshot struct {
	ID       PlanWaveID
	Number   int
	TaskRefs []string
}

// PlanRiskSnapshot is the persistence view of a PlanRisk.
type PlanRiskSnapshot struct {
	ID          PlanRiskID
	Level       string
	Description string
}

// PlanApprovalSnapshot is the persistence view of a PlanApproval.
type PlanApprovalSnapshot struct {
	ID             PlanApprovalID
	Kind           string
	RequiredBefore string
	Reason         string
	Question       string
}

// Snapshot returns the plan's full persistence view, including every child.
func (p Plan) Snapshot() PlanSnapshot {
	tasks := make([]PlanTaskSnapshot, len(p.tasks))
	for i, t := range p.tasks {
		tasks[i] = PlanTaskSnapshot{
			ID: t.id, Ref: t.ref, Category: t.category, AssigneeAgent: t.assigneeAgent,
			AssigneeLens: t.assigneeLens, Objective: t.objective, Inputs: t.inputs,
			ExpectedOutput: t.expectedOutput, DependsOn: t.dependsOn, DoD: t.dod,
			Gate: t.gate, RiskLevel: t.riskLevel,
		}
	}
	risks := make([]PlanRiskSnapshot, len(p.risks))
	for i, r := range p.risks {
		risks[i] = PlanRiskSnapshot{ID: r.id, Level: r.level, Description: r.description}
	}
	approvals := make([]PlanApprovalSnapshot, len(p.approvals))
	for i, a := range p.approvals {
		approvals[i] = PlanApprovalSnapshot{
			ID: a.id, Kind: a.kind, RequiredBefore: a.requiredBefore, Reason: a.reason, Question: a.question,
		}
	}
	waves := make([]PlanWaveSnapshot, len(p.waves))
	for i, w := range p.waves {
		waves[i] = PlanWaveSnapshot{ID: w.id, Number: w.number, TaskRefs: w.taskRefs}
	}
	return PlanSnapshot{
		ID: p.id, SessionID: p.sessionID, Agent: p.agent, Intent: p.intent, Goal: p.goal,
		Status: p.status, CreatedAt: p.createdAt, Scope: p.scope, Assumptions: p.assumptions,
		Report: p.report, Tasks: tasks, Risks: risks, Approvals: approvals, Waves: waves,
	}
}
