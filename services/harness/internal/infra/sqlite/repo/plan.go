package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"harness-workspace/services/harness/internal/domain"
)

// planRow maps the `plans` table. The small leaf structures (scope, assumptions, report) are
// stored as JSON; the DAG, risks, approvals, and waves are child tables. SessionID is a pointer
// so a session-less plan is stored as NULL.
type planRow struct {
	ID          string    `gorm:"column:id;primaryKey"`
	SessionID   *string   `gorm:"column:session_id"`
	Agent       string    `gorm:"column:agent"`
	Intent      string    `gorm:"column:intent"`
	Goal        string    `gorm:"column:goal"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	Scope       string    `gorm:"column:scope"`
	Assumptions string    `gorm:"column:assumptions"`
	Report      string    `gorm:"column:report"`
}

func (planRow) TableName() string { return "plans" }

// planTaskRow maps the `plan_tasks` table. inputs/depends_on/dod are JSON string arrays.
type planTaskRow struct {
	ID             string `gorm:"column:id;primaryKey"`
	PlanID         string `gorm:"column:plan_id"`
	Ref            string `gorm:"column:ref"`
	Category       string `gorm:"column:category"`
	AssigneeAgent  string `gorm:"column:assignee_agent"`
	AssigneeLens   string `gorm:"column:assignee_lens"`
	Objective      string `gorm:"column:objective"`
	ExpectedOutput string `gorm:"column:expected_output"`
	Gate           string `gorm:"column:gate"`
	RiskLevel      string `gorm:"column:risk_level"`
	Inputs         string `gorm:"column:inputs"`
	DependsOn      string `gorm:"column:depends_on"`
	DoD            string `gorm:"column:dod"`
}

func (planTaskRow) TableName() string { return "plan_tasks" }

// planRiskRow maps the `plan_risks` table.
type planRiskRow struct {
	ID          string `gorm:"column:id;primaryKey"`
	PlanID      string `gorm:"column:plan_id"`
	Level       string `gorm:"column:level"`
	Description string `gorm:"column:description"`
}

func (planRiskRow) TableName() string { return "plan_risks" }

// planApprovalRow maps the `plan_approvals` table.
type planApprovalRow struct {
	ID             string `gorm:"column:id;primaryKey"`
	PlanID         string `gorm:"column:plan_id"`
	Type           string `gorm:"column:type"`
	RequiredBefore string `gorm:"column:required_before"`
	Reason         string `gorm:"column:reason"`
	Question       string `gorm:"column:question"`
}

func (planApprovalRow) TableName() string { return "plan_approvals" }

// planWaveRow maps the `plan_waves` table.
type planWaveRow struct {
	ID         string `gorm:"column:id;primaryKey"`
	PlanID     string `gorm:"column:plan_id"`
	WaveNumber int    `gorm:"column:wave_number"`
}

func (planWaveRow) TableName() string { return "plan_waves" }

// planWaveTaskRow maps the `plan_wave_tasks` join (wave→task membership).
type planWaveTaskRow struct {
	PlanWaveID string `gorm:"column:plan_wave_id;primaryKey"`
	PlanTaskID string `gorm:"column:plan_task_id;primaryKey"`
}

func (planWaveTaskRow) TableName() string { return "plan_wave_tasks" }

// planWriter is a GORM-backed domain.PlanWriter bound to a db handle.
type planWriter struct {
	db *gorm.DB
}

// NewPlanWriter builds a domain.PlanWriter over db.
func NewPlanWriter(db *gorm.DB) domain.PlanWriter { return planWriter{db: db} }

// Save inserts the plan row and all its children, idempotent on the plan id: if the plan
// already exists the insert is a no-op and the children are left untouched.
func (w planWriter) Save(ctx context.Context, p domain.Plan) error {
	snap := p.Snapshot()
	db := w.db.WithContext(ctx)

	scope, err := toJSON(snap.Scope)
	if err != nil {
		return fmt.Errorf("repo.planWriter.Save: marshal scope: %w", err)
	}
	report, err := toJSON(snap.Report)
	if err != nil {
		return fmt.Errorf("repo.planWriter.Save: marshal report: %w", err)
	}

	var sessionID *string
	if snap.SessionID != "" {
		s := string(snap.SessionID)
		sessionID = &s
	}
	row := planRow{
		ID:          string(snap.ID),
		SessionID:   sessionID,
		Agent:       snap.Agent,
		Intent:      snap.Intent,
		Goal:        snap.Goal,
		Status:      string(snap.Status),
		CreatedAt:   snap.CreatedAt,
		Scope:       scope,
		Assumptions: jsonArray(snap.Assumptions),
		Report:      report,
	}
	res := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&row)
	if res.Error != nil {
		return fmt.Errorf("repo.planWriter.Save: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil // already persisted — idempotent no-op
	}

	taskIDByRef := make(map[string]string, len(snap.Tasks))
	if len(snap.Tasks) > 0 {
		rows := make([]planTaskRow, len(snap.Tasks))
		for i, t := range snap.Tasks {
			rows[i] = planTaskRow{
				ID:             string(t.ID),
				PlanID:         string(snap.ID),
				Ref:            t.Ref,
				Category:       string(t.Category),
				AssigneeAgent:  t.AssigneeAgent,
				AssigneeLens:   t.AssigneeLens,
				Objective:      t.Objective,
				ExpectedOutput: t.ExpectedOutput,
				Gate:           t.Gate,
				RiskLevel:      t.RiskLevel,
				Inputs:         jsonArray(t.Inputs),
				DependsOn:      jsonArray(t.DependsOn),
				DoD:            jsonArray(t.DoD),
			}
			taskIDByRef[t.Ref] = string(t.ID)
		}
		if err := db.Create(&rows).Error; err != nil {
			return fmt.Errorf("repo.planWriter.Save: tasks: %w", err)
		}
	}

	if len(snap.Risks) > 0 {
		rows := make([]planRiskRow, len(snap.Risks))
		for i, r := range snap.Risks {
			rows[i] = planRiskRow{ID: string(r.ID), PlanID: string(snap.ID), Level: r.Level, Description: r.Description}
		}
		if err := db.Create(&rows).Error; err != nil {
			return fmt.Errorf("repo.planWriter.Save: risks: %w", err)
		}
	}

	if len(snap.Approvals) > 0 {
		rows := make([]planApprovalRow, len(snap.Approvals))
		for i, a := range snap.Approvals {
			rows[i] = planApprovalRow{
				ID: string(a.ID), PlanID: string(snap.ID), Type: a.Kind,
				RequiredBefore: a.RequiredBefore, Reason: a.Reason, Question: a.Question,
			}
		}
		if err := db.Create(&rows).Error; err != nil {
			return fmt.Errorf("repo.planWriter.Save: approvals: %w", err)
		}
	}

	for _, wv := range snap.Waves {
		waveRow := planWaveRow{ID: string(wv.ID), PlanID: string(snap.ID), WaveNumber: wv.Number}
		if err := db.Create(&waveRow).Error; err != nil {
			return fmt.Errorf("repo.planWriter.Save: wave: %w", err)
		}
		var joins []planWaveTaskRow
		for _, ref := range wv.TaskRefs {
			taskID, ok := taskIDByRef[ref]
			if !ok {
				continue // ref integrity is validated in the domain; skip defensively
			}
			joins = append(joins, planWaveTaskRow{PlanWaveID: string(wv.ID), PlanTaskID: taskID})
		}
		if len(joins) > 0 {
			if err := db.Create(&joins).Error; err != nil {
				return fmt.Errorf("repo.planWriter.Save: wave tasks: %w", err)
			}
		}
	}
	return nil
}

// toJSON marshals a value to its JSON string form.
func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// jsonArray marshals a string slice to a JSON array, normalising nil/empty to "[]" so the
// column never holds "null".
func jsonArray(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss) // []string never fails to marshal
	return string(b)
}

// staticcheck: ensure planWriter satisfies the domain port.
var _ domain.PlanWriter = planWriter{}
