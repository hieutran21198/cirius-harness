package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"harness-workspace/services/harness/internal/domain"
)

// planReader is a GORM-backed domain.PlanReader bound to a db handle. It reads the six plan
// tables back into a PlanSnapshot and rehydrates the aggregate (which re-validates the DAG, so a
// corrupt or partial read fails loudly).
type planReader struct {
	db *gorm.DB
}

// NewPlanReader builds a domain.PlanReader over db.
func NewPlanReader(db *gorm.DB) domain.PlanReader { return planReader{db: db} }

// FindByID loads the plan with the given id, or ErrPlanNotFound.
func (r planReader) FindByID(ctx context.Context, id domain.PlanID) (domain.Plan, error) {
	var row planRow
	err := r.db.WithContext(ctx).Where("id = ?", string(id)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader.FindByID: %w", err)
	}
	return r.load(ctx, row)
}

// LatestForSession loads the newest plan produced in the session, or ErrPlanNotFound.
func (r planReader) LatestForSession(ctx context.Context, sessionID domain.SessionID) (domain.Plan, error) {
	var rows []planRow
	err := r.db.WithContext(ctx).
		Where("session_id = ?", string(sessionID)).
		Order("created_at DESC, id DESC").Limit(1).Find(&rows).Error
	if err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader.LatestForSession: %w", err)
	}
	if len(rows) == 0 {
		return domain.Plan{}, domain.ErrPlanNotFound
	}
	return r.load(ctx, rows[0])
}

// load reads the children for a plan row and rehydrates the aggregate.
func (r planReader) load(ctx context.Context, row planRow) (domain.Plan, error) {
	db := r.db.WithContext(ctx)
	planID := row.ID

	var taskRows []planTaskRow
	if err := db.Where("plan_id = ?", planID).Find(&taskRows).Error; err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: tasks: %w", err)
	}
	var riskRows []planRiskRow
	if err := db.Where("plan_id = ?", planID).Find(&riskRows).Error; err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: risks: %w", err)
	}
	var approvalRows []planApprovalRow
	if err := db.Where("plan_id = ?", planID).Find(&approvalRows).Error; err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: approvals: %w", err)
	}
	var waveRows []planWaveRow
	if err := db.Where("plan_id = ?", planID).Order("wave_number").Find(&waveRows).Error; err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: waves: %w", err)
	}

	refByTaskID := make(map[string]string, len(taskRows))
	tasks := make([]domain.PlanTaskSnapshot, len(taskRows))
	for i, t := range taskRows {
		refByTaskID[t.ID] = t.Ref
		tasks[i] = domain.PlanTaskSnapshot{
			ID: domain.PlanTaskID(t.ID), Ref: t.Ref, Category: domain.Category(t.Category),
			AssigneeAgent: t.AssigneeAgent, AssigneeLens: t.AssigneeLens, Objective: t.Objective,
			Inputs: fromJSONArray(t.Inputs), ExpectedOutput: t.ExpectedOutput,
			DependsOn: fromJSONArray(t.DependsOn), DoD: fromJSONArray(t.DoD),
			Gate: t.Gate, RiskLevel: t.RiskLevel,
		}
	}
	risks := make([]domain.PlanRiskSnapshot, len(riskRows))
	for i, rr := range riskRows {
		risks[i] = domain.PlanRiskSnapshot{ID: domain.PlanRiskID(rr.ID), Level: rr.Level, Description: rr.Description}
	}
	approvals := make([]domain.PlanApprovalSnapshot, len(approvalRows))
	for i, a := range approvalRows {
		approvals[i] = domain.PlanApprovalSnapshot{
			ID: domain.PlanApprovalID(a.ID), Kind: a.Type, RequiredBefore: a.RequiredBefore,
			Reason: a.Reason, Question: a.Question,
		}
	}

	waves := make([]domain.PlanWaveSnapshot, len(waveRows))
	for i, w := range waveRows {
		var joins []planWaveTaskRow
		if err := db.Where("plan_wave_id = ?", w.ID).Find(&joins).Error; err != nil {
			return domain.Plan{}, fmt.Errorf("repo.planReader: wave tasks: %w", err)
		}
		refs := make([]string, 0, len(joins))
		for _, j := range joins {
			if ref, ok := refByTaskID[j.PlanTaskID]; ok {
				refs = append(refs, ref)
			}
		}
		waves[i] = domain.PlanWaveSnapshot{ID: domain.PlanWaveID(w.ID), Number: w.WaveNumber, TaskRefs: refs}
	}

	var scope domain.Scope
	if err := json.Unmarshal([]byte(row.Scope), &scope); err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: scope: %w", err)
	}
	var report domain.Report
	if err := json.Unmarshal([]byte(row.Report), &report); err != nil {
		return domain.Plan{}, fmt.Errorf("repo.planReader: report: %w", err)
	}

	var sessionID domain.SessionID
	if row.SessionID != nil {
		sessionID = domain.SessionID(*row.SessionID)
	}

	snap := domain.PlanSnapshot{
		ID: domain.PlanID(row.ID), SessionID: sessionID, Agent: row.Agent, Intent: row.Intent,
		Goal: row.Goal, Status: domain.PlanStatus(row.Status), CreatedAt: row.CreatedAt,
		Scope: scope, Assumptions: fromJSONArray(row.Assumptions), Report: report,
		Tasks: tasks, Risks: risks, Approvals: approvals, Waves: waves,
	}
	return domain.RehydratePlan(snap)
}

// fromJSONArray decodes a JSON string-array column back to a []string; "", "null", and "[]" all
// decode to nil/empty (the inverse of jsonArray in plan.go).
func fromJSONArray(s string) []string {
	if s == "" {
		return nil
	}
	var ss []string
	if err := json.Unmarshal([]byte(s), &ss); err != nil {
		return nil
	}
	return ss
}

// staticcheck: ensure planReader satisfies the domain port.
var _ domain.PlanReader = planReader{}
